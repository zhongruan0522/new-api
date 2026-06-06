package model

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

func setupUserCacheInvalidationTestDB(t *testing.T) {
	t.Helper()

	oldDB := DB
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		if DB != nil {
			if sqlDB, err := DB.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
		DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Token{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
}

func createCacheInvalidationUserWithToken(t *testing.T, username string) User {
	t.Helper()

	user := User{
		Username: username,
		Password: "password",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	token := Token{
		UserId: user.Id,
		Key:    username + "-token",
		Name:   username + "-token",
		Status: common.TokenStatusEnabled,
	}
	if err := DB.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
	return user
}

func TestHardDeleteUserByIdInvalidatesTokenCacheWithoutRedis(t *testing.T) {
	setupUserCacheInvalidationTestDB(t)
	user := createCacheInvalidationUserWithToken(t, "cache-hard-delete-user")
	if err := HardDeleteUserById(user.Id); err != nil {
		t.Fatalf("HardDeleteUserById: %v", err)
	}

	var count int64
	if err := DB.Unscoped().Model(&User{}).Where("id = ?", user.Id).Count(&count).Error; err != nil {
		t.Fatalf("count user: %v", err)
	}
	if count != 0 {
		t.Fatalf("deleted user count = %d, want 0", count)
	}
}

func TestSoftDeleteUserInvalidatesTokenCacheWithoutRedis(t *testing.T) {
	setupUserCacheInvalidationTestDB(t)
	user := createCacheInvalidationUserWithToken(t, "cache-soft-delete-user")
	if err := user.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var count int64
	if err := DB.Model(&User{}).Where("id = ?", user.Id).Count(&count).Error; err != nil {
		t.Fatalf("count user: %v", err)
	}
	if count != 0 {
		t.Fatalf("active user count = %d, want 0", count)
	}
}
