package billing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NookMux/NookMux/internal/config/operation"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/gin-gonic/gin"
)

func newQuotaTestContext() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c
}

func TestNewEmptyUsageRetryErrorForNativeRequest(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{RequestConversionChain: []relayconstant.RelayFormat{relayconstant.RelayFormatOpenAI}}

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
	if shared.IsSkipRetryError(apiErr) {
		t.Fatal("expected empty usage error not to skip retry")
	}
}

func TestNewEmptyUsageRetryErrorSkipsConvertedRequest(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{RequestConversionChain: []relayconstant.RelayFormat{relayconstant.RelayFormatOpenAI, relayconstant.RelayFormatClaude}}

	if apiErr := NewEmptyUsageRetryError(newQuotaTestContext(), relayInfo); apiErr != nil {
		t.Fatalf("expected converted empty usage not to force retry, got %v", apiErr)
	}
}
