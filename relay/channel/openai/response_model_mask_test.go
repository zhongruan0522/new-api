package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/dto"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/types"
)

func testRelayInfo(alias, upstream string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName:   alias,
		ResponseModelName: alias,
		RelayFormat:       types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: upstream,
		},
	}
}

func TestOpenaiHandlerMasksResponseModel(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := testRelayInfo("alias-model", "real-model")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"id":"chatcmpl_1",
			"object":"chat.completion",
			"created":1710000000,
			"model":"real-model",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`)),
	}

	if _, apiErr := OpenaiHandler(c, info, resp); apiErr != nil {
		t.Fatalf("OpenaiHandler error: %v", apiErr)
	}
	out := w.Body.String()
	if !strings.Contains(out, `"model":"alias-model"`) || strings.Contains(out, `"model":"real-model"`) {
		t.Fatalf("response model was not masked: %s", out)
	}
}

func TestOpenAIStreamAndFinalUsageMaskResponseModel(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := testRelayInfo("alias-model", "real-model")
	info.ShouldIncludeUsage = true

	data := `{"id":"chatcmpl_1","object":"chat.completion.chunk","created":1710000000,"model":"real-model","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":null}]}`
	if err := sendStreamData(c, info, data, false); err != nil {
		t.Fatalf("sendStreamData error: %v", err)
	}
	HandleFinalResponse(c, info, data, "chatcmpl_1", 1710000000, "real-model", "", &dto.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2}, false)

	out := w.Body.String()
	if strings.Contains(out, `"model":"real-model"`) {
		t.Fatalf("stream output leaked upstream model: %s", out)
	}
	if count := strings.Count(out, `"model":"alias-model"`); count < 2 {
		t.Fatalf("stream output alias model count = %d, want at least 2: %s", count, out)
	}
}

func TestResponsesHandlerAndStreamMaskResponseModel(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	info := testRelayInfo("alias-model", "real-model")

	respBody, err := common.Marshal(dto.OpenAIResponsesResponse{
		ID:        "resp_1",
		Object:    "response",
		CreatedAt: 1710000000,
		Status:    "completed",
		Model:     "real-model",
		Output: []dto.ResponsesOutput{{
			Type:   "message",
			ID:     "msg_1",
			Status: "completed",
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{{
				Type: "output_text",
				Text: "ok",
			}},
		}},
		Usage: &dto.Usage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(respBody)))}
	if _, apiErr := OaiResponsesHandler(c, info, resp); apiErr != nil {
		t.Fatalf("OaiResponsesHandler error: %v", apiErr)
	}

	streamRecorder := httptest.NewRecorder()
	streamCtx, _ := gin.CreateTestContext(streamRecorder)
	streamCtx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	completedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.completed",
		Response: &dto.OpenAIResponsesResponse{
			ID:        "resp_1",
			Object:    "response",
			CreatedAt: 1710000000,
			Status:    "completed",
			Model:     "real-model",
			Usage:     &dto.Usage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
		},
	})
	if err != nil {
		t.Fatalf("marshal stream event: %v", err)
	}
	streamResp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: " + string(completedEvent) + "\n\ndata: [DONE]\n\n"))}
	if _, apiErr := OaiResponsesStreamHandler(streamCtx, info, streamResp); apiErr != nil {
		t.Fatalf("OaiResponsesStreamHandler error: %v", apiErr)
	}

	out := w.Body.String() + streamRecorder.Body.String()
	if strings.Contains(out, `"model":"real-model"`) {
		t.Fatalf("responses output leaked upstream model: %s", out)
	}
	if !strings.Contains(out, `"model":"alias-model"`) {
		t.Fatalf("responses output missing alias model: %s", out)
	}
}
