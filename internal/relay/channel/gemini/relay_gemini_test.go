package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"

	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
)

func TestStreamResponseGeminiChat2OpenAIPreservesThoughtAndText(t *testing.T) {
	stop := "STOP"
	thought := shared.GeminiPart{Text: "thinking", Thought: true}
	thought.SetThoughtSignature("sig_123")
	resp, isStop := streamResponseGeminiChat2OpenAI(&shared.GeminiChatResponse{
		Candidates: []shared.GeminiChatCandidate{{
			Index:        0,
			FinishReason: &stop,
			Content: shared.GeminiChatContent{Parts: []shared.GeminiPart{thought, {
				Text: "answer",
			}}},
		}},
	})

	if !isStop {
		t.Fatal("isStop = false, want true")
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("got %d choices, want 1", len(resp.Choices))
	}
	choice := resp.Choices[0]
	if choice.Delta.GetReasoningContent() != "thinking" {
		t.Fatalf("reasoning content = %q, want thinking", choice.Delta.GetReasoningContent())
	}
	if choice.Delta.ReasoningSignature == nil || *choice.Delta.ReasoningSignature != "sig_123" {
		t.Fatalf("reasoning signature = %v, want sig_123", choice.Delta.ReasoningSignature)
	}
	if choice.Delta.GetContentString() != "answer" {
		t.Fatalf("content = %q, want answer", choice.Delta.GetContentString())
	}
	if choice.FinishReason != nil {
		t.Fatalf("finish reason = %v, want nil because STOP is emitted as a separate stop chunk", choice.FinishReason)
	}
}

func TestGeminiChatStreamHandlerMasksResponseModel(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName:   "alias-model",
		ResponseModelName: "alias-model",
		RelayFormat:       relayconstant.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "real-model",
		},
	}
	finish := "STOP"
	event, err := jsonx.Marshal(shared.GeminiChatResponse{
		Candidates: []shared.GeminiChatCandidate{{
			Index:        0,
			FinishReason: &finish,
			Content: shared.GeminiChatContent{Parts: []shared.GeminiPart{{
				Text: "answer",
			}}},
		}},
	})
	if err != nil {
		t.Fatalf("marshal gemini event: %v", err)
	}
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: " + string(event) + "\n\n"))}

	if _, apiErr := GeminiChatStreamHandler(c, info, resp); apiErr != nil {
		t.Fatalf("GeminiChatStreamHandler error: %v", apiErr)
	}
	out := w.Body.String()
	if strings.Contains(out, `"model":"real-model"`) {
		t.Fatalf("gemini stream output leaked upstream model: %s", out)
	}
	if !strings.Contains(out, `"model":"alias-model"`) {
		t.Fatalf("gemini stream output missing alias model: %s", out)
	}
}

func TestGeminiChatStreamHandlerClaudeKeepsMultipleToolCallsDistinct(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName:   "claude-alias",
		ResponseModelName: "claude-alias",
		RelayFormat:       relayconstant.RelayFormatClaude,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-2.5-pro",
		},
	}

	firstEvent := marshalGeminiStreamEvent(t, shared.GeminiChatResponse{
		Candidates: []shared.GeminiChatCandidate{{
			Index: 0,
			Content: shared.GeminiChatContent{Parts: []shared.GeminiPart{{
				FunctionCall: &shared.FunctionCall{FunctionName: "weather", Arguments: map[string]any{"city": "Shanghai"}},
			}}},
		}},
	})
	secondEvent := marshalGeminiStreamEvent(t, shared.GeminiChatResponse{
		Candidates: []shared.GeminiChatCandidate{{
			Index: 0,
			Content: shared.GeminiChatContent{Parts: []shared.GeminiPart{{
				FunctionCall: &shared.FunctionCall{FunctionName: "calendar", Arguments: map[string]any{"day": "today"}},
			}}},
		}},
		UsageMetadata: shared.GeminiUsageMetadata{
			PromptTokenCount:        2,
			ToolUsePromptTokenCount: 3,
			CandidatesTokenCount:    5,
			TotalTokenCount:         10,
		},
	})
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(firstEvent + secondEvent))}

	if _, apiErr := GeminiChatStreamHandler(c, info, resp); apiErr != nil {
		t.Fatalf("GeminiChatStreamHandler error: %v", apiErr)
	}
	out := w.Body.String()
	if !strings.Contains(out, `"type":"tool_use"`) || !strings.Contains(out, `"name":"weather"`) || !strings.Contains(out, `"name":"calendar"`) {
		t.Fatalf("claude stream output missing tool_use blocks: %s", out)
	}
	if !strings.Contains(out, `"index":0`) || !strings.Contains(out, `"index":1`) {
		t.Fatalf("claude stream output did not assign distinct tool indexes: %s", out)
	}
	if strings.Contains(out, `"finish_reason":"tool_calls"`) {
		t.Fatalf("claude stream output leaked OpenAI finish_reason: %s", out)
	}
	if !strings.Contains(out, `"stop_reason":"tool_use"`) || !strings.Contains(out, `"type":"message_stop"`) {
		t.Fatalf("claude stream output missing final tool_use stop: %s", out)
	}
	if !strings.Contains(out, `"input_tokens":2`) {
		t.Fatalf("claude stream output should preserve official prompt total excluding tool-use audit: %s", out)
	}
}

func marshalGeminiStreamEvent(t *testing.T, response shared.GeminiChatResponse) string {
	t.Helper()
	payload, err := jsonx.Marshal(response)
	if err != nil {
		t.Fatalf("marshal gemini event: %v", err)
	}
	return "data: " + string(payload) + "\n\n"
}
