package service

import (
	"github.com/zhongruan0522/new-api/model"
	"gorm.io/gorm"
)

func autoMigrateTargetMainSchema(db *gorm.DB) error {
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
		&model.Midjourney{},
		&model.TopUp{},
		&model.QuotaData{},
		&model.Task{},
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
