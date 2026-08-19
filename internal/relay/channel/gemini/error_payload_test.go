package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/internal/types"
	"github.com/gin-gonic/gin"
)

// 上游在 HTTP 200 中携带 {"error":{...}} 载荷（Gemini/中间网关会把 429/5xx
// 转成 200 下发）。handler 若不识别，计费阶段会因 totalTokens=0 被误记为
// 「status_code=502, 上游没有返回计费信息，无法扣费（可能是上游超时）」。
func newGeminiErrorTestContext(t *testing.T, path string) *gin.Context {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return c
}

func newGeminiTestInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName:   "alias-model",
		ResponseModelName: "alias-model",
		RelayFormat:       types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "real-model",
		},
	}
}

func TestGeminiStreamHandlerReturnsUpstreamErrorPayload(t *testing.T) {
	c := newGeminiErrorTestContext(t, "/v1/chat/completions")
	info := newGeminiTestInfo()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: " + `{"error":{"code":429,"message":"Resource has been exhausted","status":"RESOURCE_EXHAUSTED"}}` + "\n\ndata: [DONE]\n\n")),
	}

	usage, apiErr := GeminiChatStreamHandler(c, info, resp)
	require.NotNil(t, apiErr, "error payload must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Resource has been exhausted")
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
}

func TestGeminiTextGenerationHandlerReturnsUpstreamErrorPayload(t *testing.T) {
	c := newGeminiErrorTestContext(t, "/v1beta/models/real-model:generateContent")
	info := newGeminiTestInfo()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"code":503,"message":"The model is overloaded","status":"UNAVAILABLE"}}`)),
	}

	usage, apiErr := GeminiTextGenerationHandler(c, info, resp)
	require.NotNil(t, apiErr, "error payload must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "The model is overloaded")
	assert.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
}

func TestGeminiTextGenerationStreamHandlerReturnsUpstreamErrorPayload(t *testing.T) {
	c := newGeminiErrorTestContext(t, "/v1beta/models/real-model:streamGenerateContent")
	info := newGeminiTestInfo()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: " + `{"error":{"code":429,"message":"Resource has been exhausted","status":"RESOURCE_EXHAUSTED"}}` + "\n\ndata: [DONE]\n\n")),
	}

	usage, apiErr := GeminiTextGenerationStreamHandler(c, info, resp)
	require.NotNil(t, apiErr, "error payload must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Resource has been exhausted")
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
}

// 回归保护：正常流不应被错误识别影响。
func TestGeminiStreamHandlerNormalResponseStillWorks(t *testing.T) {
	c := newGeminiErrorTestContext(t, "/v1/chat/completions")
	info := newGeminiTestInfo()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: " + `{"candidates":[{"content":{"parts":[{"text":"hi"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}` + "\n\ndata: [DONE]\n\n")),
	}

	usage, apiErr := GeminiChatStreamHandler(c, info, resp)
	assert.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 15, usage.TotalTokens)
}
