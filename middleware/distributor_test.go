package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/model"
	"gorm.io/gorm"
)

func setupDistributorTestDB(t *testing.T) {
	t.Helper()

	oldDB := model.DB
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled
	oldUsingSQLite := common.UsingSQLite
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldUsingMySQL := common.UsingMySQL

	common.RedisEnabled = false
	common.MemoryCacheEnabled = true
	common.BatchUpdateEnabled = false
	common.UsingSQLite = true
	common.UsingPostgreSQL = false
	common.UsingMySQL = false

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&model.Channel{}, &model.Ability{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	model.DB = db
	model.InitChannelCache()

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
		common.UsingSQLite = oldUsingSQLite
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.UsingMySQL = oldUsingMySQL
		if oldMemoryCacheEnabled && oldDB != nil {
			model.InitChannelCache()
		}
	})
}

func createDistributorTestChannel(t *testing.T, supportSubscription bool) model.Channel {
	t.Helper()

	priority := int64(0)
	weight := uint(1)
	channel := model.Channel{
		Type:                constant.ChannelTypeOpenAI,
		Key:                 "test-key",
		Status:              common.ChannelStatusEnabled,
		Name:                "test-channel",
		Models:              "claude-haiku-4-5-20251001",
		Group:               "Coding",
		Priority:            &priority,
		Weight:              &weight,
		SupportSubscription: supportSubscription,
	}
	if err := model.DB.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if err := channel.AddAbilities(model.DB); err != nil {
		t.Fatalf("create abilities: %v", err)
	}
	return channel
}

func performDistributorTestRequest(billingMode string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 1)
		common.SetContextKey(c, constant.ContextKeyUserGroup, "Coding")
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "Coding")
		common.SetContextKey(c, constant.ContextKeySubscriptionActive, true)
		common.SetContextKey(c, constant.ContextKeySubscriptionId, 100)
		common.SetContextKey(c, constant.ContextKeyTokenBillingMode, billingMode)
		common.SetContextKey(c, constant.ContextKeyTokenModelLimitEnabled, false)
		c.Next()
	})
	router.Use(Distribute())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"channel_id": common.GetContextKeyInt(c, constant.ContextKeyChannelId),
		})
	})

	body := bytes.NewBufferString(`{"model":"claude-haiku-4-5-20251001","messages":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestDistributeWalletTokenDoesNotRequireSubscriptionChannel(t *testing.T) {
	setupDistributorTestDB(t)
	channel := createDistributorTestChannel(t, false)
	model.InitChannelCache()

	recorder := performDistributorTestRequest(common.TokenBillingModeWallet)
	if recorder.Code != http.StatusOK {
		t.Fatalf("wallet request status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(fmt.Sprintf(`"channel_id":%d`, channel.Id))) {
		t.Fatalf("wallet request did not select ordinary channel %d, body = %s", channel.Id, recorder.Body.String())
	}
}

func TestDistributeSubscriptionTokenRequiresSubscriptionChannel(t *testing.T) {
	setupDistributorTestDB(t)
	createDistributorTestChannel(t, false)
	model.InitChannelCache()

	recorder := performDistributorTestRequest(common.TokenBillingModeSubscription)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("subscription request status = %d, want %d, body = %s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("无可用渠道")) {
		t.Fatalf("subscription request error should report no available channel, body = %s", recorder.Body.String())
	}
}
