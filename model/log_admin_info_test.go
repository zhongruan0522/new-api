package model

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

func setupLogAdminInfoTestDB(t *testing.T) {
	t.Helper()

	oldDB := DB
	oldLogDB := LOG_DB
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Log{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	DB = db
	LOG_DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
		LOG_DB = oldLogDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func TestRecordLogWithAdminInfoIsStrippedFromUserLogs(t *testing.T) {
	setupLogAdminInfoTestDB(t)

	user := &User{
		Id:       1,
		Username: "target-user",
		Status:   common.UserStatusEnabled,
		AffCode:  "log-admin-info-target",
	}
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	RecordLogWithAdminInfo(user.Id, LogTypeManage, "管理员强制禁用了用户的两步验证", map[string]interface{}{
		"admin_id":       99,
		"admin_username": "root-admin",
	})

	var logs []*Log
	if err := LOG_DB.Where("user_id = ?", user.Id).Find(&logs).Error; err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(logs))
	}
	if !strings.Contains(logs[0].Other, "admin_info") {
		t.Fatalf("stored log does not contain admin_info: %s", logs[0].Other)
	}

	formatUserLogs(logs, 0)
	if strings.Contains(logs[0].Other, "admin_info") || strings.Contains(logs[0].Other, "root-admin") {
		t.Fatalf("formatted user log leaked admin info: %s", logs[0].Other)
	}
	if strings.Contains(logs[0].Content, "99") || strings.Contains(logs[0].Content, "root-admin") {
		t.Fatalf("log content leaked admin identity: %s", logs[0].Content)
	}
}

func TestRecordErrorLogStoresAndFiltersUpstreamRequestId(t *testing.T) {
	setupLogAdminInfoTestDB(t)
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "http://example.test/v1/chat/completions", nil)
	c.Set("username", "target-user")
	c.Set(common.RequestIdKey, "local-id")
	c.Set(common.UpstreamRequestIdKey, "upstream-id")

	RecordErrorLog(c, 1, 0, "gpt-test", "token", "upstream failed", 0, 10, false, "default", nil)

	var stored Log
	if err := LOG_DB.First(&stored).Error; err != nil {
		t.Fatalf("query stored log: %v", err)
	}
	if stored.RequestId != "local-id" {
		t.Fatalf("stored request id = %q, want local-id", stored.RequestId)
	}
	if stored.UpstreamRequestId != "upstream-id" {
		t.Fatalf("stored upstream request id = %q, want upstream-id", stored.UpstreamRequestId)
	}

	logs, total, err := GetAllLogs(LogTypeUnknown, 0, 0, "", "", "", 0, 20, 0, "", "", "upstream-id", "", "", "", "")
	if err != nil {
		t.Fatalf("GetAllLogs error = %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].UpstreamRequestId != "upstream-id" {
		t.Fatalf("filtered logs total=%d logs=%#v, want one upstream-id log", total, logs)
	}
}

func TestRecordErrorLogTruncatesStoredContent(t *testing.T) {
	setupLogAdminInfoTestDB(t)
	gin.SetMode(gin.TestMode)
	oldDebug := common.DebugEnabled
	common.DebugEnabled = false
	t.Cleanup(func() {
		common.DebugEnabled = oldDebug
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "http://example.test/v1/chat/completions", nil)
	c.Set("username", "target-user")

	content := strings.Repeat("x", common.LocalLogContentLimit+128)
	RecordErrorLog(c, 1, 0, "gpt-test", "token", content, 0, 10, false, "default", nil)

	var stored Log
	if err := LOG_DB.First(&stored).Error; err != nil {
		t.Fatalf("query stored log: %v", err)
	}
	if !strings.Contains(stored.Content, "[truncated") {
		t.Fatalf("stored content was not truncated: length=%d content=%q", len(stored.Content), stored.Content)
	}
	if strings.Contains(stored.Content, strings.Repeat("x", common.LocalLogContentLimit+1)) {
		t.Fatal("stored content includes more than the allowed preview")
	}
}
