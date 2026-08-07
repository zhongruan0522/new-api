package common

import (
	"encoding/json"
	"testing"

	"github.com/NookMux/NookMux/dto"
)

func TestRemoveClaudeDisabledFieldsBlocksCacheControlAndSpeedByDefault(t *testing.T) {
	requestBody := []byte(`{
		"model":"claude-sonnet-4-5",
		"speed":"fast",
		"service_tier":"auto",
		"system":[{"type":"text","text":"reuse this","cache_control":{"type":"ephemeral"}}],
		"messages":[{"role":"user","content":[{"type":"text","text":"hello","cache_control":{"type":"ephemeral"}}]}],
		"tools":[{"name":"lookup","description":"search","input_schema":{"type":"object"},"cache_control":{"type":"ephemeral"}}]
	}`)

	out, err := RemoveClaudeDisabledFields(requestBody, dto.ChannelOtherSettings{})
	if err != nil {
		t.Fatalf("RemoveClaudeDisabledFields returned error: %v", err)
	}
	data := mustDecodeJSON(t, out)

	if hasJSONField(data, "cache_control") {
		t.Fatalf("cache_control should be blocked by default: %s", string(out))
	}
	if root := data.(map[string]any); root["speed"] != nil {
		t.Fatalf("speed should be blocked by default: %s", string(out))
	}
	if root := data.(map[string]any); root["service_tier"] != nil {
		t.Fatalf("service_tier should still follow the common default blocklist: %s", string(out))
	}
}

func TestRemoveClaudeDisabledFieldsAllowsConfiguredPassthrough(t *testing.T) {
	requestBody := []byte(`{
		"model":"claude-sonnet-4-5",
		"speed":"fast",
		"service_tier":"auto",
		"messages":[{"role":"user","content":[{"type":"text","text":"hello","cache_control":{"type":"ephemeral"}}]}]
	}`)

	out, err := RemoveClaudeDisabledFields(requestBody, dto.ChannelOtherSettings{
		AllowCacheControl: true,
		AllowSpeed:        true,
		AllowServiceTier:  true,
	})
	if err != nil {
		t.Fatalf("RemoveClaudeDisabledFields returned error: %v", err)
	}
	data := mustDecodeJSON(t, out)

	if !hasJSONField(data, "cache_control") {
		t.Fatalf("cache_control should pass through when enabled: %s", string(out))
	}
	root := data.(map[string]any)
	if root["speed"] != "fast" {
		t.Fatalf("speed should pass through when enabled: %#v", root["speed"])
	}
	if root["service_tier"] != "auto" {
		t.Fatalf("service_tier should pass through with its own setting enabled: %#v", root["service_tier"])
	}
}

func mustDecodeJSON(t *testing.T, data []byte) any {
	t.Helper()
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v\n%s", err, string(data))
	}
	return decoded
}

func hasJSONField(value any, field string) bool {
	switch typed := value.(type) {
	case map[string]any:
		if _, ok := typed[field]; ok {
			return true
		}
		for _, child := range typed {
			if hasJSONField(child, field) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if hasJSONField(child, field) {
				return true
			}
		}
	}
	return false
}
