package service

import "github.com/QuantumNous/new-api/model"

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
	gormTableCopyStep[model.Midjourney]{name: "midjourneys", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.TopUp]{name: "top_ups", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.QuotaData]{name: "quota_data", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Task]{name: "tasks", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Model]{name: "models", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Vendor]{name: "vendors", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.PrefillGroup]{name: "prefill_groups", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Setup]{name: "setups", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.TwoFA]{name: "two_fas", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.TwoFABackupCode]{name: "two_fa_backup_codes", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.Checkin]{name: "checkins", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.CustomOAuthProvider]{name: "custom_oauth_providers", batchSize: dbPreMigrateBatchDefault},
	gormTableCopyStep[model.UserOAuthBinding]{name: "user_oauth_bindings", batchSize: dbPreMigrateBatchDefault},
}

var dbPreMigrateLogStep = gormTableCopyStep[model.Log]{name: "logs", batchSize: dbPreMigrateBatchLog}
