package controller

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPClientAppliesSyncIdleDefaults(t *testing.T) {
	t.Setenv("SYNC_HTTP_MAX_IDLE_CONNS", "")
	t.Setenv("SYNC_HTTP_IDLE_CONN_TIMEOUT", "")

	client := newHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.MaxIdleConns != 10 {
		t.Fatalf("MaxIdleConns = %d, want 10", transport.MaxIdleConns)
	}
	if transport.IdleConnTimeout != 30*time.Second {
		t.Fatalf("IdleConnTimeout = %s, want 30s", transport.IdleConnTimeout)
	}
}

func TestNewHTTPClientAppliesSyncIdleOverrides(t *testing.T) {
	t.Setenv("SYNC_HTTP_MAX_IDLE_CONNS", "7")
	t.Setenv("SYNC_HTTP_IDLE_CONN_TIMEOUT", "11")

	client := newHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.MaxIdleConns != 7 {
		t.Fatalf("MaxIdleConns = %d, want 7", transport.MaxIdleConns)
	}
	if transport.IdleConnTimeout != 11*time.Second {
		t.Fatalf("IdleConnTimeout = %s, want 11s", transport.IdleConnTimeout)
	}
}
