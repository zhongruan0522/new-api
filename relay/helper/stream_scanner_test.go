package helper

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/NookMux/NookMux/constant"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/setting/operation_setting"
)

func streamScannerTestContext(t *testing.T) (*gin.Context, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	cleanup := func() {
		constant.StreamingTimeout = oldStreamingTimeout
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	return c, cleanup
}

func TestStreamScannerHandlerTrimsAndSkipsEmptyDataFrames(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"data:   ",
			"data:\r",
			"data:  {\"ok\":true}\r",
			"data: [DONE]",
			"",
		}, "\n"))),
	}

	var got []string
	StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
		got = append(got, data)
		return true
	})

	want := []string{`{"ok":true}`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("data frames = %#v, want %#v", got, want)
	}
}

func TestStreamScannerHandlerSkipsNonDataLines(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	var b strings.Builder
	b.WriteString(": comment line\n")
	b.WriteString("event: message\n")
	b.WriteString("id: 12345\n")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&b, "data: payload_%d\n", i)
		b.WriteString(": interleaved comment\n")
	}
	b.WriteString("data: [DONE]\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(b.String())),
	}

	var count int
	StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
		count++
		return true
	})
	if count != 100 {
		t.Fatalf("handled %d data frames, want 100", count)
	}
}

func TestStreamScannerHandlerDataWithExtraSpaces(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("data:   {\"trimmed\":true}  \r\ndata: [DONE]\n")),
	}

	var got string
	StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
		got = data
		return true
	})
	if got != `{"trimmed":true}` {
		t.Fatalf("payload = %q, want %q", got, `{"trimmed":true}`)
	}
}

func TestStreamScannerHandlerDoneStopsScanner(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	var b strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&b, "data: {\"n\":%d}\n", i)
	}
	b.WriteString("data: [DONE]\n")
	b.WriteString("data: should_not_appear\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(b.String())),
	}

	var count int
	StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
		count++
		return true
	})
	if count != 50 {
		t.Fatalf("handled %d frames, want 50 (nothing after [DONE])", count)
	}
}

func buildSSEBody(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "data: {\"id\":%d,\"choices\":[{\"delta\":{\"content\":\"token_%d\"}}]}\n", i, i)
	}
	b.WriteString("data: [DONE]\n")
	return b.String()
}

func TestStreamScannerHandlerNilInputs(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)
	info := &relaycommon.RelayInfo{}

	StreamScannerHandler(c, nil, info, func(data string) bool { return true })
	StreamScannerHandler(c, &http.Response{Body: io.NopCloser(strings.NewReader(""))}, info, nil)
}

func TestStreamScannerHandlerEmptyBody(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("")),
	}

	var called bool
	StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
		called = true
		return true
	})
	if called {
		t.Fatal("handler should not run on empty upstream body")
	}
}

func TestStreamScannerHandler1000Chunks(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	const numChunks = 1000
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(buildSSEBody(numChunks))),
	}
	info := &relaycommon.RelayInfo{}

	var count int
	StreamScannerHandler(c, resp, info, func(data string) bool {
		count++
		return true
	})
	if count != numChunks {
		t.Fatalf("handled %d chunks, want %d", count, numChunks)
	}
	if info.ReceivedResponseCount != numChunks {
		t.Fatalf("ReceivedResponseCount = %d, want %d", info.ReceivedResponseCount, numChunks)
	}
}

func TestStreamScannerHandlerOrderPreserved(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	const numChunks = 500
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(buildSSEBody(numChunks))),
	}

	var mu sync.Mutex
	received := make([]string, 0, numChunks)
	StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
		mu.Lock()
		received = append(received, data)
		mu.Unlock()
		return true
	})

	if len(received) != numChunks {
		t.Fatalf("got %d chunks, want %d", len(received), numChunks)
	}
	for i := 0; i < numChunks; i++ {
		want := fmt.Sprintf(`{"id":%d,"choices":[{"delta":{"content":"token_%d"}}]}`, i, i)
		if received[i] != want {
			t.Fatalf("chunk %d = %q, want %q", i, received[i], want)
		}
	}
}

func TestStreamScannerHandlerStopStopsStream(t *testing.T) {
	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(buildSSEBody(200))),
	}

	const stopAt = 50
	var count int
	StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
		count++
		return count < stopAt
	})
	if count != stopAt {
		t.Fatalf("handled %d chunks before stop, want %d", count, stopAt)
	}
}

func TestStreamScannerHandlerPingSentDuringSlowUpstream(t *testing.T) {
	setting := operation_setting.GetGeneralSetting()
	oldEnabled := setting.PingIntervalEnabled
	oldSeconds := setting.PingIntervalSeconds
	setting.PingIntervalEnabled = true
	setting.PingIntervalSeconds = 1
	t.Cleanup(func() {
		setting.PingIntervalEnabled = oldEnabled
		setting.PingIntervalSeconds = oldSeconds
	})

	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for i := 0; i < 4; i++ {
			fmt.Fprintf(pw, "data: chunk_%d\n", i)
			time.Sleep(400 * time.Millisecond)
		}
		fmt.Fprint(pw, "data: [DONE]\n")
	}()

	recorder := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{Body: pr}
	info := &relaycommon.RelayInfo{}

	var count int
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string) bool {
			count++
			return true
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("timed out waiting for slow stream")
	}

	if count != 4 {
		t.Fatalf("handled %d chunks, want 4", count)
	}
	if pingCount := strings.Count(recorder.Body.String(), ": PING"); pingCount < 1 {
		t.Fatalf("expected at least one SSE ping, body=%q", recorder.Body.String())
	}
}

func TestStreamScannerHandlerPingDisabledByRelayInfo(t *testing.T) {
	setting := operation_setting.GetGeneralSetting()
	oldEnabled := setting.PingIntervalEnabled
	oldSeconds := setting.PingIntervalSeconds
	setting.PingIntervalEnabled = true
	setting.PingIntervalSeconds = 1
	t.Cleanup(func() {
		setting.PingIntervalEnabled = oldEnabled
		setting.PingIntervalSeconds = oldSeconds
	})

	c, cleanup := streamScannerTestContext(t)
	t.Cleanup(cleanup)

	recorder := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(buildSSEBody(5))),
	}
	info := &relaycommon.RelayInfo{DisablePing: true}

	StreamScannerHandler(c, resp, info, func(data string) bool { return true })

	if pingCount := strings.Count(recorder.Body.String(), ": PING"); pingCount != 0 {
		t.Fatalf("expected no pings when DisablePing=true, got %d", pingCount)
	}
}

func TestGetScannerBufferSizeUsesConservativeDefault(t *testing.T) {
	oldMaxBufferMB := constant.StreamScannerMaxBufferMB
	constant.StreamScannerMaxBufferMB = 0
	t.Cleanup(func() {
		constant.StreamScannerMaxBufferMB = oldMaxBufferMB
	})

	if got := getScannerBufferSize(); got != 8<<20 {
		t.Fatalf("scanner buffer size = %d, want %d", got, 8<<20)
	}
}

func BenchmarkStreamScannerHandler(b *testing.B) {
	gin.SetMode(gin.TestMode)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	b.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})

	frame := `data: {"id":"chatcmpl","choices":[{"delta":{"content":"hello"}}]}`
	body := strings.Repeat(frame+"\n", 200) + "data: [DONE]\n"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
		}
		count := 0
		StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
			count++
			return data != ""
		})
		if count != 200 {
			b.Fatalf("handled %d frames, want 200", count)
		}
	}
}

func BenchmarkStreamScannerHandlerConcurrent128(b *testing.B) {
	gin.SetMode(gin.TestMode)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	b.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})

	const (
		concurrency     = 128
		framesPerStream = 200
	)
	frame := `data: {"id":"chatcmpl","choices":[{"delta":{"content":"hello"}}]}`
	body := strings.Repeat(frame+"\n", framesPerStream) + "data: [DONE]\n"

	var peakHeapTotal uint64
	var retainedHeapTotal uint64
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		runtime.GC()
		var before runtime.MemStats
		runtime.ReadMemStats(&before)

		var peakHeap atomic.Uint64
		recordPeak := func() {
			var current runtime.MemStats
			runtime.ReadMemStats(&current)
			if current.HeapAlloc <= before.HeapAlloc {
				return
			}
			delta := current.HeapAlloc - before.HeapAlloc
			for {
				old := peakHeap.Load()
				if delta <= old || peakHeap.CompareAndSwap(old, delta) {
					return
				}
			}
		}

		start := make(chan struct{})
		releaseFirstFrames := make(chan struct{})
		var firstFrames sync.WaitGroup
		var done sync.WaitGroup
		firstFrames.Add(concurrency)
		done.Add(concurrency)
		var totalFrames atomic.Int64
		samplerDone := make(chan struct{})

		go func() {
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					recordPeak()
				case <-samplerDone:
					recordPeak()
					return
				}
			}
		}()

		for j := 0; j < concurrency; j++ {
			go func() {
				defer done.Done()
				<-start

				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
				}
				seenFirstFrame := false
				StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
					if !seenFirstFrame {
						seenFirstFrame = true
						firstFrames.Done()
						<-releaseFirstFrames
					}
					totalFrames.Add(1)
					return data != ""
				})
			}()
		}

		close(start)
		firstFrames.Wait()
		recordPeak()
		close(releaseFirstFrames)
		done.Wait()
		close(samplerDone)

		if got, want := totalFrames.Load(), int64(concurrency*framesPerStream); got != want {
			b.Fatalf("handled %d frames, want %d", got, want)
		}

		runtime.GC()
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		if after.HeapAlloc > before.HeapAlloc {
			retainedHeapTotal += after.HeapAlloc - before.HeapAlloc
		}
		peakHeapTotal += peakHeap.Load()
	}

	b.ReportMetric(float64(peakHeapTotal)/float64(b.N)/(1024*1024), "peak_heap_mb/op")
	b.ReportMetric(float64(retainedHeapTotal)/float64(b.N)/(1024*1024), "retained_heap_mb/op")
}
