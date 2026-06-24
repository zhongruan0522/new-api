package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/types"
)

func TestStreamResponseGeminiChat2OpenAIPreservesThoughtAndText(t *testing.T) {
	stop := "STOP"
	thought := dto.GeminiPart{Text: "thinking", Thought: true}
	thought.SetThoughtSignature("sig_123")
	resp, isStop := streamResponseGeminiChat2OpenAI(&dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{
			Index:        0,
			FinishReason: &stop,
			Content: dto.GeminiChatContent{Parts: []dto.GeminiPart{thought, {
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
		RelayFormat:       types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "real-model",
		},
	}
	finish := "STOP"
	event, err := common.Marshal(dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{
			Index:        0,
			FinishReason: &finish,
			Content: dto.GeminiChatContent{Parts: []dto.GeminiPart{{
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
