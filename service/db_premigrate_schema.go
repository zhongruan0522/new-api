package service

import (
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"gorm.io/gorm"
)

func autoMigrateTargetMainSchema(db *gorm.DB) error {
	// 同 model.cleanupPrefillGroupLegacyIndex 的说明，pre-migrate 路径也需要清理
	if common.UsingPostgreSQL {
		_ = db.Exec(`ALTER TABLE "prefill_groups" DROP CONSTRAINT IF EXISTS "uni_prefill_groups_name"`).Error
		_ = db.Exec(`DROP INDEX IF EXISTS "uni_prefill_groups_name"`).Error
	}
	return db.AutoMigrate(
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
	)
}

func autoMigrateTargetLogSchema(db *gorm.DB) error {
	return db.AutoMigrate(&model.Log{})
}
