package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

var removedThirdPartyMultimodalOptionKeys = []string{
	"global.third_party_multimodal_model_id",
	"global.third_party_multimodal_call_api_type",
	"global.third_party_multimodal_system_prompt",
	"global.third_party_multimodal_first_user_prompt",
	"global.third_party_multimodal_user_agent",
	"global.third_party_multimodal_x_title",
	"global.third_party_multimodal_http_referer",
}

// cleanupRemovedMultimodalTextMode removes the deprecated third-party
// media-to-text configuration and normalizes channel settings to the remaining
// MCP URL mode.
func cleanupRemovedMultimodalTextMode() error {
	if DB == nil {
		return nil
	}

	for _, key := range removedThirdPartyMultimodalOptionKeys {
		res := DB.Delete(&Option{Key: key})
		if res.Error != nil {
			return fmt.Errorf("remove obsolete option %s failed: %w", key, res.Error)
		}
		if res.RowsAffected > 0 {
			common.SysLog("removed obsolete option: " + key)
		}
	}

	type channelSettingsRow struct {
		Id            int    `gorm:"column:id"`
		OtherSettings string `gorm:"column:settings"`
	}

	var channels []channelSettingsRow
	if err := DB.Model(&Channel{}).
		Select("id", "settings").
		Where("settings IS NOT NULL AND settings <> ?", "").
		Find(&channels).Error; err != nil {
		return fmt.Errorf("list channel settings for multimodal cleanup failed: %w", err)
	}

	updatedCount := 0
	for _, channel := range channels {
		normalized, changed, err := normalizeRemovedMultimodalChannelOtherSettingsJSON(channel.OtherSettings)
		if err != nil {
			return fmt.Errorf("normalize channel %d settings failed: %w", channel.Id, err)
		}
		if !changed {
			continue
		}
		if err := DB.Model(&Channel{}).Where("id = ?", channel.Id).Update("settings", normalized).Error; err != nil {
			return fmt.Errorf("update channel %d settings failed: %w", channel.Id, err)
		}
		updatedCount++
	}

	if updatedCount > 0 {
		common.SysLog(fmt.Sprintf("normalized %d channel multimodal settings to MCP URL mode", updatedCount))
	}
	return nil
}

func normalizeRemovedMultimodalChannelOtherSettingsJSON(raw string) (normalized string, changed bool, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false, nil
	}

	settings := make(map[string]interface{})
	if err := common.UnmarshalJsonStr(trimmed, &settings); err != nil {
		return "", false, err
	}

	modeRaw, hasMode := settings["image_auto_convert_to_url_mode"]
	mode := ""
	if hasMode {
		modeValue, ok := modeRaw.(string)
		if !ok {
			return "", false, fmt.Errorf("image_auto_convert_to_url_mode must be a string, got %T", modeRaw)
		}
		mode = strings.TrimSpace(strings.ToLower(modeValue))
		switch mode {
		case "", string(dto.ImageAutoConvertToURLModeOff):
			if modeValue != string(dto.ImageAutoConvertToURLModeOff) {
				settings["image_auto_convert_to_url_mode"] = string(dto.ImageAutoConvertToURLModeOff)
				changed = true
			}
		case string(dto.ImageAutoConvertToURLModeMCP):
			if modeValue != string(dto.ImageAutoConvertToURLModeMCP) {
				settings["image_auto_convert_to_url_mode"] = string(dto.ImageAutoConvertToURLModeMCP)
				changed = true
			}
		case "third_party_model":
			settings["image_auto_convert_to_url_mode"] = string(dto.ImageAutoConvertToURLModeMCP)
			mode = string(dto.ImageAutoConvertToURLModeMCP)
			changed = true
		default:
			return "", false, fmt.Errorf("unsupported image_auto_convert_to_url_mode: %q", modeValue)
		}
	}

	if legacyRaw, ok := settings["image_auto_convert_to_url"]; ok {
		legacyEnabled, ok := legacyRaw.(bool)
		if !ok {
			return "", false, fmt.Errorf("image_auto_convert_to_url must be a bool, got %T", legacyRaw)
		}
		if legacyEnabled && mode == "" {
			settings["image_auto_convert_to_url_mode"] = string(dto.ImageAutoConvertToURLModeMCP)
		}
		delete(settings, "image_auto_convert_to_url")
		changed = true
	}

	if !changed {
		return raw, false, nil
	}

	settingBytes, err := common.Marshal(settings)
	if err != nil {
		return "", false, err
	}

	var normalizedMap map[string]json.RawMessage
	if err := common.Unmarshal(settingBytes, &normalizedMap); err != nil {
		return "", false, err
	}
	if len(normalizedMap) == 0 {
		return "{}", true, nil
	}

	return string(settingBytes), true, nil
}
