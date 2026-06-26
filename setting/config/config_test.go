package config

import (
	"reflect"
	"testing"
)

type testNestedConfig struct {
	Values map[string]string `json:"values"`
}

type testConfig struct {
	Name      string            `json:"name,omitempty"`
	Enabled   bool              `json:"enabled"`
	Redirects map[string]string `json:"redirects"`
	Nested    testNestedConfig  `json:"nested"`
	NestedPtr *testNestedConfig `json:"nested_ptr"`
	Ignored   map[string]string `json:"-"`
	untouched map[string]string
}

func TestUpdateConfigFromMapReplacesMapFields(t *testing.T) {
	cfg := &testConfig{
		Redirects: map[string]string{
			"stale": "old",
			"kept":  "old",
		},
	}

	err := UpdateConfigFromMap(cfg, map[string]string{
		"redirects": `{"kept":"new"}`,
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap returned error: %v", err)
	}

	want := map[string]string{"kept": "new"}
	if !reflect.DeepEqual(cfg.Redirects, want) {
		t.Fatalf("redirects mismatch: got %#v, want %#v", cfg.Redirects, want)
	}
}

func TestUpdateConfigFromMapReplacesNestedMapFields(t *testing.T) {
	cfg := &testConfig{
		Nested: testNestedConfig{Values: map[string]string{
			"stale": "old",
			"kept":  "old",
		}},
		NestedPtr: &testNestedConfig{Values: map[string]string{
			"stale": "old",
			"kept":  "old",
		}},
	}

	err := UpdateConfigFromMap(cfg, map[string]string{
		"nested":     `{"values":{"kept":"new"}}`,
		"nested_ptr": `{"values":{"kept":"ptr-new"}}`,
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap returned error: %v", err)
	}

	wantNested := map[string]string{"kept": "new"}
	if !reflect.DeepEqual(cfg.Nested.Values, wantNested) {
		t.Fatalf("nested values mismatch: got %#v, want %#v", cfg.Nested.Values, wantNested)
	}
	if cfg.NestedPtr == nil {
		t.Fatal("nested ptr is nil")
	}
	wantNestedPtr := map[string]string{"kept": "ptr-new"}
	if !reflect.DeepEqual(cfg.NestedPtr.Values, wantNestedPtr) {
		t.Fatalf("nested ptr values mismatch: got %#v, want %#v", cfg.NestedPtr.Values, wantNestedPtr)
	}
}

func TestUpdateConfigFromMapReturnsParseErrors(t *testing.T) {
	cfg := &testConfig{
		Redirects: map[string]string{"kept": "old"},
	}

	err := UpdateConfigFromMap(cfg, map[string]string{
		"redirects": `{bad json}`,
	})
	if err == nil {
		t.Fatal("expected error for invalid map JSON")
	}

	want := map[string]string{"kept": "old"}
	if !reflect.DeepEqual(cfg.Redirects, want) {
		t.Fatalf("invalid update should not mutate redirects: got %#v, want %#v", cfg.Redirects, want)
	}
}

func TestValidateConfigFromMapDoesNotMutateConfig(t *testing.T) {
	cfg := &testConfig{
		Redirects: map[string]string{"kept": "old"},
	}

	err := ValidateConfigFromMap(cfg, map[string]string{
		"redirects": `{"new":"value"}`,
	})
	if err != nil {
		t.Fatalf("ValidateConfigFromMap returned error: %v", err)
	}

	want := map[string]string{"kept": "old"}
	if !reflect.DeepEqual(cfg.Redirects, want) {
		t.Fatalf("validation should not mutate redirects: got %#v, want %#v", cfg.Redirects, want)
	}
}

func TestConfigToMapUsesJSONTagNameWithoutOptions(t *testing.T) {
	cfg := &testConfig{
		Name:      "demo",
		Redirects: map[string]string{"a": "b"},
		Ignored:   map[string]string{"ignored": "true"},
		untouched: map[string]string{"private": "true"},
	}

	m, err := ConfigToMap(cfg)
	if err != nil {
		t.Fatalf("ConfigToMap returned error: %v", err)
	}

	if _, ok := m["name"]; !ok {
		t.Fatalf("expected json tag name without options, got keys %#v", m)
	}
	if _, ok := m["name,omitempty"]; ok {
		t.Fatalf("unexpected raw json tag key in map: %#v", m)
	}
	if _, ok := m["-"]; ok {
		t.Fatalf("ignored field should not be exported: %#v", m)
	}
	if _, ok := m["untouched"]; ok {
		t.Fatalf("unexported field should not be exported: %#v", m)
	}
}
