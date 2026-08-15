package controller

import (
	"encoding/json"
	"testing"
)

func TestOptionUpdateValueToStringRejectsCompositeValues(t *testing.T) {
	cases := []struct {
		name  string
		value any
		ok    bool
	}{
		{name: "string", value: `{"a":"b"}`, ok: true},
		{name: "bool", value: true, ok: true},
		{name: "number", value: float64(1), ok: true},
		{name: "array", value: []any{"voice-a"}, ok: false},
		{name: "object", value: map[string]any{"a": "b"}, ok: false},
		{name: "nil", value: nil, ok: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := optionUpdateValueToString(tc.value)
			if ok != tc.ok {
				t.Fatalf("optionUpdateValueToString ok = %v, want %v", ok, tc.ok)
			}
		})
	}
}

// PasswordLoginEnabled/PasswordRegisterEnabled 是布尔功能开关而非 secret，
// GetOptions 必须继续返回它们，否则认证设置页回落到前端默认值。
func TestIsSensitiveOptionKeyExemptsPasswordFeatureToggles(t *testing.T) {
	if isSensitiveOptionKey("PasswordLoginEnabled") {
		t.Fatal("PasswordLoginEnabled is a boolean feature toggle, must not be filtered")
	}
	if isSensitiveOptionKey("PasswordRegisterEnabled") {
		t.Fatal("PasswordRegisterEnabled is a boolean feature toggle, must not be filtered")
	}
	// secret 类 Password 键仍必须被过滤
	for _, key := range []string{"Password", "SMTPPassword", "oauth_password", "PasswordResetSecret"} {
		if !isSensitiveOptionKey(key) {
			t.Fatalf("key %q must be treated as sensitive", key)
		}
	}
}

func TestPricingJsonMapOptionValidation(t *testing.T) {
	if !isPricingJsonMapOptionKey("ModelRatio") {
		t.Fatal("ModelRatio should support JSON map incremental updates")
	}
	if err := validatePricingJsonMapOption("ModelRatio", `{"gpt-4o":2.5,"free":0}`); err != nil {
		t.Fatalf("validate numeric pricing map: %v", err)
	}
	if err := validatePricingJsonMapOption("ModelRatio", `{"gpt-4o":{"bad":true}}`); err == nil {
		t.Fatal("expected non-numeric pricing map value to fail validation")
	}
	if err := validatePricingJsonMapOption("ContextPricing", `{"tiered-model":{"enabled":true,"tiers":[{"name":"default","min_tokens":0,"model_ratio":1,"completion_ratio":2,"cache_ratio":0.5,"create_cache_ratio":1.25,"audio_ratio":1,"audio_completion_ratio":2}]}}`); err != nil {
		t.Fatalf("validate context pricing map: %v", err)
	}
}

func TestMarshalPricingJsonMapOptionPreservesRawValue(t *testing.T) {
	items := map[string]json.RawMessage{
		"free-model": json.RawMessage(`0`),
		"paid-model": json.RawMessage(`0.75`),
	}
	nextValue, err := marshalPricingJsonMapOption("ModelPrice", items)
	if err != nil {
		t.Fatalf("marshal pricing json map option: %v", err)
	}
	if nextValue != `{"free-model":0,"paid-model":0.75}` {
		t.Fatalf("unexpected marshaled value: %s", nextValue)
	}
}
