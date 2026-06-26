package controller

import "testing"

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
