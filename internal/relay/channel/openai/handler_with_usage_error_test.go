package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gin-gonic/gin"
)

// 上游在 HTTP 200 中携带 error body（部分网关会把 429/5xx 转成 200 下发）。
// handler 若不识别，客户端能看到真实错误，但计费阶段会因 usage 全零被误记为
// 「status_code=502, 上游没有返回计费信息，无法扣费（可能是上游超时）」。
func newImageErrorTestContext(t *testing.T) *gin.Context {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	return c
}

func TestOpenaiHandlerWithUsageReturnsUpstreamErrorBody(t *testing.T) {
	c := newImageErrorTestContext(t)
	info := testRelayInfo("alias-model", "real-model")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Billing hard limit reached","type":"insufficient_quota","code":"429"}}`)),
	}

	usage, apiErr := OpenaiHandlerWithUsage(c, info, resp)
	require.NotNil(t, apiErr, "error body must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Billing hard limit reached")
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
}

// 回归保护：正常图片响应不受影响。
func TestOpenaiHandlerWithUsageNormalBodyStillWorks(t *testing.T) {
	c := newImageErrorTestContext(t)
	info := testRelayInfo("alias-model", "real-model")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"created":1710000000,"data":[{"url":"https://example.com/img.png"}],"usage":{"total_tokens":1}}`)),
	}

	usage, apiErr := OpenaiHandlerWithUsage(c, info, resp)
	assert.Nil(t, apiErr)
	require.NotNil(t, usage)
}
