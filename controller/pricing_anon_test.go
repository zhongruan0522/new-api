package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/model"
)

func setupPricingTestDB(t *testing.T) {
	t.Helper()

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&model.Ability{}, &model.Channel{}, &model.Model{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

// 匿名访问 pricing 时 enable_groups 不应携带内部组名（exists=false 路径），
// 登录访问时保留。防止公开定价页向匿名调用者暴露内部分组分类法。
func TestGetPricingAnonymousStripsEnableGroups(t *testing.T) {
	setupPricingTestDB(t)
	gin.SetMode(gin.TestMode)

	ch := model.Channel{
		Id:     1,
		Status: common.ChannelStatusEnabled,
		Group:  "default",
		Models: "galo-test",
	}
	if err := model.DB.Create(&ch).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
	ability := model.Ability{
		Group:     "default",
		Model:     "galo-test",
		ChannelId: 1,
		Enabled:   true,
	}
	if err := model.DB.Create(&ability).Error; err != nil {
		t.Fatalf("create ability: %v", err)
	}

	// 匿名请求：不设置 "id" 上下文，模拟 TryUserAuth 未识别到登录态
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	GetPricing(c)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var anonBody struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &anonBody); err != nil {
		t.Fatalf("failed to unmarshal anonymous response: %v", err)
	}
	if len(anonBody.Data) == 0 {
		t.Fatal("expected at least one pricing item")
	}
	for i, item := range anonBody.Data {
		// 匿名响应必须完全不包含 enable_groups 字段（而非 "enable_groups": null）
		if _, ok := item["enable_groups"]; ok {
			t.Fatalf("item %d: anonymous pricing response must not contain enable_groups field, got %v", i, item["enable_groups"])
		}
	}

	// 登录请求：设置 "id" 上下文（用户不存在时走默认组，但 exists=true 保留 enable_groups）
	respAuthed := httptest.NewRecorder()
	cAuthed, _ := gin.CreateTestContext(respAuthed)
	cAuthed.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	cAuthed.Set("id", 0)
	GetPricing(cAuthed)

	if respAuthed.Code != http.StatusOK {
		t.Fatalf("expected 200 for authed request, got %d", respAuthed.Code)
	}
	var authedBody struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(respAuthed.Body.Bytes(), &authedBody); err != nil {
		t.Fatalf("failed to unmarshal authed response: %v", err)
	}
	if len(authedBody.Data) == 0 {
		t.Fatal("expected at least one pricing item for authed request")
	}
	found := false
	for _, item := range authedBody.Data {
		if groups, ok := item["enable_groups"].([]any); ok && len(groups) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("authed pricing response should keep enable_groups")
	}
}
