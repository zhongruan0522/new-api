package helper

import (
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
	"github.com/zhongruan0522/new-api/constant"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func TestStreamScannerHandlerTrimsAndSkipsEmptyDataFrames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

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
