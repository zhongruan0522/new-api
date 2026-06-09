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
	oldLogDB := LOG_DB
	oldRedisEnabled := common.RedisEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&Token{}, &Log{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	DB = db
	LOG_DB = db
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false

	return func() {
		DB = oldDB
		LOG_DB = oldLogDB
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

func TestApplyHistoricalTokensUsedQuotaUsesConsumeLogsForUnlimitedTokens(t *testing.T) {
	cleanup := setupTokenUsedQuotaTestDB(t)
	defer cleanup()

	unlimited := Token{
		UserId:         1,
		Key:            "historical-key",
		Name:           "historical",
		RemainQuota:    0,
		UsedQuota:      0,
		UnlimitedQuota: true,
		QuotaType:      0,
	}
	limited := Token{
		UserId:         1,
		Key:            "limited-key",
		Name:           "limited",
		RemainQuota:    100,
		UsedQuota:      5,
		UnlimitedQuota: false,
		QuotaType:      1,
	}
	if err := DB.Create(&unlimited).Error; err != nil {
		t.Fatalf("create unlimited token: %v", err)
	}
	if err := DB.Create(&limited).Error; err != nil {
		t.Fatalf("create limited token: %v", err)
	}

	logs := []Log{
		{TokenId: unlimited.Id, Type: LogTypeConsume, Quota: 120},
		{TokenId: unlimited.Id, Type: LogTypeConsume, Quota: 80},
		{TokenId: unlimited.Id, Type: LogTypeError, Quota: 999},
		{TokenId: limited.Id, Type: LogTypeConsume, Quota: 300},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatalf("create logs: %v", err)
	}

	tokens := []*Token{&unlimited, &limited}
	if err := ApplyHistoricalTokensUsedQuota(tokens); err != nil {
		t.Fatalf("apply historical token used quota: %v", err)
	}

	if unlimited.UsedQuota != 200 {
		t.Fatalf("unlimited used quota = %d, want 200", unlimited.UsedQuota)
	}
	if limited.UsedQuota != 5 {
		t.Fatalf("limited used quota = %d, want 5", limited.UsedQuota)
	}
}

func TestApplyHistoricalTokensUsedQuotaKeepsLargerStoredValue(t *testing.T) {
	cleanup := setupTokenUsedQuotaTestDB(t)
	defer cleanup()

	token := Token{
		UserId:         1,
		Key:            "stored-key",
		Name:           "stored",
		RemainQuota:    0,
		UsedQuota:      500,
		UnlimitedQuota: true,
		QuotaType:      0,
	}
	if err := DB.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := LOG_DB.Create(&Log{TokenId: token.Id, Type: LogTypeConsume, Quota: 100}).Error; err != nil {
		t.Fatalf("create log: %v", err)
	}

	if err := ApplyHistoricalTokenUsedQuota(&token); err != nil {
		t.Fatalf("apply historical token used quota: %v", err)
	}
	if token.UsedQuota != 500 {
		t.Fatalf("used quota = %d, want stored value 500", token.UsedQuota)
	}
}
