package relay

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/constant"
	"github.com/NookMux/NookMux/internal/dto"
	"github.com/NookMux/NookMux/internal/relay/channel/openai"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/internal/service"
	"github.com/NookMux/NookMux/internal/types"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
)

func BenchmarkOpenAIStreamRelayConcurrent128(b *testing.B) {
	gin.SetMode(gin.TestMode)
	service.InitHttpClient()

	const (
		concurrency     = 128
		framesPerStream = 100
	)

	releaseUpstream := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintln(w, `data: {"id":"chatcmpl-bench","object":"chat.completion.chunk","created":1,"model":"gpt-bench","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`)
		if flusher != nil {
			flusher.Flush()
		}
		<-releaseUpstream
		for i := 1; i < framesPerStream; i++ {
			fmt.Fprintln(w, `data: {"id":"chatcmpl-bench","object":"chat.completion.chunk","created":1,"model":"gpt-bench","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`)
		}
		fmt.Fprintln(w, `data: {"id":"chatcmpl-bench","object":"chat.completion.chunk","created":1,"model":"gpt-bench","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":8,"completion_tokens":100,"total_tokens":108}}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer upstream.Close()

	var peakHeapTotal uint64
	var retainedHeapTotal uint64
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		releaseUpstream = make(chan struct{})
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

		var upstreamStarted atomic.Int64
		var done sync.WaitGroup
		start := make(chan struct{})
		samplerDone := make(chan struct{})
		done.Add(concurrency)

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
				request := &dto.GeneralOpenAIRequest{
					Model: "gpt-bench",
					Messages: []dto.Message{
						{Role: "user", Content: "hello"},
					},
					Stream: true,
					StreamOptions: &dto.StreamOptions{
						IncludeUsage: true,
					},
				}
				body, err := jsonx.Marshal(request)
				if err != nil {
					b.Errorf("marshal request: %v", err)
					return
				}
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				c.Set(string(constant.ContextKeyChannelType), constant.ChannelTypeOpenAI)
				c.Set(string(constant.ContextKeyChannelId), 1)
				c.Set(string(constant.ContextKeyChannelBaseUrl), upstream.URL)
				c.Set(string(constant.ContextKeyChannelKey), "test-key")
				c.Set(string(constant.ContextKeyOriginalModel), "gpt-bench")
				c.Set(string(constant.ContextKeyChannelSetting), dto.ChannelSettings{})
				c.Set(string(constant.ContextKeyChannelOtherSetting), dto.ChannelOtherSettings{})

				info, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAI, request, nil)
				if err != nil {
					b.Errorf("gen relay info: %v", err)
					return
				}
				info.RelayMode = relayconstant.RelayModeChatCompletions
				info.RequestURLPath = "/v1/chat/completions"
				info.IsStream = true
				info.SetEstimatePromptTokens(8)
				info.InitChannelMeta(c)

				adaptor := &openai.Adaptor{}
				adaptor.Init(info)
				converted, err := adaptor.ConvertOpenAIRequest(c, info, request)
				if err != nil {
					b.Errorf("convert request: %v", err)
					return
				}
				jsonData, err := jsonx.Marshal(converted)
				if err != nil {
					b.Errorf("marshal converted request: %v", err)
					return
				}
				respAny, err := adaptor.DoRequest(c, info, bytes.NewReader(jsonData))
				if err != nil {
					b.Errorf("do request: %v", err)
					return
				}
				upstreamStarted.Add(1)
				resp := respAny.(*http.Response)
				if resp.StatusCode != http.StatusOK {
					b.Errorf("status = %d, want 200", resp.StatusCode)
					return
				}
				usage, apiErr := adaptor.DoResponse(c, resp, info)
				if apiErr != nil {
					b.Errorf("do response: %v", apiErr)
					return
				}
				u := usage.(*dto.Usage)
				if u.TotalTokens == 0 {
					b.Errorf("usage total tokens = 0")
					return
				}
				if !strings.Contains(recorder.Body.String(), "[DONE]") {
					b.Errorf("stream response missing DONE")
					return
				}
			}()
		}

		close(start)
		for upstreamStarted.Load() < concurrency {
			runtime.Gosched()
		}
		recordPeak()
		close(releaseUpstream)
		done.Wait()
		close(samplerDone)

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
