package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/model"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"gorm.io/gorm"
)

func setupBillingSessionTestDB(t *testing.T) {
	t.Helper()

	oldDB := model.DB
	oldRedisEnabled := common.RedisEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled
	oldUsingSQLite := common.UsingSQLite
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldUsingMySQL := common.UsingMySQL

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserSubscription{}, &model.SubscriptionBill{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	model.DB = db
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.UsingSQLite = true
	common.UsingPostgreSQL = false
	common.UsingMySQL = false

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
		common.UsingSQLite = oldUsingSQLite
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.UsingMySQL = oldUsingMySQL
	})
}

func newBillingSessionTestContext(mode string, subscriptionId int) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(c, constant.ContextKeySubscriptionActive, subscriptionId > 0)
	common.SetContextKey(c, constant.ContextKeySubscriptionId, subscriptionId)
	common.SetContextKey(c, constant.ContextKeyTokenBillingMode, mode)
	return c
}

func createBillingSessionTestUser(t *testing.T, quota int) int {
	t.Helper()
	user := model.User{
		Username: "billing-test-user",
		Password: "password",
		Quota:    quota,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.Id
}

func createBillingSessionTestSubscription(t *testing.T, userId int, windowQuota int, totalQuota int) int {
	t.Helper()
	now := common.GetTimestamp()
	sub := model.UserSubscription{
		UserId:             userId,
		PlanId:             1,
		PlanName:           "test plan",
		Price:              0,
		Status:             common.SubscriptionStatusActive,
		StartsAt:           now - 60,
		ExpiresAt:          now + 3600,
		NextResetAt:        now + 1800,
		DurationCount:      1,
		DurationUnit:       "day",
		TotalQuota:         totalQuota,
		ResetQuota:         windowQuota,
		ResetIntervalCount: 1,
		ResetIntervalUnit:  "day",
		WindowUsedQuota:    0,
		UsedTotalQuota:     0,
		CreatedTime:        now,
		UpdatedTime:        now,
	}
	if err := model.DB.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	return sub.Id
}

func getBillingSessionTestUserQuota(t *testing.T, userId int) int {
	t.Helper()
	var quota int
	if err := model.DB.Model(&model.User{}).Where("id = ?", userId).Select("quota").Scan(&quota).Error; err != nil {
		t.Fatalf("read user quota: %v", err)
	}
	return quota
}

func getBillingSessionTestSubscription(t *testing.T, subscriptionId int) model.UserSubscription {
	t.Helper()
	var sub model.UserSubscription
	if err := model.DB.First(&sub, "id = ?", subscriptionId).Error; err != nil {
		t.Fatalf("read subscription: %v", err)
	}
	return sub
}

func getBillingSessionTestBills(t *testing.T, subscriptionId int) []model.SubscriptionBill {
	t.Helper()
	var bills []model.SubscriptionBill
	if err := model.DB.Where("subscription_id = ?", subscriptionId).Order("id asc").Find(&bills).Error; err != nil {
		t.Fatalf("read subscription bills: %v", err)
	}
	return bills
}

func newBillingSessionRelayInfo(userId int, mode string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		UserId:           userId,
		TokenQuotaType:   0,
		TokenUnlimited:   true,
		TokenBillingMode: mode,
		RequestId:        "billing-test-request",
	}
}

func TestBillingSessionWalletModeIgnoresActiveSubscription(t *testing.T) {
	setupBillingSessionTestDB(t)
	userId := createBillingSessionTestUser(t, 1000)
	subscriptionId := createBillingSessionTestSubscription(t, userId, 500, 500)

	info := newBillingSessionRelayInfo(userId, common.TokenBillingModeWallet)
	session, apiErr := NewBillingSession(newBillingSessionTestContext(common.TokenBillingModeWallet, subscriptionId), info, 200)
	if apiErr != nil {
		t.Fatalf("NewBillingSession returned error: %v", apiErr)
	}
	if session == nil {
		t.Fatalf("expected billing session")
	}
	if info.BillingSource != BillingSourceWallet {
		t.Fatalf("billing source = %q, want %q", info.BillingSource, BillingSourceWallet)
	}
	if quota := getBillingSessionTestUserQuota(t, userId); quota != 800 {
		t.Fatalf("user quota after wallet preconsume = %d, want 800", quota)
	}
	if sub := getBillingSessionTestSubscription(t, subscriptionId); sub.UsedTotalQuota != 0 || sub.WindowUsedQuota != 0 {
		t.Fatalf("subscription unexpectedly consumed: %+v", sub)
	}
}

func TestBillingSessionSubscriptionModeUsesOnlySubscription(t *testing.T) {
	setupBillingSessionTestDB(t)
	userId := createBillingSessionTestUser(t, 0)
	subscriptionId := createBillingSessionTestSubscription(t, userId, 500, 500)

	info := newBillingSessionRelayInfo(userId, common.TokenBillingModeSubscription)
	session, apiErr := NewBillingSession(newBillingSessionTestContext(common.TokenBillingModeSubscription, subscriptionId), info, 200)
	if apiErr != nil {
		t.Fatalf("NewBillingSession returned error: %v", apiErr)
	}
	if session == nil {
		t.Fatalf("expected billing session")
	}
	if info.BillingSource != BillingSourceSubscription {
		t.Fatalf("billing source = %q, want %q", info.BillingSource, BillingSourceSubscription)
	}
	if quota := getBillingSessionTestUserQuota(t, userId); quota != 0 {
		t.Fatalf("user quota after subscription preconsume = %d, want 0", quota)
	}
	if sub := getBillingSessionTestSubscription(t, subscriptionId); sub.UsedTotalQuota != 200 || sub.WindowUsedQuota != 200 {
		t.Fatalf("subscription used quota = total %d window %d, want 200/200", sub.UsedTotalQuota, sub.WindowUsedQuota)
	}
}

func TestBillingSessionSubscriptionThenWalletSplitsAndRefunds(t *testing.T) {
	setupBillingSessionTestDB(t)
	userId := createBillingSessionTestUser(t, 1000)
	subscriptionId := createBillingSessionTestSubscription(t, userId, 300, 300)

	info := newBillingSessionRelayInfo(userId, common.TokenBillingModeSubscriptionThen)
	session, apiErr := NewBillingSession(newBillingSessionTestContext(common.TokenBillingModeSubscriptionThen, subscriptionId), info, 700)
	if apiErr != nil {
		t.Fatalf("NewBillingSession returned error: %v", apiErr)
	}
	if session == nil {
		t.Fatalf("expected billing session")
	}
	if info.BillingSource != BillingSourceSubscriptionWallet {
		t.Fatalf("billing source = %q, want %q", info.BillingSource, BillingSourceSubscriptionWallet)
	}
	if quota := getBillingSessionTestUserQuota(t, userId); quota != 600 {
		t.Fatalf("user quota after split preconsume = %d, want 600", quota)
	}
	if sub := getBillingSessionTestSubscription(t, subscriptionId); sub.UsedTotalQuota != 300 || sub.WindowUsedQuota != 300 {
		t.Fatalf("subscription used quota after split preconsume = total %d window %d, want 300/300", sub.UsedTotalQuota, sub.WindowUsedQuota)
	}

	if err := session.Settle(250); err != nil {
		t.Fatalf("Settle returned error: %v", err)
	}
	if info.BillingSource != BillingSourceSubscription {
		t.Fatalf("billing source after refund = %q, want %q", info.BillingSource, BillingSourceSubscription)
	}
	if quota := getBillingSessionTestUserQuota(t, userId); quota != 1000 {
		t.Fatalf("user quota after settle refund = %d, want 1000", quota)
	}
	if sub := getBillingSessionTestSubscription(t, subscriptionId); sub.UsedTotalQuota != 250 || sub.WindowUsedQuota != 250 {
		t.Fatalf("subscription used quota after settle refund = total %d window %d, want 250/250", sub.UsedTotalQuota, sub.WindowUsedQuota)
	}
}

func TestBillingSessionSubscriptionThenWalletSettleRecordsSettleBill(t *testing.T) {
	setupBillingSessionTestDB(t)
	userId := createBillingSessionTestUser(t, 1000)
	subscriptionId := createBillingSessionTestSubscription(t, userId, 500, 500)

	info := newBillingSessionRelayInfo(userId, common.TokenBillingModeSubscriptionThen)
	session, apiErr := NewBillingSession(newBillingSessionTestContext(common.TokenBillingModeSubscriptionThen, subscriptionId), info, 200)
	if apiErr != nil {
		t.Fatalf("NewBillingSession returned error: %v", apiErr)
	}
	if err := session.Settle(450); err != nil {
		t.Fatalf("Settle returned error: %v", err)
	}

	bills := getBillingSessionTestBills(t, subscriptionId)
	if len(bills) != 2 {
		t.Fatalf("bill count = %d, want 2", len(bills))
	}
	if bills[0].Event != common.SubscriptionBillEventPreConsume || bills[0].Quota != 200 {
		t.Fatalf("preconsume bill = event %q quota %d, want %q/200", bills[0].Event, bills[0].Quota, common.SubscriptionBillEventPreConsume)
	}
	if bills[1].Event != common.SubscriptionBillEventSettle || bills[1].Quota != 250 {
		t.Fatalf("settle bill = event %q quota %d, want %q/250", bills[1].Event, bills[1].Quota, common.SubscriptionBillEventSettle)
	}
}

func TestBillingSessionSubscriptionModeRequiresActiveSubscription(t *testing.T) {
	setupBillingSessionTestDB(t)
	userId := createBillingSessionTestUser(t, 1000)

	info := newBillingSessionRelayInfo(userId, common.TokenBillingModeSubscription)
	if _, apiErr := NewBillingSession(newBillingSessionTestContext(common.TokenBillingModeSubscription, 0), info, 200); apiErr == nil {
		t.Fatalf("expected subscription mode without active subscription to fail")
	}
}
