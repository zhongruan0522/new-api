package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/zhongruan0522/new-api/common"
)

func TestInitHttpClientAppliesRelayTransportLimits(t *testing.T) {
	oldMaxIdleConns := common.RelayMaxIdleConns
	oldMaxIdleConnsPerHost := common.RelayMaxIdleConnsPerHost
	oldIdleConnTimeout := common.RelayIdleConnTimeout
	oldRelayTimeout := common.RelayTimeout
	oldHTTPClient := httpClient
	t.Cleanup(func() {
		common.RelayMaxIdleConns = oldMaxIdleConns
		common.RelayMaxIdleConnsPerHost = oldMaxIdleConnsPerHost
		common.RelayIdleConnTimeout = oldIdleConnTimeout
		common.RelayTimeout = oldRelayTimeout
		httpClient = oldHTTPClient
	})

	common.RelayMaxIdleConns = 24
	common.RelayMaxIdleConnsPerHost = 8
	common.RelayIdleConnTimeout = 17
	common.RelayTimeout = 3

	InitHttpClient()

	if httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if httpClient.Timeout != 3*time.Second {
		t.Fatalf("client timeout = %s, want 3s", httpClient.Timeout)
	}
	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", httpClient.Transport)
	}
	if transport.MaxIdleConns != 24 {
		t.Fatalf("MaxIdleConns = %d, want 24", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 8 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 8", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 17*time.Second {
		t.Fatalf("IdleConnTimeout = %s, want 17s", transport.IdleConnTimeout)
	}
}
