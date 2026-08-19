package common_handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/gin-gonic/gin"
)

// 上游在 HTTP 200 中携带 error body（部分网关会把 429/5xx 转成 200 下发）。
// handler 若不识别，客户端能看到真实错误，但计费阶段会因 usage 全零被误记为
// 「status_code=502, 上游没有返回计费信息，无法扣费（可能是上游超时）」。
func TestRerankHandlerReturnsUpstreamErrorBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/rerank", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "alias-model",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "real-model",
		},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Concurrent rate limit exceeded","type":"rate_limit_error","code":429}}`)),
	}

	usage, apiErr := RerankHandler(c, info, resp)
	require.NotNil(t, apiErr, "error body must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Concurrent rate limit exceeded")
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.Equal(t, shared.ErrorCode("429"), apiErr.GetErrorCode())
}

// 回归保护：正常 rerank 响应不受影响。
func TestRerankHandlerNormalResponseStillWorks(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/rerank", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "alias-model",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "real-model",
		},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"results":[{"index":0,"relevance_score":0.9}],"usage":{"total_tokens":10}}`)),
	}

	usage, apiErr := RerankHandler(c, info, resp)
	assert.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 10, usage.PromptTokens)
}
