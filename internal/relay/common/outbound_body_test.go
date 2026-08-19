package common

import (
	"io"
	"testing"
)

func TestNewOutboundJSONBodyPreservesSizeAndContent(t *testing.T) {
	body, size, closer, err := NewOutboundJSONBody([]byte(`{"prompt":"hello"}`))
	if err != nil {
		t.Fatalf("NewOutboundJSONBody returned error: %v", err)
	}
	defer closer.Close()

	if size != int64(len(`{"prompt":"hello"}`)) {
		t.Fatalf("size = %d, want %d", size, len(`{"prompt":"hello"}`))
	}
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != `{"prompt":"hello"}` {
		t.Fatalf("body = %q, want original JSON", got)
	}
}
