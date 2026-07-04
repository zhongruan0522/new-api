package model

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type userUpdateCountingLogger struct {
	logger.Interface
	userUpdates *int64
}

func (l userUpdateCountingLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	normalizedSQL := strings.ToLower(strings.TrimSpace(sql))
	if strings.HasPrefix(normalizedSQL, "update") &&
		(strings.Contains(normalizedSQL, "`users`") || strings.Contains(normalizedSQL, `"users"`)) {
		atomic.AddInt64(l.userUpdates, 1)
	}
	l.Interface.Trace(ctx, begin, func() (string, int64) {
		return sql, rows
	}, err)
}

func setupUserBatchUpdateTestDB(t *testing.T, userUpdateCounter *int64) func() {
	t.Helper()

	oldDB := DB

	config := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if userUpdateCounter != nil {
		config.Logger = userUpdateCountingLogger{
			Interface:   logger.Default.LogMode(logger.Silent),
			userUpdates: userUpdateCounter,
		}
	}
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), config)
	require.NoError(t, err, "open sqlite test db")
	require.NoError(t, db.AutoMigrate(&User{}), "migrate sqlite test db")

	resetBatchUpdateStores()
	DB = db

	return func() {
		resetBatchUpdateStores()
		DB = oldDB
	}
}

func resetBatchUpdateStores() {
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateLocks[i].Lock()
		batchUpdateStores[i] = make(map[int]int)
		batchUpdateLocks[i].Unlock()
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
	require.NoError(t, DB.Create(&user).Error, "create user")
	return user
}

func loadUser(t *testing.T, id int) User {
	t.Helper()

	var user User
	require.NoError(t, DB.First(&user, id).Error, "load user")
	return user
}

func queueRelayBilling(userID int, cost int) {
	addNewRecord(BatchUpdateTypeUserQuota, userID, -cost)
	addNewRecord(BatchUpdateTypeUsedQuota, userID, cost)
	addNewRecord(BatchUpdateTypeRequestCount, userID, 1)
}

func TestBatchUpdateFlushesQuotaUsageAndRequestCountTogether(t *testing.T) {
	cleanup := setupUserBatchUpdateTestDB(t, nil)
	defer cleanup()

	user := createBatchUpdateTestUser(t, "batch-u1", "batch-aff-1", 10_000, 0, 0)
	queueRelayBilling(user.Id, 300)
	batchUpdate()

	got := loadUser(t, user.Id)
	assert.Equal(t, 9700, got.Quota)
	assert.Equal(t, 300, got.UsedQuota)
	assert.Equal(t, 1, got.RequestCount)
}

func TestBatchUpdateAccumulatesMultiplePendingUserUpdates(t *testing.T) {
	cleanup := setupUserBatchUpdateTestDB(t, nil)
	defer cleanup()

	user := createBatchUpdateTestUser(t, "batch-u2", "batch-aff-2", 10_000, 100, 5)
	queueRelayBilling(user.Id, 200)
	queueRelayBilling(user.Id, 150)
	batchUpdate()

	got := loadUser(t, user.Id)
	assert.Equal(t, 9650, got.Quota)
	assert.Equal(t, 450, got.UsedQuota)
	assert.Equal(t, 7, got.RequestCount)
}

func TestBatchUpdateDoesNotMixUsers(t *testing.T) {
	cleanup := setupUserBatchUpdateTestDB(t, nil)
	defer cleanup()

	alice := createBatchUpdateTestUser(t, "batch-alice", "batch-aff-a", 5_000, 0, 0)
	bob := createBatchUpdateTestUser(t, "batch-bob", "batch-aff-b", 8_000, 50, 2)

	queueRelayBilling(alice.Id, 100)
	queueRelayBilling(bob.Id, 200)
	batchUpdate()

	gotAlice := loadUser(t, alice.Id)
	gotBob := loadUser(t, bob.Id)

	assert.Equal(t, 4900, gotAlice.Quota)
	assert.Equal(t, 100, gotAlice.UsedQuota)
	assert.Equal(t, 1, gotAlice.RequestCount)
	assert.Equal(t, 7800, gotBob.Quota)
	assert.Equal(t, 250, gotBob.UsedQuota)
	assert.Equal(t, 3, gotBob.RequestCount)
}

func TestBatchUpdateWritesUserQuotaUsageAndRequestCountOncePerUser(t *testing.T) {
	var userUpdateCount int64
	cleanup := setupUserBatchUpdateTestDB(t, &userUpdateCount)
	defer cleanup()

	user := createBatchUpdateTestUser(t, "batch-count", "batch-aff-count", 10_000, 0, 0)
	atomic.StoreInt64(&userUpdateCount, 0)

	queueRelayBilling(user.Id, 200)
	queueRelayBilling(user.Id, 150)
	batchUpdate()

	got := loadUser(t, user.Id)
	assert.Equal(t, 9650, got.Quota)
	assert.Equal(t, 350, got.UsedQuota)
	assert.Equal(t, 2, got.RequestCount)
	assert.Equal(t, int64(1), atomic.LoadInt64(&userUpdateCount))
}
