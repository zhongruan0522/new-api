package service

import "github.com/NookMux/NookMux/model"

var dbPreMigrateMainSteps = []dbPreMigrateStep{
	gormTableCopyStep[model.User]{name: "users", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Option]{name: "options", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Channel]{name: "channels", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Token]{name: "tokens", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.PasskeyCredential]{name: "passkey_credentials", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Redemption]{name: "redemptions", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Ability]{name: "abilities", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.StoredImage]{name: "stored_images", batchSize: dbPreMigrateBatchBlob},
	gormTableCopyStep[model.StoredVideo]{name: "stored_videos", batchSize: dbPreMigrateBatchBlob},
	gormTableCopyStep[model.TopUp]{name: "top_ups", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.QuotaData]{name: "quota_data", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Model]{name: "models", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Vendor]{name: "vendors", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.PrefillGroup]{name: "prefill_groups", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Setup]{name: "setups", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.TwoFA]{name: "two_fas", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.TwoFABackupCode]{name: "two_fa_backup_codes", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Checkin]{name: "checkins", batchSize: dbPreMigrateBatchDefault},
	// Tables kept in sync with model.migrateDB()'s AutoMigrate list. They were
	// previously omitted here, which would drop their data on a pre-/same-type DB
	// migration. MiniMaxVoice uses a custom TableName ("minimax_voices").
	gormTableCopyStep[model.Ticket]{name: "tickets", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.TicketEntry]{name: "ticket_entries", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.DynamicRatioRule]{name: "dynamic_ratio_rules", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.AuditLog]{name: "audit_logs", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.MiniMaxVoice]{name: "minimax_voices", batchSize: dbPreMigrateBatchDefault},
}

var dbPreMigrateLogStep = gormTableCopyStep[model.Log]{name: "logs", batchSize: dbPreMigrateBatchLog}
