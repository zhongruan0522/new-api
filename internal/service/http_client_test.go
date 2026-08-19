package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/common"

	"github.com/gorilla/websocket"
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

func TestNewProxyWebSocketDialer(t *testing.T) {
	// 空代理返回默认拨号器。
	dialer, err := NewProxyWebSocketDialer("")
	if err != nil {
		t.Fatalf("empty proxy returned error: %v", err)
	}
	if dialer != defaultWebSocketDialerForCompare() {
		t.Fatal("empty proxy should return the default dialer")
	}

	// http 代理返回携带 Proxy 函数的独立拨号器。
	dialer, err = NewProxyWebSocketDialer("http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("http proxy returned error: %v", err)
	}
	if dialer == nil || dialer.Proxy == nil {
		t.Fatal("http proxy dialer should have a Proxy function")
	}

	// socks5 代理返回携带 NetDial 的独立拨号器。
	dialer, err = NewProxyWebSocketDialer("socks5://user:pass@127.0.0.1:1080")
	if err != nil {
		t.Fatalf("socks5 proxy returned error: %v", err)
	}
	if dialer == nil || dialer.NetDial == nil {
		t.Fatal("socks5 proxy dialer should have a NetDial function")
	}

	// 非法 scheme 必须报错而不是静默直连。
	if _, err = NewProxyWebSocketDialer("ftp://127.0.0.1:21"); err == nil {
		t.Fatal("unsupported scheme should return an error")
	}
}

func defaultWebSocketDialerForCompare() *websocket.Dialer {
	return websocket.DefaultDialer
}
