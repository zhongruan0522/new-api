package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NookMux/NookMux/constant"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/types"
	"github.com/gin-gonic/gin"
)

func TestOpenaiHandlerCountsLlamaCachedTokensFromTimings(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: "llama-3.1-8b",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"id": "chatcmpl-llama",
			"object": "chat.completion",
			"created": 1710000000,
			"model": "llama-3.1-8b",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "hello"},
				"finish_reason": "stop"
			}],
			"usage": {
				"prompt_tokens": 120,
				"completion_tokens": 5,
				"total_tokens": 125
			},
			"timings": {
				"cache_n": 64
			}
		}`)),
	}

	usage, apiErr := OpenaiHandler(c, info, resp)
	if apiErr != nil {
		t.Fatalf("OpenaiHandler error: %v", apiErr)
	}
	if usage.PromptTokensDetails.CachedTokens != 64 {
		t.Fatalf("cached tokens = %d, want 64", usage.PromptTokensDetails.CachedTokens)
	}
}
