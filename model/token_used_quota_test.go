package model

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

func setupTokenUsedQuotaTestDB(t *testing.T) func() {
	t.Helper()

	oldDB := DB
	oldRedisEnabled := common.RedisEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&Token{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	DB = db
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false

	return func() {
		DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
	}
}

func TestUpdateTokenUsedQuotaLeavesRemainQuotaUnchanged(t *testing.T) {
	cleanup := setupTokenUsedQuotaTestDB(t)
	defer cleanup()

	token := Token{
		UserId:         1,
		Key:            "test-key",
		Name:           "unlimited",
		RemainQuota:    500,
		UsedQuota:      10,
		UnlimitedQuota: true,
		QuotaType:      0,
	}
	if err := DB.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := UpdateTokenUsedQuota(token.Id, token.Key, 120); err != nil {
		t.Fatalf("update token used quota: %v", err)
	}
	if err := UpdateTokenUsedQuota(token.Id, token.Key, -20); err != nil {
		t.Fatalf("refund token used quota: %v", err)
	}

	var got Token
	if err := DB.First(&got, token.Id).Error; err != nil {
		t.Fatalf("load token: %v", err)
	}
	if got.RemainQuota != 500 {
		t.Fatalf("remain quota changed: got %d, want 500", got.RemainQuota)
	}
	if got.UsedQuota != 110 {
		t.Fatalf("used quota = %d, want 110", got.UsedQuota)
	}
	if got.AccessedTime == 0 {
		t.Fatal("accessed time was not updated")
	}
}

func TestUpdateTokenUsedQuotaOnlyUpdatesTargetToken(t *testing.T) {
	cleanup := setupTokenUsedQuotaTestDB(t)
	defer cleanup()

	target := Token{
		UserId:         1,
		Key:            "target-key",
		Name:           "target",
		RemainQuota:    0,
		UsedQuota:      10,
		UnlimitedQuota: true,
		QuotaType:      0,
	}
	other := Token{
		UserId:         1,
		Key:            "other-key",
		Name:           "other",
		RemainQuota:    0,
		UsedQuota:      20,
		UnlimitedQuota: true,
		QuotaType:      0,
	}
	if err := DB.Create(&target).Error; err != nil {
		t.Fatalf("create target token: %v", err)
	}
	if err := DB.Create(&other).Error; err != nil {
		t.Fatalf("create other token: %v", err)
	}

	if err := UpdateTokenUsedQuota(target.Id, target.Key, 90); err != nil {
		t.Fatalf("update target token used quota: %v", err)
	}

	var gotTarget Token
	if err := DB.First(&gotTarget, target.Id).Error; err != nil {
		t.Fatalf("load target token: %v", err)
	}
	var gotOther Token
	if err := DB.First(&gotOther, other.Id).Error; err != nil {
		t.Fatalf("load other token: %v", err)
	}
	if gotTarget.UsedQuota != 100 {
		t.Fatalf("target used quota = %d, want 100", gotTarget.UsedQuota)
	}
	if gotOther.UsedQuota != 20 {
		t.Fatalf("other used quota = %d, want 20", gotOther.UsedQuota)
	}
}

func TestBatchUpdateTokenUsedQuotaLeavesRemainQuotaUnchanged(t *testing.T) {
	cleanup := setupTokenUsedQuotaTestDB(t)
	defer cleanup()
	common.BatchUpdateEnabled = true

	token := Token{
		UserId:         1,
		Key:            "batch-test-key",
		Name:           "unlimited-batch",
		RemainQuota:    700,
		UsedQuota:      30,
		UnlimitedQuota: true,
		QuotaType:      0,
	}
	if err := DB.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := UpdateTokenUsedQuota(token.Id, token.Key, 50); err != nil {
		t.Fatalf("queue token used quota update: %v", err)
	}
	if err := UpdateTokenUsedQuota(token.Id, token.Key, -10); err != nil {
		t.Fatalf("queue token used quota refund: %v", err)
	}
	batchUpdate()

	var got Token
	if err := DB.First(&got, token.Id).Error; err != nil {
		t.Fatalf("load token: %v", err)
	}
	if got.RemainQuota != 700 {
		t.Fatalf("remain quota changed: got %d, want 700", got.RemainQuota)
	}
	if got.UsedQuota != 70 {
		t.Fatalf("used quota = %d, want 70", got.UsedQuota)
	}
}
