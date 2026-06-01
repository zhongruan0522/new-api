package service

import (
	"net/http"
	"testing"

	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/setting/operation_setting"
	"github.com/zhongruan0522/new-api/types"
)

func TestNewEmptyUsageRetryErrorForNativeRequest(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI}}

	apiErr := NewEmptyUsageRetryError(relayInfo)
	if apiErr == nil {
		t.Fatal("expected native empty usage to return retryable error")
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, apiErr.StatusCode)
	}
	if !operation_setting.ShouldRetryByStatusCode(apiErr.StatusCode) {
		t.Fatalf("expected status %d to be included in automatic retry ranges", apiErr.StatusCode)
	}
	if types.IsSkipRetryError(apiErr) {
		t.Fatal("expected empty usage error not to skip retry")
	}
}

func TestNewEmptyUsageRetryErrorSkipsConvertedRequest(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude}}

	if apiErr := NewEmptyUsageRetryError(relayInfo); apiErr != nil {
		t.Fatalf("expected converted empty usage not to force retry, got %v", apiErr)
	}
}
