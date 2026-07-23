package relay

import (
	"testing"

	"github.com/NookMux/NookMux/dto"
)

func TestResolveImageQuality(t *testing.T) {
	cases := []struct {
		name    string
		quality string
		want    string
	}{
		{"empty falls back to standard", "", "standard"},
		{"standard is preserved", "standard", "standard"},
		{"hd is preserved", "hd", "hd"},
		{"low is preserved", "low", "low"},
		{"medium is preserved", "medium", "medium"},
		{"high is preserved", "high", "high"},
		{"auto is preserved", "auto", "auto"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &dto.ImageRequest{Quality: tc.quality}
			got := resolveImageQuality(req)
			if got != tc.want {
				t.Fatalf("resolveImageQuality(%q) = %q, want %q", tc.quality, got, tc.want)
			}
		})
	}
}

func TestResolveImageQualityDoesNotForceStandard(t *testing.T) {
	// Regression test: a non-hd upstream quality used to be silently rewritten
	// to "standard", which broke upstreams that support values like low/medium/high.
	req := &dto.ImageRequest{Quality: "medium"}
	if got := resolveImageQuality(req); got != "medium" {
		t.Fatalf("expected user-provided quality to be preserved, got %q", got)
	}
}
