package operation_setting

import "testing"

func TestSanitizeSubscriptionModelRatiosJSON(t *testing.T) {
	sanitized, err := SanitizeSubscriptionModelRatiosJSON(`{" gpt-4o ":1.25,"gemini-2.5-flash-lite-preview-thinking-abc":0.8}`)
	if err != nil {
		t.Fatalf("SanitizeSubscriptionModelRatiosJSON returned error: %v", err)
	}

	if err := UpdateSubscriptionModelRatiosByJSONString(sanitized); err != nil {
		t.Fatalf("UpdateSubscriptionModelRatiosByJSONString returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = UpdateSubscriptionModelRatiosByJSONString("{}")
	})

	if ratio, ok, key := GetSubscriptionModelRatio("gpt-4o"); !ok || key != "gpt-4o" || ratio != 1.25 {
		t.Fatalf("GetSubscriptionModelRatio(gpt-4o) = %v, %v, %q", ratio, ok, key)
	}
	if ratio, ok, key := GetSubscriptionModelRatio("gemini-2.5-flash-lite-preview-thinking-abc"); !ok || key != "gemini-2.5-flash-lite-thinking-*" || ratio != 0.8 {
		t.Fatalf("GetSubscriptionModelRatio(gemini) = %v, %v, %q", ratio, ok, key)
	}
}

func TestSanitizeSubscriptionModelRatiosJSONRejectsInvalidRatio(t *testing.T) {
	if _, err := SanitizeSubscriptionModelRatiosJSON(`{"gpt-4o":0}`); err == nil {
		t.Fatalf("expected zero ratio to be rejected")
	}
	if _, err := SanitizeSubscriptionModelRatiosJSON(`{"":1}`); err == nil {
		t.Fatalf("expected empty model name to be rejected")
	}
	if _, err := SanitizeSubscriptionModelRatiosJSON(`not json`); err == nil {
		t.Fatalf("expected malformed JSON to be rejected")
	}
}
