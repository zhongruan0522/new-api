package model

import (
	"strings"
	"testing"

	"github.com/zhongruan0522/new-api/common"
)

func TestNormalizeRemovedMultimodalChannelOtherSettingsJSON(t *testing.T) {
	raw := `{"claude_beta_query":true,"image_auto_convert_to_url":true,"image_auto_convert_to_url_mode":"third_party_model"}`

	normalized, changed, err := normalizeRemovedMultimodalChannelOtherSettingsJSON(raw)
	if err != nil {
		t.Fatalf("expected err=nil, got %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	var got map[string]interface{}
	if err := common.Unmarshal([]byte(normalized), &got); err != nil {
		t.Fatalf("expected normalized JSON to be valid, got %v", err)
	}
	if got["image_auto_convert_to_url_mode"] != "mcp" {
		t.Fatalf("expected image_auto_convert_to_url_mode=mcp, got %#v", got["image_auto_convert_to_url_mode"])
	}
	if _, ok := got["image_auto_convert_to_url"]; ok {
		t.Fatalf("expected legacy image_auto_convert_to_url to be removed, got %#v", got)
	}
	if got["claude_beta_query"] != true {
		t.Fatalf("expected unrelated settings to be preserved, got %#v", got)
	}
}

func TestNormalizeRemovedMultimodalChannelOtherSettingsJSONRejectsInvalidType(t *testing.T) {
	_, _, err := normalizeRemovedMultimodalChannelOtherSettingsJSON(`{"image_auto_convert_to_url_mode":123}`)
	if err == nil {
		t.Fatal("expected invalid mode type to fail")
	}
}

func TestChannelValidateSettingsRejectsRemovedMultimodalFields(t *testing.T) {
	tests := []struct {
		name          string
		otherSettings string
		wantContains  string
	}{
		{
			name:          "legacy bool field",
			otherSettings: `{"image_auto_convert_to_url":true}`,
			wantContains:  `image_auto_convert_to_url`,
		},
		{
			name:          "removed third party mode",
			otherSettings: `{"image_auto_convert_to_url_mode":"third_party_model"}`,
			wantContains:  `third_party_model`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			channel := &Channel{OtherSettings: tc.otherSettings}
			err := channel.ValidateSettings()
			if err == nil {
				t.Fatal("expected removed field to be rejected")
			}
			if !strings.Contains(err.Error(), tc.wantContains) {
				t.Fatalf("expected error to contain %q, got %q", tc.wantContains, err.Error())
			}
		})
	}
}
