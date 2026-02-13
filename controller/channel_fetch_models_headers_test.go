package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

func TestBuildFetchModelsHeaders_SkipsPassthroughRulesAndClientHeader(t *testing.T) {
	override := `{
  "*": true,
  "re:^X-Trace-.*$": true,
  "X-Foo": "{client_header:X-Foo}",
  "User-Agent": "TestUA",
  "Authorization": "Bearer {api_key}"
}`
	channel := &model.Channel{
		Type:           constant.ChannelTypeOpenAI,
		HeaderOverride: &override,
	}

	headers, err := buildFetchModelsHeaders(channel, "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := headers.Get("Authorization"); got != "Bearer abc123" {
		t.Fatalf("expected Authorization %q, got %q", "Bearer abc123", got)
	}
	if got := headers.Get("User-Agent"); got != "TestUA" {
		t.Fatalf("expected User-Agent %q, got %q", "TestUA", got)
	}
	if got := headers.Get("*"); got != "" {
		t.Fatalf("expected passthrough rule header to be skipped, got %q", got)
	}
	if got := headers.Get("X-Foo"); got != "" {
		t.Fatalf("expected client_header placeholder to be skipped, got %q", got)
	}
}

func TestBuildFetchModelsHeaders_AppliesCherryStudioDefaultHeaders(t *testing.T) {
	channel := &model.Channel{
		Type: constant.ChannelTypeOpenAI,
	}

	headers, err := buildFetchModelsHeaders(channel, "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := headers.Get("x-title"); got != fetchModelsDefaultXTitle {
		t.Fatalf("expected x-title %q, got %q", fetchModelsDefaultXTitle, got)
	}
	if got := headers.Get("http-referer"); got != fetchModelsDefaultHTTPReferer {
		t.Fatalf("expected http-referer %q, got %q", fetchModelsDefaultHTTPReferer, got)
	}
	if got := headers.Get("user-agent"); got != fetchModelsDefaultUserAgent {
		t.Fatalf("expected user-agent %q, got %q", fetchModelsDefaultUserAgent, got)
	}
}

func TestBuildFetchModelsGeminiHeaders_PrefersAuthorization(t *testing.T) {
	override := `{"Authorization":"Bearer {api_key}"}`
	channel := &model.Channel{
		Type:           constant.ChannelTypeGemini,
		HeaderOverride: &override,
	}

	headers, err := buildFetchModelsGeminiHeaders(channel, "token-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := headers.Get("Authorization"); got != "Bearer token-123" {
		t.Fatalf("expected Authorization %q, got %q", "Bearer token-123", got)
	}
	if got := headers.Get("x-goog-api-key"); got != "" {
		t.Fatalf("expected x-goog-api-key to be omitted when Authorization is set, got %q", got)
	}
}

func TestBuildFetchModelsGeminiHeaders_DefaultsToXGoogAPIKey(t *testing.T) {
	channel := &model.Channel{
		Type: constant.ChannelTypeGemini,
	}

	headers, err := buildFetchModelsGeminiHeaders(channel, "token-456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := headers.Get("x-goog-api-key"); got != "token-456" {
		t.Fatalf("expected x-goog-api-key %q, got %q", "token-456", got)
	}
}

func TestBuildFetchModelsHeaders_ReturnsErrorOnNonStringHeaderValue(t *testing.T) {
	override := `{"User-Agent": true}`
	channel := &model.Channel{
		Type:           constant.ChannelTypeOpenAI,
		HeaderOverride: &override,
	}

	_, err := buildFetchModelsHeaders(channel, "abc123")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
