package topupstore

import (
	"errors"
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/log"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func setupTopUpCallbackTestDB(t *testing.T) {
	t.Helper()

	oldDB := dbstore.DB
	oldLogDB := dbstore.LOG_DB
	oldQuotaPerUnit := common.QuotaPerUnit
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&userstore.User{}, &TopUp{}, &logstore.Log{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	dbstore.DB = db
	dbstore.LOG_DB = db
	common.QuotaPerUnit = 100
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		dbstore.DB = oldDB
		dbstore.LOG_DB = oldLogDB
		common.QuotaPerUnit = oldQuotaPerUnit
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func createTopUpCallbackUser(t *testing.T, id int) {
	t.Helper()

	user := &userstore.User{
		Id:       id,
		Username: fmt.Sprintf("topup-callback-user-%d", id),
		Status:   common.UserStatusEnabled,
		Quota:    0,
	}
	if err := dbstore.DB.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func createTopUpCallbackTopUp(t *testing.T, tradeNo string, provider string, method string, money float64) {
	t.Helper()

	topUp := &TopUp{
		UserId:          1,
		Amount:          2,
		Money:           money,
		TradeNo:         tradeNo,
		PaymentMethod:   method,
		PaymentProvider: provider,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		t.Fatalf("create topup: %v", err)
	}
}

func topUpCallbackStatus(t *testing.T, tradeNo string) string {
	t.Helper()

	topUp := GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		t.Fatalf("topup %s not found", tradeNo)
	}
	return topUp.Status
}

// TestRechargeCreditsPendingOrder 验证 Recharge 对 pending 订单成功入账。
func TestRechargeCreditsPendingOrder(t *testing.T) {
	setupTopUpCallbackTestDB(t)
	createTopUpCallbackUser(t, 1)
	createTopUpCallbackTopUp(t, "recharge-ok", PaymentProviderStripe, PaymentMethodStripe, 9.99)

	if err := Recharge("recharge-ok", "cus_test_123"); err != nil {
		t.Fatalf("Recharge error = %v", err)
	}
	if got := topUpCallbackStatus(t, "recharge-ok"); got != common.TopUpStatusSuccess {
		t.Fatalf("topup status = %s, want %s", got, common.TopUpStatusSuccess)
	}
	var user userstore.User
	if err := dbstore.DB.Select("quota", "stripe_customer").Where("id = ?", 1).First(&user).Error; err != nil {
		t.Fatalf("get user: %v", err)
	}
	// Recharge 的入账额度 = Money * QuotaPerUnit = 9.99 * 100。
	if user.Quota != 999 {
		t.Fatalf("user quota = %d, want 999", user.Quota)
	}
	if user.StripeCustomer != "cus_test_123" {
		t.Fatalf("stripe_customer = %q, want %q", user.StripeCustomer, "cus_test_123")
	}
}

// TestRechargeDuplicateReturnsStatusInvalidError 验证对已成功订单再次调用 Recharge
// 返回可被 errors.Is 判断的 ErrTopUpStatusInvalid sentinel error。
// 这是 StripeWebhook 侧区分"重复投递已处理"与"真实入账失败"的依据。
func TestRechargeDuplicateReturnsStatusInvalidError(t *testing.T) {
	setupTopUpCallbackTestDB(t)
	createTopUpCallbackUser(t, 1)
	createTopUpCallbackTopUp(t, "recharge-dup", PaymentProviderStripe, PaymentMethodStripe, 9.99)

	if err := Recharge("recharge-dup", "cus_test_123"); err != nil {
		t.Fatalf("first Recharge error = %v", err)
	}

	err := Recharge("recharge-dup", "cus_test_123")
	if err == nil {
		t.Fatal("expected error on duplicate Recharge, got nil")
	}
	if !errors.Is(err, ErrTopUpStatusInvalid) {
		t.Fatalf("Recharge error = %v, want ErrTopUpStatusInvalid (errors.Is)", err)
	}

	// 重复投递不得二次加款。
	var user userstore.User
	if err := dbstore.DB.Select("quota").Where("id = ?", 1).First(&user).Error; err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Quota != 999 {
		t.Fatalf("user quota = %d, want 999 (no double credit)", user.Quota)
	}
}

// TestCompleteEpayTopUpDuplicateReturnsStatusInvalidError 验证 CompleteEpayTopUp
// 的幂等性：已成功订单再次调用返回 ErrTopUpStatusInvalid，且不重复加款。
func TestCompleteEpayTopUpDuplicateReturnsStatusInvalidError(t *testing.T) {
	setupTopUpCallbackTestDB(t)
	createTopUpCallbackUser(t, 1)
	createTopUpCallbackTopUp(t, "epay-dup", PaymentProviderEpay, "alipay", 9.99)

	if err := CompleteEpayTopUp("epay-dup", "alipay", "9.99"); err != nil {
		t.Fatalf("first CompleteEpayTopUp error = %v", err)
	}

	err := CompleteEpayTopUp("epay-dup", "alipay", "9.99")
	if err == nil {
		t.Fatal("expected error on duplicate CompleteEpayTopUp, got nil")
	}
	if !errors.Is(err, ErrTopUpStatusInvalid) {
		t.Fatalf("CompleteEpayTopUp error = %v, want ErrTopUpStatusInvalid (errors.Is)", err)
	}

	var user userstore.User
	if err := dbstore.DB.Select("quota").Where("id = ?", 1).First(&user).Error; err != nil {
		t.Fatalf("get user: %v", err)
	}
	// CompleteEpayTopUp 的入账额度 = Amount * QuotaPerUnit = 2 * 100。
	if user.Quota != 200 {
		t.Fatalf("user quota = %d, want 200 (no double credit)", user.Quota)
	}
}
