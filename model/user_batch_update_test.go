package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

func setupUserBatchUpdateTestDB(t *testing.T) func() {
	t.Helper()

	oldDB := DB
	oldRedisEnabled := common.RedisEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	DB = db
	common.RedisEnabled = false
	common.BatchUpdateEnabled = true

	return func() {
		// DecreaseUserQuota updates quota cache asynchronously; wait before restoring RedisEnabled.
		time.Sleep(50 * time.Millisecond)
		DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
	}
}

func createBatchUpdateTestUser(t *testing.T, username, affCode string, quota, usedQuota, requestCount int) User {
	t.Helper()

	user := User{
		Username:     username,
		Password:     "password123",
		Role:         common.RoleCommonUser,
		Status:       common.UserStatusEnabled,
		AffCode:      affCode,
		Quota:        quota,
		UsedQuota:    usedQuota,
		RequestCount: requestCount,
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func loadUser(t *testing.T, id int) User {
	t.Helper()

	var user User
	if err := DB.First(&user, id).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	return user
}

// Simulates one relay billing cycle: deduct remaining quota and record usage + request count.
func simulateRelayBilling(t *testing.T, userID int, cost int) {
	t.Helper()

	if err := DecreaseUserQuota(userID, cost); err != nil {
		t.Fatalf("decrease user quota: %v", err)
	}
	UpdateUserUsedQuotaAndRequestCount(userID, cost)
}

func TestBatchUpdateFlushesQuotaUsageAndRequestCountTogether(t *testing.T) {
	cleanup := setupUserBatchUpdateTestDB(t)
	defer cleanup()

	user := createBatchUpdateTestUser(t, "batch-u1", "batch-aff-1", 10_000, 0, 0)
	simulateRelayBilling(t, user.Id, 300)
	batchUpdate()

	got := loadUser(t, user.Id)
	if got.Quota != 9700 {
		t.Fatalf("quota = %d, want 9700", got.Quota)
	}
	if got.UsedQuota != 300 {
		t.Fatalf("used_quota = %d, want 300", got.UsedQuota)
	}
	if got.RequestCount != 1 {
		t.Fatalf("request_count = %d, want 1", got.RequestCount)
	}
}

func TestBatchUpdateAccumulatesMultiplePendingUserUpdates(t *testing.T) {
	cleanup := setupUserBatchUpdateTestDB(t)
	defer cleanup()

	user := createBatchUpdateTestUser(t, "batch-u2", "batch-aff-2", 10_000, 100, 5)
	simulateRelayBilling(t, user.Id, 200)
	simulateRelayBilling(t, user.Id, 150)
	batchUpdate()

	got := loadUser(t, user.Id)
	if got.Quota != 9650 {
		t.Fatalf("quota = %d, want 9650", got.Quota)
	}
	if got.UsedQuota != 450 {
		t.Fatalf("used_quota = %d, want 450", got.UsedQuota)
	}
	if got.RequestCount != 7 {
		t.Fatalf("request_count = %d, want 7", got.RequestCount)
	}
}

func TestBatchUpdateDoesNotMixUsers(t *testing.T) {
	cleanup := setupUserBatchUpdateTestDB(t)
	defer cleanup()

	alice := createBatchUpdateTestUser(t, "batch-alice", "batch-aff-a", 5_000, 0, 0)
	bob := createBatchUpdateTestUser(t, "batch-bob", "batch-aff-b", 8_000, 50, 2)

	simulateRelayBilling(t, alice.Id, 100)
	simulateRelayBilling(t, bob.Id, 200)
	batchUpdate()

	gotAlice := loadUser(t, alice.Id)
	gotBob := loadUser(t, bob.Id)

	if gotAlice.Quota != 4900 || gotAlice.UsedQuota != 100 || gotAlice.RequestCount != 1 {
		t.Fatalf("alice: quota=%d used=%d requests=%d, want 4900/100/1",
			gotAlice.Quota, gotAlice.UsedQuota, gotAlice.RequestCount)
	}
	if gotBob.Quota != 7800 || gotBob.UsedQuota != 250 || gotBob.RequestCount != 3 {
		t.Fatalf("bob: quota=%d used=%d requests=%d, want 7800/250/3",
			gotBob.Quota, gotBob.UsedQuota, gotBob.RequestCount)
	}
}
