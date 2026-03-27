package model

import (
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

// cleanupRemovedOAuth drops columns and tables left behind after removing
// Telegram login, Discord OAuth and custom OAuth providers.
// This migration is idempotent: it checks before acting and tolerates
// "column does not exist" / "table does not exist" errors.
func cleanupRemovedOAuth() {
	if DB == nil {
		return
	}

	migrator := DB.Migrator()

	// ------------------------------------------------------------------
	// 1) Drop columns from the users table
	// ------------------------------------------------------------------
	for _, col := range []string{"discord_id", "telegram_id"} {
		if migrator.HasColumn(&User{}, col) {
			if err := migrator.DropColumn(&User{}, col); err != nil {
				common.SysError("failed to drop users." + col + ": " + err.Error())
			} else {
				common.SysLog("dropped column users." + col)
			}
		}
	}

	// ------------------------------------------------------------------
	// 2) Drop custom_oauth_providers table
	// ------------------------------------------------------------------
	dropTableIfExists(migrator, "custom_oauth_providers")

	// ------------------------------------------------------------------
	// 3) Drop user_oauth_bindings table
	// ------------------------------------------------------------------
	dropTableIfExists(migrator, "user_oauth_bindings")

	// ------------------------------------------------------------------
	// 4) Remove related option rows that are no longer loaded
	// ------------------------------------------------------------------
	for _, key := range []string{
		"TelegramOAuthEnabled",
		"TelegramBotToken",
		"TelegramBotName",
		"discord.enabled",
		"discord.client_id",
		"discord.client_secret",
	} {
		if res := DB.Delete(&Option{Key: key}); res.Error != nil {
			common.SysError("failed to remove option " + key + ": " + res.Error.Error())
		} else if res.RowsAffected > 0 {
			common.SysLog("removed obsolete option: " + key)
		}
	}
}

func dropTableIfExists(migrator gorm.Migrator, tableName string) {
	if migrator.HasTable(tableName) {
		if err := migrator.DropTable(tableName); err != nil {
			common.SysError("failed to drop table " + tableName + ": " + err.Error())
		} else {
			common.SysLog("dropped table " + tableName)
		}
	}
}
