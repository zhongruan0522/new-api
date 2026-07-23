package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/setting/operation_setting"
	"github.com/NookMux/NookMux/types"
)

func TestShouldRetryUsesNumericUpstreamErrorCode(t *testing.T) {
	orig := operation_setting.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { operation_setting.AutomaticRetryStatusCodeRanges = orig })
	operation_setting.AutomaticRetryStatusCodeRanges = []operation_setting.StatusCodeRange{{Start: 500, End: 599}}

	for _, code := range []string{"502", "504"} {
		t.Run(code, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			err := types.WithOpenAIError(types.OpenAIError{
				Message: "upstream gateway error",
				Type:    "upstream_error",
				Code:    code,
			}, http.StatusOK)

			require.True(t, shouldRetry(c, err, 1))
		})
	}
}

func TestShouldRetryIgnoresNonNumericUpstreamErrorCode(t *testing.T) {
	orig := operation_setting.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { operation_setting.AutomaticRetryStatusCodeRanges = orig })
	operation_setting.AutomaticRetryStatusCodeRanges = []operation_setting.StatusCodeRange{{Start: 500, End: 599}}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "bad request",
		Type:    "invalid_request_error",
		Code:    "invalid_request",
	}, http.StatusBadRequest)

	require.False(t, shouldRetry(c, err, 1))
}

func TestShouldRetryUsesOriginalStatusCodeAfterMapping(t *testing.T) {
	orig := operation_setting.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { operation_setting.AutomaticRetryStatusCodeRanges = orig })
	operation_setting.AutomaticRetryStatusCodeRanges = []operation_setting.StatusCodeRange{{Start: 429, End: 429}}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "rate limited",
		Type:    "upstream_error",
		Code:    "rate_limit_exceeded",
	}, http.StatusTooManyRequests)

	service.ResetStatusCode(err, `{"429":"200"}`)

	require.Equal(t, http.StatusTooManyRequests, err.OriginalStatusCode)
	require.Equal(t, http.StatusOK, err.StatusCode)
	require.True(t, shouldRetry(c, err, 1))
}
