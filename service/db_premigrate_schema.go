package service

import (
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"gorm.io/gorm"
)

func autoMigrateTargetMainSchema(db *gorm.DB) error {
	// 同 model.cleanupLegacyUniqueIndexes 的说明，pre-migrate 路径也需要清理
	if common.UsingPostgreSQL {
		model.CleanupLegacyUniqueConstraints(db, "prefill_groups", "name", []string{"uni_prefill_groups_name", "idx_prefill_groups_name"})
		model.CleanupLegacyUniqueConstraints(db, "models", "model_name", []string{"uni_models_model_name", "idx_models_model_name"})
		model.CleanupLegacyUniqueConstraints(db, "vendors", "name", []string{"uni_vendors_name", "idx_vendors_name"})
		model.CleanupLegacyUniqueConstraints(db, "passkey_credentials", "user_id", []string{"uni_passkey_credentials_user_id", "idx_passkey_credentials_user_id"})
	}
	if common.UsingSQLite {
		model.CleanupLegacyUniqueConstraints(db, "passkey_credentials", "user_id", []string{"uni_passkey_credentials_user_id", "idx_passkey_credentials_user_id"})
	}
	if common.UsingMySQL {
		model.CleanupLegacyUniqueConstraints(db, "passkey_credentials", "user_id", []string{"uni_passkey_credentials_user_id", "idx_passkey_credentials_user_id"})
	}
	if err := db.AutoMigrate(
		&model.Channel{},
		&model.Token{},
		&model.User{},
		&model.PasskeyCredential{},
		&model.Option{},
		&model.Redemption{},
		&model.Ability{},
		&model.Log{},
		&model.StoredImage{},
		&model.StoredVideo{},
		&model.TopUp{},
		&model.QuotaData{},
		&model.Model{},
		&model.Vendor{},
		&model.PrefillGroup{},
		&model.Setup{},
		&model.TwoFA{},
		&model.TwoFABackupCode{},
		&model.Checkin{},
	); err != nil {
		return err
	}
	model.CleanupRemovedQuotaDataCacheStats(db)
	return nil
}

func autoMigrateTargetLogSchema(db *gorm.DB) error {
	return db.AutoMigrate(&model.Log{})
}
