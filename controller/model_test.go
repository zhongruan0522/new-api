package controller

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/dto"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

func setupListModelsTestDB(t *testing.T) {
	t.Helper()

	oldDB := model.DB
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Ability{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	model.DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func TestListModelsIncludesContextPricingOnlyModel(t *testing.T) {
	setupListModelsTestDB(t)
	gin.SetMode(gin.TestMode)

	if err := ratio_setting.UpdateContextPricingByJSONString(`{
	  "context-priced-model": {
	    "enabled": true,
	    "tiers": [
	      {"min_tokens": 0, "model_ratio": 1, "completion_ratio": 2, "cache_ratio": 0.5, "create_cache_ratio": 1.25, "audio_ratio": 3, "audio_completion_ratio": 4}
	    ]
	  }
	}`); err != nil {
		t.Fatalf("UpdateContextPricingByJSONString returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = ratio_setting.UpdateContextPricingByJSONString("{}")
	})

	user := model.User{
		Id:          1,
		Username:    "list-models-user",
		Password:    "password123",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "List Models User",
		Group:       "paid-group",
		AffCode:     "list-models-aff",
	}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	priority := int64(0)
	ability := model.Ability{
		Group:     "paid-group",
		Model:     "context-priced-model",
		ChannelId: 1,
		Enabled:   true,
		Priority:  &priority,
	}
	if err := model.DB.Create(&ability).Error; err != nil {
		t.Fatalf("create ability: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", user.Id)

	ListModels(c, constant.ChannelTypeOpenAI)

	var response struct {
		Success bool               `json:"success"`
		Data    []dto.OpenAIModels `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !response.Success {
		t.Fatalf("ListModels success = false, response = %s", recorder.Body.String())
	}
	if len(response.Data) != 1 || response.Data[0].Id != "context-priced-model" {
		t.Fatalf("ListModels data = %+v, want context-priced-model", response.Data)
	}
}
