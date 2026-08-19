package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"gorm.io/gorm"
)

// markerRemovedChatPlayground is the one-shot migration marker recorded in the
// options table once this cleanup has fully completed. Subsequent startups skip
// the otherwise-every-start scan over the users table. Delete the row to force a
// re-run (e.g. after importing legacy data).
const markerRemovedChatPlayground = "data_migration.removed_chat_playground.v1.done"

func cleanupRemovedChatPlaygroundData() error {
	if DB == nil {
		return nil
	}

	if isDataMigrationDone(markerRemovedChatPlayground) {
		return nil
	}

	start := time.Now()

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
	var scannedUsers int64
	var users []User
	result := DB.
		Select("id", "setting").
		Where("setting <> '' AND setting LIKE ?", "%sidebar_modules%").
		FindInBatches(&users, 200, func(tx *gorm.DB, _ int) error {
			for i := range users {
				scannedUsers++
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

	common.SysLog(fmt.Sprintf("chat playground cleanup: scanned %d users, updated %d users.setting in %s",
		scannedUsers, updatedUsers, time.Since(start).Round(time.Millisecond)))

	markDataMigrationDone(markerRemovedChatPlayground)
	return nil
}

// removeUserSettingSidebarModules removes the sidebar_modules key from user setting JSON.
func removeUserSettingSidebarModules(settingJSON string) (string, bool, error) {
	settingJSON = strings.TrimSpace(settingJSON)
	if settingJSON == "" {
		return settingJSON, false, nil
	}

	var m map[string]any
	if err := jsonx.Unmarshal([]byte(settingJSON), &m); err != nil {
		// Setting JSON is corrupted; avoid destructive updates.
		return settingJSON, false, nil
	}

	if _, ok := m["sidebar_modules"]; !ok {
		return settingJSON, false, nil
	}

	delete(m, "sidebar_modules")

	b, err := jsonx.Marshal(m)
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}
