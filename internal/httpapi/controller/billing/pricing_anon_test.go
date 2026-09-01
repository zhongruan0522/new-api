package billingcontroller

import (
	"encoding/json"
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/infra/redis"
	"github.com/NookMux/NookMux/internal/store/channel"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/NookMux/NookMux/internal/store/vendor_meta"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupPricingTestDB(t *testing.T) {
	t.Helper()

	oldDB := dbstore.DB
	oldLogDB := dbstore.LOG_DB
	oldRedisEnabled := redis.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(
		&channelstore.Ability{},
		&channelstore.Channel{},
		&vendormetastore.Model{},
		&pricingstore.ModelPricePlan{},
		&pricingstore.ModelPriceComponent{},
	); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	dbstore.DB = db
	dbstore.LOG_DB = db
	redis.RedisEnabled = false
	common.MemoryCacheEnabled = false
	pricingstore.InvalidateModelPricePlanCache()

	t.Cleanup(func() {
		pricingstore.InvalidateModelPricePlanCache()
		dbstore.DB = oldDB
		dbstore.LOG_DB = oldLogDB
		redis.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func TestGetPricingAnonymousStripsGroupScopedComponentPlans(t *testing.T) {
	setupPricingTestDB(t)
	gin.SetMode(gin.TestMode)

	channel := channelstore.Channel{
		Id:     2,
		Status: common.ChannelStatusEnabled,
		Group:  "default",
		Models: "component-price-model",
	}
	if err := dbstore.DB.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
	ability := channelstore.Ability{
		Group:     "default",
		Model:     "component-price-model",
		ChannelId: channel.Id,
		Enabled:   true,
	}
	if err := dbstore.DB.Create(&ability).Error; err != nil {
		t.Fatalf("create ability: %v", err)
	}
	if err := pricingstore.ReplaceModelPricePlans([]contract.ModelPricePlan{
		testMarketplacePricePlan("component-price-model", "", "1"),
		testMarketplacePricePlan("component-price-model", "internal", "2"),
	}); err != nil {
		t.Fatalf("persist price plans: %v", err)
	}
	if err := pricingstore.RefreshPricing(); err != nil {
		t.Fatalf("refresh pricing: %v", err)
	}

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	GetPricing(context)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var body struct {
		Data []struct {
			ModelName  string                    `json:"model_name"`
			PricePlans []contract.ModelPricePlan `json:"price_plans"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode pricing response: %v", err)
	}
	for _, item := range body.Data {
		if item.ModelName != "component-price-model" {
			continue
		}
		if len(item.PricePlans) != 1 || item.PricePlans[0].EffectiveGroup != "" {
			t.Fatalf("anonymous response exposed group-scoped price plans: %+v", item.PricePlans)
		}
		for _, cached := range pricingstore.GetPricing() {
			if cached.ModelName == "component-price-model" && len(cached.PricePlans) == 2 {
				return
			}
		}
		t.Fatal("anonymous filtering mutated the shared marketplace price-plan cache")
	}
	t.Fatal("component-price-model was missing from anonymous pricing response")
}

func testMarketplacePricePlan(modelName, effectiveGroup, unitPrice string) contract.ModelPricePlan {
	return contract.ModelPricePlan{
		ModelName:             modelName,
		EffectiveGroup:        effectiveGroup,
		BillingMode:           contract.BillingModeToken,
		Currency:              "USD",
		ExchangeRate:          "1",
		PricePrecision:        12,
		RoundingMode:          contract.PriceRoundingHalfUp,
		GroupMultiplierSource: contract.GroupMultiplierSourceInherit,
		Components: []contract.ModelPriceComponent{{
			Component: contract.PriceComponentInput,
			Unit:      contract.PriceUnitPerMillionTokens,
			UnitPrice: unitPrice,
		}},
	}
}

// 匿名访问 pricing 时 enable_groups 不应携带内部组名（exists=false 路径），
// 登录访问时保留。防止公开定价页向匿名调用者暴露内部分组分类法。
func TestGetPricingAnonymousStripsEnableGroups(t *testing.T) {
	setupPricingTestDB(t)
	gin.SetMode(gin.TestMode)

	ch := channelstore.Channel{
		Id:     1,
		Status: common.ChannelStatusEnabled,
		Group:  "default",
		Models: "galo-test",
	}
	if err := dbstore.DB.Create(&ch).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
	ability := channelstore.Ability{
		Group:     "default",
		Model:     "galo-test",
		ChannelId: 1,
		Enabled:   true,
	}
	if err := dbstore.DB.Create(&ability).Error; err != nil {
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
