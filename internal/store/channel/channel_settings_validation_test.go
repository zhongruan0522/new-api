package channelstore

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

func buildOtherSettingsJSON(t *testing.T, routing string) string {
	t.Helper()
	return `{"openrouter_routing":` + routing + `}`
}

func TestValidateSettingsAcceptsValidOpenRouterRouting(t *testing.T) {
	otherSettings := buildOtherSettingsJSON(t, `{
		"order": ["anthropic", "google-vertex/us-east5"],
		"only": ["deepinfra"],
		"ignore": ["bad-provider"],
		"allow_fallbacks": false,
		"require_parameters": true,
		"data_collection": "deny",
		"zdr": true,
		"quantizations": ["fp8", "int4"],
		"sort": {"by": "price", "partition": "none"},
		"preferred_min_throughput": {"p90": 50},
		"preferred_max_latency": 2.5,
		"max_price": {"prompt": 1, "completion": 2, "request": 0.5, "image": 0.1}
	}`)
	channel := &Channel{Type: constant.ChannelTypeOpenRouter, OtherSettings: otherSettings}
	if err := channel.ValidateSettings(); err != nil {
		t.Fatalf("expected valid openrouter_routing to pass, got %v", err)
	}
}

func TestValidateSettingsRejectsRoutingOnNonOpenRouterChannel(t *testing.T) {
	otherSettings := buildOtherSettingsJSON(t, `{"data_collection":"deny"}`)
	channel := &Channel{Type: constant.ChannelTypeOpenAI, OtherSettings: otherSettings}
	err := channel.ValidateSettings()
	if err == nil || !strings.Contains(err.Error(), "openrouter_routing") {
		t.Fatalf("expected openrouter_routing on non-OpenRouter channel to fail, got %v", err)
	}
}

func TestValidateSettingsRejectsInvalidRoutingValues(t *testing.T) {
	cases := []struct {
		name    string
		routing string
		wantErr string
	}{
		{"bad data_collection", `{"data_collection":"strict"}`, "data_collection"},
		{"bad sort by", `{"sort":{"by":"quality"}}`, "sort.by"},
		{"bad sort partition", `{"sort":{"by":"price","partition":"random"}}`, "sort.partition"},
		{"empty order slug", `{"order":["anthropic","  "]}`, "empty entry"},
		{"negative max_price", `{"max_price":{"prompt":-1}}`, "max_price.prompt"},
		{"negative threshold", `{"preferred_max_latency":-2}`, "preferred_max_latency"},
		{"multi percentile threshold", `{"preferred_min_throughput":{"p50":1,"p90":2}}`, "exactly one percentile"},
		{"unknown percentile", `{"preferred_min_throughput":{"p95":1}}`, "invalid openrouter percentile"},
		{"null percentile value", `{"preferred_max_latency":{"p50":null}}`, "must not be null"},
		{"non-numeric percentile value", `{"preferred_min_throughput":{"p50":"fast"}}`, "must be a number"},
		{"unknown routing field", `{"orders":["anthropic"]}`, "不是有效字段"},
		{"unknown sort field", `{"sort":{"by":"price","direction":"asc"}}`, "不是有效字段"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			channel := &Channel{
				Type:          constant.ChannelTypeOpenRouter,
				OtherSettings: buildOtherSettingsJSON(t, tc.routing),
			}
			err := channel.ValidateSettings()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateSettingsAcceptsLegacySettingsWithoutRouting(t *testing.T) {
	channel := &Channel{
		Type:          constant.ChannelTypeOpenRouter,
		OtherSettings: `{"openrouter_enterprise":true,"claude_beta_query":true}`,
	}
	if err := channel.ValidateSettings(); err != nil {
		t.Fatalf("expected legacy settings to pass unchanged, got %v", err)
	}
}

func TestOpenRouterThresholdUnmarshalDualShape(t *testing.T) {
	var numeric shared.OpenRouterThreshold
	if err := jsonx.Unmarshal([]byte(`25`), &numeric); err != nil {
		t.Fatalf("numeric threshold failed to parse: %v", err)
	}
	if numeric.Percentile != "" || numeric.Value != 25 {
		t.Fatalf("unexpected numeric threshold parse: %+v", numeric)
	}

	var percentile shared.OpenRouterThreshold
	if err := jsonx.Unmarshal([]byte(`{"p75": 40}`), &percentile); err != nil {
		t.Fatalf("percentile threshold failed to parse: %v", err)
	}
	if percentile.Percentile != "p75" || percentile.Value != 40 {
		t.Fatalf("unexpected percentile threshold parse: %+v", percentile)
	}
}
