package tokenstore

import (
	"testing"
)

func TestMaskTokenKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "empty", key: "", want: ""},
		{name: "short", key: "abcd", want: "****"},
		{name: "full length 48", key: "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKL", want: "************************************************"},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := MaskTokenKey(testCase.key)
			if got != testCase.want {
				t.Fatalf("MaskTokenKey(%q) = %q, want %q", testCase.key, got, testCase.want)
			}
			if testCase.key != "" && len(got) != len(testCase.key) {
				t.Fatalf("masked length = %d, want %d", len(got), len(testCase.key))
			}
		})
	}
}

func TestTokenGetMaskedAndFullKey(t *testing.T) {
	t.Parallel()

	token := &Token{Key: "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKL"}
	if token.GetFullKey() != token.Key {
		t.Fatalf("GetFullKey() = %q, want %q", token.GetFullKey(), token.Key)
	}
	masked := token.GetMaskedKey()
	if masked == token.Key {
		t.Fatal("GetMaskedKey() must not return the real key")
	}
	if len(masked) != len(token.Key) {
		t.Fatalf("GetMaskedKey length = %d, want %d", len(masked), len(token.Key))
	}
	for _, ch := range masked {
		if ch != '*' {
			t.Fatalf("GetMaskedKey() = %q, want all '*'", masked)
		}
	}
}
