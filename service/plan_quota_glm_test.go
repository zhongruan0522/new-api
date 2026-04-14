package service

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
)

func TestGlmLimitRespUnmarshalSupportsNumericNextResetTime(t *testing.T) {
	raw := []byte(`{"data":{"limits":[{"type":"TOKENS_LIMIT","percentage":12,"nextResetTime":1735689600000}]}}`)

	var resp glmLimitResp
	if err := common.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("common.Unmarshal() error = %v, want nil", err)
	}

	if len(resp.Data.Limits) != 1 {
		t.Fatalf("limits count = %d, want 1", len(resp.Data.Limits))
	}

	if got := string(resp.Data.Limits[0].NextResetTime); got != "2025-01-01T00:00:00Z" {
		t.Fatalf("nextResetTime = %q, want %q", got, "2025-01-01T00:00:00Z")
	}
}

func TestGlmLimitRespUnmarshalKeepsStringNextResetTime(t *testing.T) {
	raw := []byte(`{"data":{"limits":[{"unit":6,"percentage":35,"nextResetTime":"2026-04-20T00:00:00+08:00"}]}}`)

	var resp glmLimitResp
	if err := common.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("common.Unmarshal() error = %v, want nil", err)
	}

	if len(resp.Data.Limits) != 1 {
		t.Fatalf("limits count = %d, want 1", len(resp.Data.Limits))
	}

	if got := string(resp.Data.Limits[0].NextResetTime); got != "2026-04-20T00:00:00+08:00" {
		t.Fatalf("nextResetTime = %q, want %q", got, "2026-04-20T00:00:00+08:00")
	}
}
