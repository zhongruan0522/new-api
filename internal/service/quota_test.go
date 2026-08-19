package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NookMux/NookMux/internal/config/operation"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/internal/types"

	"github.com/gin-gonic/gin"
)

func newQuotaTestContext() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c
}

func TestNewEmptyUsageRetryErrorForNativeRequest(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI}}

	apiErr := NewEmptyUsageRetryError(newQuotaTestContext(), relayInfo)
	if apiErr == nil {
		t.Fatal("expected native empty usage to return retryable error")
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, apiErr.StatusCode)
	}
	if !operation.ShouldRetryByStatusCode(apiErr.StatusCode) {
		t.Fatalf("expected status %d to be included in automatic retry ranges", apiErr.StatusCode)
	}
	if types.IsSkipRetryError(apiErr) {
		t.Fatal("expected empty usage error not to skip retry")
	}
}

func TestNewEmptyUsageRetryErrorSkipsConvertedRequest(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude}}

	if apiErr := NewEmptyUsageRetryError(newQuotaTestContext(), relayInfo); apiErr != nil {
		t.Fatalf("expected converted empty usage not to force retry, got %v", apiErr)
	}
}
