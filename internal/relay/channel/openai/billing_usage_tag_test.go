package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/billing"
	"github.com/NookMux/NookMux/internal/httpapi"

	"github.com/gin-gonic/gin"
)

// TestOpenaiSTTHandlerTagsUsageSourceAndSkipsEstimate 验证 STT 解析点：
// 上游返回真实 usage 时显式标识 OpenAI Chat 来源并可归一化落列；
// 上游无 usage 的本地估算分支打 local 标志、billing_details 不落列。
func TestOpenaiSTTHandlerTagsUsageSourceAndSkipsEstimate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("real upstream usage", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", nil)
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o-transcribe"},
		}
		body := `{"text":"hello","usage":{"type":"tokens","input_tokens":120,"input_token_details":{"cached_tokens":40,"text_tokens":80,"audio_tokens":40},"output_tokens":30,"total_tokens":150}}`
		resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}

		apiErr, usage := OpenaiSTTHandler(c, resp, info, "json")
		if apiErr != nil {
			t.Fatalf("OpenaiSTTHandler error: %v", apiErr)
		}
		if usage.PromptTokens != 120 || usage.CompletionTokens != 30 {
			t.Fatalf("usage = %+v, want real upstream tokens", usage)
		}
		if info.UsageSource != relayconstant.UsageSourceOpenAIChat {
			t.Fatalf("usage source = %q, want %q", info.UsageSource, relayconstant.UsageSourceOpenAIChat)
		}
		// STT 的 input_token_details（单数）当前不映射进 shared.Usage（预存在
		// 口径，阶段 1 不改），JSON 如实记录空拆分，不虚构明细。
		got := billing.BuildBillingDetailsForLog(c, info, usage)
		want := `{"schema_version":1,"tokens":{"input":{},"output":{},"cache":{}}}`
		if got != want {
			t.Fatalf("billing_details = %s, want %s", got, want)
		}
	})

	t.Run("estimate fallback flagged local", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", nil)
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "whisper-1"},
		}
		info.SetEstimatePromptTokens(25)
		resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"text":"hi"}`))}

		apiErr, usage := OpenaiSTTHandler(c, resp, info, "json")
		if apiErr != nil {
			t.Fatalf("OpenaiSTTHandler error: %v", apiErr)
		}
		if usage.PromptTokens != 25 {
			t.Fatalf("usage = %+v, want estimate prompt tokens", usage)
		}
		if !httpapi.GetContextKeyBool(c, common.ContextKeyLocalCountTokens) {
			t.Fatalf("estimate fallback must set ContextKeyLocalCountTokens")
		}
	})
}

// TestOpenaiRealtimeHandlerTagsUsageSource 验证 Realtime 解析点按 Responses
// 同族规则标识来源。
func TestOpenaiRealtimeHandlerTagsUsageSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
	info := &relaycommon.RelayInfo{
		RelayFormat: relayconstant.RelayFormatOpenAIRealtime,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o-realtime-preview"},
	}

	// OpenaiRealtimeHandler 需要建立 websocket 连接，这里无法直接驱动完整流程；
	// 但打标必须发生在函数入口（任何提前返回也要有来源标识），
	// 因此只验证入口打标行为：调用后立即检查（连接失败也会先打标）。
	_, _ = OpenaiRealtimeHandler(c, info)
	if info.UsageSource != relayconstant.UsageSourceOpenAIResponses {
		t.Fatalf("usage source = %q, want %q", info.UsageSource, relayconstant.UsageSourceOpenAIResponses)
	}
}
