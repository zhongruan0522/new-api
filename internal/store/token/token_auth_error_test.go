package tokenstore_test

import (
	"errors"
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/token"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
)

func setupTokenAuthErrorTestDB(t *testing.T) {
	t.Helper()

	oldDB := dbstore.DB
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
	dbstore.InitCol()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&userstore.User{}, &tokenstore.Token{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	dbstore.DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		dbstore.DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		common.UsingSQLite = oldUsingSQLite
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.UsingMySQL = oldUsingMySQL
		dbstore.InitCol()
	})
}

func TestValidateUserTokenReturnsGenericInvalidForExhaustedToken(t *testing.T) {
	setupTokenAuthErrorTestDB(t)

	user := userstore.User{
		Id:          1,
		Username:    "token-user",
		Password:    "password123",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Token User",
		Group:       "default",
		AffCode:     "token-auth-aff-user",
	}
	if err := dbstore.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	token := tokenstore.Token{
		UserId:      user.Id,
		Key:         "exhaustedtoken",
		Status:      common.TokenStatusExhausted,
		Name:        "exhausted",
		ExpiredTime: -1,
		RemainQuota: 0,
	}
	if err := dbstore.DB.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}

	_, err := tokenstore.ValidateUserToken("exhaustedtoken")
	if !errors.Is(err, dbstore.ErrTokenInvalid) {
		t.Fatalf("ValidateUserToken err = %v, want %v", err, dbstore.ErrTokenInvalid)
	}
}
