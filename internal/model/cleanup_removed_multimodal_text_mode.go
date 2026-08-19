package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/dto"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"gorm.io/gorm"
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

// markerRemovedMultimodalTextMode is the one-shot migration marker recorded in
// the options table once this cleanup has fully completed. Subsequent startups
// skip the otherwise-every-start scan over the channels table. Delete the row to
// force a re-run (e.g. after importing legacy data).
const markerRemovedMultimodalTextMode = "data_migration.removed_multimodal_text_mode.v1.done"

// cleanupRemovedMultimodalTextMode removes the deprecated third-party
// media-to-text configuration and normalizes channel settings to the remaining
// MCP URL mode.
func cleanupRemovedMultimodalTextMode() error {
	if DB == nil {
		return nil
	}

	if isDataMigrationDone(markerRemovedMultimodalTextMode) {
		return nil
	}

	start := time.Now()

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

	// Text-level pre-filter narrows the scan to rows that can possibly need
	// normalization: encoding/json serializes keys verbatim, so any row that truly
	// needs a change must contain one of these substrings. Rows that match the
	// filter but turn out not to need a change are skipped after precise Go-level
	// JSON parsing in normalizeRemovedMultimodalChannelOtherSettingsJSON.
	var scannedChannels int64
	var updatedCount int64
	var channels []channelSettingsRow
	result := DB.Model(&Channel{}).
		Select("id", "settings").
		// Single Where with explicit parentheses so the OR stays grouped regardless
		// of how GORM merges conditions, keeping precedence identical across
		// SQLite/MySQL/PostgreSQL.
		Where("(settings IS NOT NULL AND settings <> ?) AND (settings LIKE ? OR settings LIKE ?)",
			"", "%image_auto_convert_to_url%", "%image_auto_convert_to_url_mode%").
		FindInBatches(&channels, 200, func(tx *gorm.DB, _ int) error {
			for i := range channels {
				scannedChannels++
				ch := channels[i]
				normalized, changed, err := normalizeRemovedMultimodalChannelOtherSettingsJSON(ch.OtherSettings)
				if err != nil {
					return fmt.Errorf("normalize channel %d settings failed: %w", ch.Id, err)
				}
				if !changed {
					continue
				}
				if err := tx.Model(&Channel{}).Where("id = ?", ch.Id).Update("settings", normalized).Error; err != nil {
					return fmt.Errorf("update channel %d settings failed: %w", ch.Id, err)
				}
				updatedCount++
			}
			return nil
		})
	if result.Error != nil {
		return fmt.Errorf("channel multimodal cleanup failed: %w", result.Error)
	}

	common.SysLog(fmt.Sprintf("multimodal text-mode cleanup: scanned %d channels, normalized %d channels to MCP URL mode in %s",
		scannedChannels, updatedCount, time.Since(start).Round(time.Millisecond)))

	markDataMigrationDone(markerRemovedMultimodalTextMode)
	return nil
}

func normalizeRemovedMultimodalChannelOtherSettingsJSON(raw string) (normalized string, changed bool, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false, nil
	}

	settings := make(map[string]interface{})
	if err := jsonx.UnmarshalJsonStr(trimmed, &settings); err != nil {
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

	settingBytes, err := jsonx.Marshal(settings)
	if err != nil {
		return "", false, err
	}

	var normalizedMap map[string]json.RawMessage
	if err := jsonx.Unmarshal(settingBytes, &normalizedMap); err != nil {
		return "", false, err
	}
	if len(normalizedMap) == 0 {
		return "{}", true, nil
	}

	return string(settingBytes), true, nil
}
