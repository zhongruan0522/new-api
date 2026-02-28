package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

func cleanupRemovedChatPlaygroundData() error {
	if DB == nil {
		return nil
	}

	// ---------------------------------------------------------------------
	// 1) Remove obsolete options: Chats / ChatLink / ChatLink2
	// ---------------------------------------------------------------------
	if res := DB.Delete(&Option{Key: "Chats"}); res.Error != nil {
		return res.Error
	} else if res.RowsAffected > 0 {
		common.SysLog("removed obsolete option: Chats")
	}
	for _, k := range []string{"ChatLink", "ChatLink2"} {
		if res := DB.Delete(&Option{Key: k}); res.Error != nil {
			return res.Error
		} else if res.RowsAffected > 0 {
			common.SysLog(fmt.Sprintf("removed obsolete option: %s", k))
		}
	}

	// ---------------------------------------------------------------------
	// 2) Sanitize global admin sidebar modules
	// ---------------------------------------------------------------------
	var sidebarAdmin Option
	if err := DB.First(&sidebarAdmin, &Option{Key: "SidebarModulesAdmin"}).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		sanitized, changed, err := SanitizeSidebarModulesConfigJSON(sidebarAdmin.Value)
		if err != nil {
			return err
		}
		if changed {
			if err := DB.Model(&Option{}).Where(&Option{Key: sidebarAdmin.Key}).Update("value", sanitized).Error; err != nil {
				return err
			}
			common.SysLog("cleaned option SidebarModulesAdmin: removed chat section")
		}
	}

	// ---------------------------------------------------------------------
	// 3) Sanitize per-user sidebar modules stored in users.setting
	// ---------------------------------------------------------------------
	var updatedUsers int64
	var users []User
	result := DB.
		Select("id", "setting").
		Where("setting <> '' AND setting LIKE ?", "%sidebar_modules%").
		FindInBatches(&users, 200, func(tx *gorm.DB, _ int) error {
			for i := range users {
				u := users[i]
				if !strings.Contains(u.Setting, "sidebar_modules") {
					continue
				}
				sanitizedSetting, changed, err := sanitizeUserSettingSidebarModulesJSON(u.Setting)
				if err != nil {
					return err
				}
				if !changed {
					continue
				}
				if err := tx.Model(&User{}).Where("id = ?", u.Id).Update("setting", sanitizedSetting).Error; err != nil {
					return err
				}
				updatedUsers++
			}
			return nil
		})
	if result.Error != nil {
		return result.Error
	}
	if updatedUsers > 0 {
		common.SysLog(fmt.Sprintf("cleaned %d users.setting.sidebar_modules: removed chat section", updatedUsers))
	}
	return nil
}

func sanitizeUserSettingSidebarModulesJSON(settingJSON string) (string, bool, error) {
	settingJSON = strings.TrimSpace(settingJSON)
	if settingJSON == "" {
		return settingJSON, false, nil
	}

	var m map[string]any
	if err := common.Unmarshal([]byte(settingJSON), &m); err != nil {
		// Setting JSON is corrupted; avoid destructive updates.
		return settingJSON, false, nil
	}

	rawSidebarModules, ok := m["sidebar_modules"]
	if !ok {
		return settingJSON, false, nil
	}

	var sidebarModulesStr string
	switch v := rawSidebarModules.(type) {
	case string:
		sidebarModulesStr = v
	default:
		// Unexpected type (object/array/etc). Marshal to JSON string for sanitation.
		b, err := common.Marshal(v)
		if err != nil {
			return "", false, err
		}
		sidebarModulesStr = string(b)
	}

	sanitizedSidebar, changed, err := SanitizeSidebarModulesConfigJSON(sidebarModulesStr)
	if err != nil {
		return "", false, err
	}
	if !changed {
		return settingJSON, false, nil
	}

	// Store back as string (expected by dto.UserSetting and existing APIs).
	m["sidebar_modules"] = sanitizedSidebar

	b, err := common.Marshal(m)
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}

func SanitizeSidebarModulesConfigJSON(configJSON string) (string, bool, error) {
	configJSON = strings.TrimSpace(configJSON)
	if configJSON == "" {
		return configJSON, false, nil
	}

	var config map[string]any
	if err := common.Unmarshal([]byte(configJSON), &config); err != nil {
		// Invalid sidebar config: clear it so the frontend falls back to defaults.
		return "", true, nil
	}

	if _, exists := config["chat"]; !exists {
		return configJSON, false, nil
	}
	delete(config, "chat")

	b, err := common.Marshal(config)
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}
