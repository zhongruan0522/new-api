package dto

import "testing"

func TestParseImageAutoConvertToURLMode_LegacyCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		settings ChannelOtherSettings
		want     ImageAutoConvertToURLMode
		wantOK   bool
	}{
		{
			name:     "explicit mcp",
			settings: ChannelOtherSettings{ImageAutoConvertToURLMode: "mcp"},
			want:     ImageAutoConvertToURLModeMCP,
			wantOK:   true,
		},
		{
			name:     "legacy third party mode rewrites to mcp",
			settings: ChannelOtherSettings{ImageAutoConvertToURLMode: "third_party_model"},
			want:     ImageAutoConvertToURLModeMCP,
			wantOK:   true,
		},
		{
			name:     "legacy bool rewrites to mcp",
			settings: ChannelOtherSettings{ImageAutoConvertToURL: true},
			want:     ImageAutoConvertToURLModeMCP,
			wantOK:   true,
		},
		{
			name:     "invalid mode rejected",
			settings: ChannelOtherSettings{ImageAutoConvertToURLMode: "unexpected"},
			want:     ImageAutoConvertToURLModeOff,
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.settings.ParseImageAutoConvertToURLMode()
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("got (%q, %v), want (%q, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
