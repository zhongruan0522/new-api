package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"

	"github.com/NookMux/NookMux/internal/domain/billing"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/pkg/jsonx"

	"github.com/gin-gonic/gin"
)

// TestGeminiStreamHandlerTagsUsageSourceAndStashesMetadata 验证 Gemini 流式
// 解析点显式标识 usage 来源，并保留原始 usageMetadata（toolUsePromptTokenCount
// 被并入转换结果后，归一化需要原始拆分才能"只审计不进计价输入"）。
func TestGeminiStreamHandlerTagsUsageSourceAndStashesMetadata(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		RelayFormat: relayconstant.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-2.5-flash"},
	}
	event, err := jsonx.Marshal(shared.GeminiChatResponse{
		UsageMetadata: shared.GeminiUsageMetadata{
			PromptTokenCount:        151,
			ToolUsePromptTokenCount: 18329,
			CandidatesTokenCount:    1089,
			TotalTokenCount:         2269,
			PromptTokensDetails: []shared.GeminiPromptTokensDetails{
				{Modality: "TEXT", TokenCount: 151},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal gemini event: %v", err)
	}
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: " + string(event) + "\n\n"))}

	usage, apiErr := GeminiChatStreamHandler(c, info, resp)
	if apiErr != nil {
		t.Fatalf("GeminiChatStreamHandler error: %v", apiErr)
	}
	if info.UsageSource != relayconstant.UsageSourceGemini {
		t.Fatalf("usage source = %q, want %q", info.UsageSource, relayconstant.UsageSourceGemini)
	}
	if info.UsageGeminiMetadata == nil || info.UsageGeminiMetadata.PromptTokenCount != 151 ||
		info.UsageGeminiMetadata.ToolUsePromptTokenCount != 18329 {
		t.Fatalf("usage gemini metadata = %+v, want raw counts preserved", info.UsageGeminiMetadata)
	}

	// toolUsePromptTokenCount 只审计：官方输入总量不包含它。
	bu, warnings, err := billing.BuildBillingUsage(relayconstant.UsageSourceGemini, usage, info.UsageGeminiMetadata)
	if err != nil {
		t.Fatalf("BuildBillingUsage() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if bu.PromptAggregateTokens != 151 || bu.InputTokens() != 151 {
		t.Fatalf("prompt aggregate/input = %d/%d, want 151/151", bu.PromptAggregateTokens, bu.InputTokens())
	}
	if bu.ToolUsePromptTokens == nil || *bu.ToolUsePromptTokens != 18329 {
		t.Fatalf("tool use audit = %v, want 18329", bu.ToolUsePromptTokens)
	}
}

// TestGeminiChatHandlerTagsUsageSourceAndStashesMetadata 验证非流式 Gemini
// 解析点（OpenAI 格式请求 → Gemini 上游）。
func TestGeminiChatHandlerTagsUsageSourceAndStashesMetadata(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		RelayFormat: relayconstant.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-2.5-flash"},
	}
	finish := "STOP"
	body, err := jsonx.Marshal(shared.GeminiChatResponse{
		Candidates: []shared.GeminiChatCandidate{{
			Index:        0,
			FinishReason: &finish,
			Content:      shared.GeminiChatContent{Parts: []shared.GeminiPart{{Text: "answer"}}},
		}},
		UsageMetadata: shared.GeminiUsageMetadata{
			PromptTokenCount:        40,
			ToolUsePromptTokenCount: 60,
			CandidatesTokenCount:    8,
			TotalTokenCount:         48,
		},
	})
	if err != nil {
		t.Fatalf("marshal gemini response: %v", err)
	}
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(body)))}

	usage, apiErr := GeminiChatHandler(c, info, resp)
	if apiErr != nil {
		t.Fatalf("GeminiChatHandler error: %v", apiErr)
	}
	if info.UsageSource != relayconstant.UsageSourceGemini {
		t.Fatalf("usage source = %q, want %q", info.UsageSource, relayconstant.UsageSourceGemini)
	}
	if info.UsageGeminiMetadata == nil || info.UsageGeminiMetadata.ToolUsePromptTokenCount != 60 {
		t.Fatalf("usage gemini metadata = %+v, want raw tool use count", info.UsageGeminiMetadata)
	}
	bu, _, err := billing.BuildBillingUsage(relayconstant.UsageSourceGemini, usage, info.UsageGeminiMetadata)
	if err != nil {
		t.Fatalf("BuildBillingUsage() error = %v", err)
	}
	if bu.PromptAggregateTokens != 40 {
		t.Fatalf("prompt aggregate = %d, want 40", bu.PromptAggregateTokens)
	}
}
