package aws

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestFormatRequestPreservesContextManagement(t *testing.T) {
	const contextManagement = `{"type":"clear_at_turn","clear_at_turn":{"turns":3}}`
	body := strings.NewReader(`{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens": 1024,
		"messages": [{"role": "user", "content": "hello"}],
		"context_management": ` + contextManagement + `
	}`)

	got, err := formatRequest(body, http.Header{})
	if err != nil {
		t.Fatalf("formatRequest: %v", err)
	}
	if got == nil {
		t.Fatal("formatRequest returned nil request")
	}
	if !bytes.Equal(got.ContextManagement, []byte(contextManagement)) {
		t.Fatalf("ContextManagement = %s, want %s", got.ContextManagement, contextManagement)
	}
	if got.AnthropicVersion != "bedrock-2023-05-31" {
		t.Fatalf("AnthropicVersion = %q", got.AnthropicVersion)
	}
}
