package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/NookMux/NookMux/internal/config/operation"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryUsesNumericUpstreamErrorCode(t *testing.T) {
	orig := operation.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { operation.AutomaticRetryStatusCodeRanges = orig })
	operation.AutomaticRetryStatusCodeRanges = []operation.StatusCodeRange{{Start: 500, End: 599}}

	for _, code := range []string{"502", "504"} {
		t.Run(code, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			err := shared.WithOpenAIError(shared.OpenAIError{
				Message: "upstream gateway error",
				Type:    "upstream_error",
				Code:    code,
			}, http.StatusOK)

			require.True(t, shouldRetry(c, err, 1))
		})
	}
}

func TestShouldRetryIgnoresNonNumericUpstreamErrorCode(t *testing.T) {
	orig := operation.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { operation.AutomaticRetryStatusCodeRanges = orig })
	operation.AutomaticRetryStatusCodeRanges = []operation.StatusCodeRange{{Start: 500, End: 599}}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := shared.WithOpenAIError(shared.OpenAIError{
		Message: "bad request",
		Type:    "invalid_request_error",
		Code:    "invalid_request",
	}, http.StatusBadRequest)

	require.False(t, shouldRetry(c, err, 1))
}

func TestShouldRetryUsesOriginalStatusCodeAfterMapping(t *testing.T) {
	orig := operation.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { operation.AutomaticRetryStatusCodeRanges = orig })
	operation.AutomaticRetryStatusCodeRanges = []operation.StatusCodeRange{{Start: 429, End: 429}}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := shared.WithOpenAIError(shared.OpenAIError{
		Message: "rate limited",
		Type:    "upstream_error",
		Code:    "rate_limit_exceeded",
	}, http.StatusTooManyRequests)

	service.ResetStatusCode(err, `{"429":"200"}`)

	require.Equal(t, http.StatusTooManyRequests, err.OriginalStatusCode)
	require.Equal(t, http.StatusOK, err.StatusCode)
	require.True(t, shouldRetry(c, err, 1))
}

func TestShouldRetryConfiguredTransientStatusCodes(t *testing.T) {
	orig := operation.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { operation.AutomaticRetryStatusCodeRanges = orig })
	require.NoError(t, operation.AutomaticRetryStatusCodesFromString("100-199,300-399,401-407,409-599"))

	for _, statusCode := range []int{http.StatusTooManyRequests, 529} {
		t.Run(strconv.Itoa(statusCode), func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			err := shared.WithOpenAIError(shared.OpenAIError{
				Message: "transient upstream failure",
				Type:    "upstream_error",
				Code:    statusCode,
			}, statusCode)

			require.True(t, shouldRetry(c, err, 20))
		})
	}
}
