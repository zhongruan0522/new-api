package logstore

import (
	"testing"
)

func TestSerializeLogOtherNormalizesNilAndEmpty(t *testing.T) {
	if got := serializeLogOther(nil); got != "{}" {
		t.Fatalf("serializeLogOther(nil) = %q, want {}", got)
	}
	if got := serializeLogOther(map[string]interface{}{}); got != "{}" {
		t.Fatalf("serializeLogOther(empty) = %q, want {}", got)
	}
	got := serializeLogOther(map[string]interface{}{"k": "v"})
	if got != `{"k":"v"}` {
		t.Fatalf("serializeLogOther(map) = %q, want {\"k\":\"v\"}", got)
	}
}
