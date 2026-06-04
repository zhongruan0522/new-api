package model

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

func setupTokenAuthErrorTestDB(t *testing.T) {
	t.Helper()

	oldDB := DB
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldUsingSQLite := common.UsingSQLite
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldUsingMySQL := common.UsingMySQL

	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	common.UsingSQLite = true
	common.UsingPostgreSQL = false
	common.UsingMySQL = false
	initCol()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Token{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		common.UsingSQLite = oldUsingSQLite
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.UsingMySQL = oldUsingMySQL
		initCol()
	})
}

func TestValidateUserTokenReturnsGenericInvalidForExhaustedToken(t *testing.T) {
	setupTokenAuthErrorTestDB(t)

	user := User{
		Id:          1,
		Username:    "token-user",
		Password:    "password123",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Token User",
		Group:       "default",
		AffCode:     "token-auth-aff-user",
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	token := Token{
		UserId:      user.Id,
		Key:         "exhaustedtoken",
		Status:      common.TokenStatusExhausted,
		Name:        "exhausted",
		ExpiredTime: -1,
		RemainQuota: 0,
	}
	if err := DB.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}

	_, err := ValidateUserToken("exhaustedtoken")
	if !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("ValidateUserToken err = %v, want %v", err, ErrTokenInvalid)
	}
}
