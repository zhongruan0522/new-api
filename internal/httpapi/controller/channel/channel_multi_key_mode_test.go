package channelcontroller

import (
	"encoding/json"
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/store/channel"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"net/http/httptest"
	"strings"
	"testing"
)

// setupMultiKeyModeTestDB 为多密钥模式校验测试准备内存 SQLite，
// 关闭内存缓存以避免 InitChannelCache 触碰全局渠道缓存，结束后恢复全局状态。
func setupMultiKeyModeTestDB(t *testing.T) {
	t.Helper()

	oldDB := dbstore.DB
	oldMemoryCacheEnabled := common.MemoryCacheEnabled

	common.MemoryCacheEnabled = false

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&channelstore.Channel{}, &channelstore.Ability{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	dbstore.DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		dbstore.DB = oldDB
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func performChannelJSONRequest(t *testing.T, handler gin.HandlerFunc, method string, body string) (bool, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", w.Body.String(), err)
	}
	return resp.Success, resp.Message
}

func seedMultiKeyChannel(t *testing.T) {
	t.Helper()

	channel := &channelstore.Channel{
		Id:     1,
		Type:   constant.ChannelTypeOpenAI,
		Name:   "multi-key",
		Key:    "sk-1\nsk-2",
		Models: "gpt-4o-mini",
		Group:  "default",
		Status: common.ChannelStatusEnabled,
		ChannelInfo: channelstore.ChannelInfo{
			IsMultiKey:   true,
			MultiKeySize: 2,
			MultiKeyMode: constant.MultiKeyModeRandom,
		},
	}
	if err := dbstore.DB.Create(channel).Error; err != nil {
		t.Fatalf("seed multi-key channel: %v", err)
	}
}

func TestUpdateChannelRejectsInvalidMultiKeyMode(t *testing.T) {
	setupMultiKeyModeTestDB(t)
	seedMultiKeyChannel(t)

	success, message := performChannelJSONRequest(t, UpdateChannel, "PUT",
		`{"id": 1, "multi_key_mode": "sticky"}`)
	if success {
		t.Fatal("expected invalid multi_key_mode to be rejected")
	}
	if message != i18n.MsgChannelMultiKeyModeInvalid {
		t.Fatalf("expected error key %q, got %q", i18n.MsgChannelMultiKeyModeInvalid, message)
	}

	reloaded, err := channelstore.GetChannelById(1, true)
	if err != nil {
		t.Fatalf("reload channel: %v", err)
	}
	if reloaded.ChannelInfo.MultiKeyMode != constant.MultiKeyModeRandom {
		t.Fatalf("rejected update must not change stored mode, got %q", reloaded.ChannelInfo.MultiKeyMode)
	}
}

func TestUpdateChannelSwitchesMultiKeyMode(t *testing.T) {
	setupMultiKeyModeTestDB(t)
	seedMultiKeyChannel(t)

	success, message := performChannelJSONRequest(t, UpdateChannel, "PUT",
		`{"id": 1, "multi_key_mode": "polling"}`)
	if !success {
		t.Fatalf("expected valid mode switch to succeed, got message %q", message)
	}

	reloaded, err := channelstore.GetChannelById(1, true)
	if err != nil {
		t.Fatalf("reload channel: %v", err)
	}
	if reloaded.ChannelInfo.MultiKeyMode != constant.MultiKeyModePolling {
		t.Fatalf("expected stored mode to switch to polling, got %q", reloaded.ChannelInfo.MultiKeyMode)
	}
	if !reloaded.ChannelInfo.IsMultiKey {
		t.Fatal("mode switch must not drop the multi-key flag")
	}
}

func TestAddChannelRejectsInvalidMultiKeyMode(t *testing.T) {
	setupMultiKeyModeTestDB(t)

	for _, mode := range []string{"", "sticky", "RANDOM"} {
		body := fmt.Sprintf(
			`{"mode": "multi_to_single", "multi_key_mode": %q, "channel": {"name": "x", "type": 1, "key": "sk-1\nsk-2", "models": "gpt-4o-mini", "group": "default"}}`,
			mode,
		)
		success, message := performChannelJSONRequest(t, AddChannel, "POST", body)
		if success {
			t.Fatalf("expected multi_to_single with multi_key_mode=%q to be rejected", mode)
		}
		if message != i18n.MsgChannelMultiKeyModeInvalid {
			t.Fatalf("multi_key_mode=%q: expected error key %q, got %q", mode, i18n.MsgChannelMultiKeyModeInvalid, message)
		}
	}

	var count int64
	if err := dbstore.DB.Model(&channelstore.Channel{}).Count(&count).Error; err != nil {
		t.Fatalf("count channels: %v", err)
	}
	if count != 0 {
		t.Fatalf("rejected requests must not create channels, got %d", count)
	}
}
