package model

import (
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
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
	// 2) (Historically sanitized SidebarModulesAdmin chat section — no longer needed)
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	// 3) Remove per-user sidebar_modules from users.setting
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
				sanitizedSetting, changed, err := removeUserSettingSidebarModules(u.Setting)
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
		common.SysLog(fmt.Sprintf("removed sidebar_modules from %d users.setting", updatedUsers))
	}
	return nil
}

// removeUserSettingSidebarModules removes the sidebar_modules key from user setting JSON.
func removeUserSettingSidebarModules(settingJSON string) (string, bool, error) {
	settingJSON = strings.TrimSpace(settingJSON)
	if settingJSON == "" {
		return settingJSON, false, nil
	}

	var m map[string]any
	if err := common.Unmarshal([]byte(settingJSON), &m); err != nil {
		// Setting JSON is corrupted; avoid destructive updates.
		return settingJSON, false, nil
	}

	if _, ok := m["sidebar_modules"]; !ok {
		return settingJSON, false, nil
	}

	delete(m, "sidebar_modules")

	b, err := common.Marshal(m)
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}


