package model

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

func setupModelPricingTestDatabase(t *testing.T) {
	t.Helper()

	originalDB := DB
	originalOptionMap := common.OptionMap
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initCol()

	databasePath := filepath.Join(t.TempDir(), "model-pricing.db")
	database, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test database: %v", err)
	}
	if err := database.AutoMigrate(&Option{}, &ModelPricing{}); err != nil {
		t.Fatalf("migrate sqlite test database: %v", err)
	}
	DB = database
	common.OptionMap = make(map[string]string)

	t.Cleanup(func() {
		DB = originalDB
		common.OptionMap = originalOptionMap
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		initCol()
	})
}

func preserveModelPricingRuntimeSettings(t *testing.T) {
	t.Helper()

	originalModelPrice := ratio_setting.ModelPrice2JSONString()
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalCompletionRatio := ratio_setting.CompletionRatio2JSONString()
	originalCacheRatio := ratio_setting.CacheRatio2JSONString()
	originalCreateCacheRatio := ratio_setting.CreateCacheRatio2JSONString()
	originalAudioRatio := ratio_setting.AudioRatio2JSONString()
	originalAudioCompletionRatio := ratio_setting.AudioCompletionRatio2JSONString()
	originalContextPricing := ratio_setting.ContextPricing2JSONString()

	t.Cleanup(func() {
		_ = ratio_setting.UpdateModelPriceByJSONString(originalModelPrice)
		_ = ratio_setting.UpdateModelRatioByJSONString(originalModelRatio)
		_ = ratio_setting.UpdateCompletionRatioByJSONString(originalCompletionRatio)
		_ = ratio_setting.UpdateCacheRatioByJSONString(originalCacheRatio)
		_ = ratio_setting.UpdateCreateCacheRatioByJSONString(originalCreateCacheRatio)
		_ = ratio_setting.UpdateAudioRatioByJSONString(originalAudioRatio)
		_ = ratio_setting.UpdateAudioCompletionRatioByJSONString(originalAudioCompletionRatio)
		_ = ratio_setting.UpdateContextPricingByJSONString(originalContextPricing)
	})
}

func TestSyncModelPricingTableAndCacheMigratesCurrentSettings(t *testing.T) {
	setupModelPricingTestDatabase(t)
	preserveModelPricingRuntimeSettings(t)

	contextPricingJSON := `{"tiered-model":{"enabled":true,"tiers":[{"name":"default","min_tokens":0,"model_ratio":3,"completion_ratio":4,"cache_ratio":0.5,"create_cache_ratio":1.25,"audio_ratio":2,"audio_completion_ratio":3}]}}`
	pricingInputs := map[string]string{
		"model price":            `{"free-fixed-model":0,"paid-fixed-model":0.2}`,
		"model ratio":            `{"free-ratio-model":0,"wildcard-model-*":2.5}`,
		"completion ratio":       `{"free-ratio-model":0,"wildcard-model-*":3}`,
		"cache ratio":            `{"free-ratio-model":0.1}`,
		"create cache ratio":     `{"free-ratio-model":1.25}`,
		"audio ratio":            `{"audio-model":8}`,
		"audio completion ratio": `{"audio-model":2}`,
		"context pricing":        contextPricingJSON,
	}
	if err := ratio_setting.UpdateModelPriceByJSONString(pricingInputs["model price"]); err != nil {
		t.Fatalf("set model price: %v", err)
	}
	if err := ratio_setting.UpdateModelRatioByJSONString(pricingInputs["model ratio"]); err != nil {
		t.Fatalf("set model ratio: %v", err)
	}
	if err := ratio_setting.UpdateCompletionRatioByJSONString(pricingInputs["completion ratio"]); err != nil {
		t.Fatalf("set completion ratio: %v", err)
	}
	if err := ratio_setting.UpdateCacheRatioByJSONString(pricingInputs["cache ratio"]); err != nil {
		t.Fatalf("set cache ratio: %v", err)
	}
	if err := ratio_setting.UpdateCreateCacheRatioByJSONString(pricingInputs["create cache ratio"]); err != nil {
		t.Fatalf("set create cache ratio: %v", err)
	}
	if err := ratio_setting.UpdateAudioRatioByJSONString(pricingInputs["audio ratio"]); err != nil {
		t.Fatalf("set audio ratio: %v", err)
	}
	if err := ratio_setting.UpdateAudioCompletionRatioByJSONString(pricingInputs["audio completion ratio"]); err != nil {
		t.Fatalf("set audio completion ratio: %v", err)
	}
	if err := ratio_setting.UpdateContextPricingByJSONString(pricingInputs["context pricing"]); err != nil {
		t.Fatalf("set context pricing: %v", err)
	}

	if err := SyncModelPricingTableAndCache(); err != nil {
		t.Fatalf("sync model pricing table: %v", err)
	}

	var freeFixedModel ModelPricing
	if err := DB.Where("model_name = ?", "free-fixed-model").First(&freeFixedModel).Error; err != nil {
		t.Fatalf("load free fixed model pricing row: %v", err)
	}
	if freeFixedModel.ModelPrice == nil || *freeFixedModel.ModelPrice != 0 {
		t.Fatalf("expected explicit zero model price, got %#v", freeFixedModel.ModelPrice)
	}

	var freeRatioModel ModelPricing
	if err := DB.Where("model_name = ?", "free-ratio-model").First(&freeRatioModel).Error; err != nil {
		t.Fatalf("load free ratio model pricing row: %v", err)
	}
	if freeRatioModel.ModelRatio == nil || *freeRatioModel.ModelRatio != 0 {
		t.Fatalf("expected explicit zero model ratio, got %#v", freeRatioModel.ModelRatio)
	}

	price, priceExists := ratio_setting.GetModelPrice("free-fixed-model", false)
	if !priceExists || price != 0 {
		t.Fatalf("expected cached explicit zero model price, got price=%v exists=%v", price, priceExists)
	}
	ratio, ratioExists, _ := ratio_setting.GetModelRatio("free-ratio-model")
	if !ratioExists || ratio != 0 {
		t.Fatalf("expected cached explicit zero model ratio, got ratio=%v exists=%v", ratio, ratioExists)
	}
	if wildcardRatio, wildcardExists, _ := ratio_setting.GetModelRatio("wildcard-model-*"); !wildcardExists || wildcardRatio != 2.5 {
		t.Fatalf("expected wildcard model ratio to survive migration, got ratio=%v exists=%v", wildcardRatio, wildcardExists)
	}
	if _, contextPricingEnabled, err := ratio_setting.MatchContextPricingTier("tiered-model", 100); !contextPricingEnabled || err != nil {
		t.Fatalf("expected context pricing cache to survive migration, enabled=%v err=%v", contextPricingEnabled, err)
	}
}

func TestUpdateOptionWritesModelPricingTableAndSynthesizesOptionMap(t *testing.T) {
	setupModelPricingTestDatabase(t)
	preserveModelPricingRuntimeSettings(t)

	if err := UpdateOption("ModelPrice", `{"new-free-model":0,"new-paid-model":0.75}`); err != nil {
		t.Fatalf("update model price option: %v", err)
	}

	var freeModelPricing ModelPricing
	if err := DB.Where("model_name = ?", "new-free-model").First(&freeModelPricing).Error; err != nil {
		t.Fatalf("load new free model pricing row: %v", err)
	}
	if freeModelPricing.ModelPrice == nil || *freeModelPricing.ModelPrice != 0 {
		t.Fatalf("expected explicit zero model price in table, got %#v", freeModelPricing.ModelPrice)
	}

	optionValue := common.OptionMap["ModelPrice"]
	parsedOptionValue := map[string]float64{}
	if err := common.UnmarshalJsonStr(optionValue, &parsedOptionValue); err != nil {
		t.Fatalf("parse synthesized ModelPrice option %q: %v", optionValue, err)
	}
	if parsedOptionValue["new-free-model"] != 0 || parsedOptionValue["new-paid-model"] != 0.75 {
		t.Fatalf("unexpected synthesized ModelPrice option: %#v", parsedOptionValue)
	}
}

func TestModelPricingTableWinsOverStaleLegacyOption(t *testing.T) {
	setupModelPricingTestDatabase(t)
	preserveModelPricingRuntimeSettings(t)

	now := time.Now().Unix()
	if err := DB.Create(&ModelPricing{
		ModelName:  "table-model",
		ModelPrice: float64Pointer(0.42),
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed model pricing table: %v", err)
	}
	if err := DB.Create(&Option{Key: "ModelPrice", Value: `{"legacy-model":0.99}`}).Error; err != nil {
		t.Fatalf("seed legacy model price option: %v", err)
	}

	loadOptionsFromDatabase()

	if legacyPrice, exists := ratio_setting.GetModelPrice("legacy-model", false); exists {
		t.Fatalf("expected stale legacy option to be overridden by table, got price=%v", legacyPrice)
	}
	tablePrice, exists := ratio_setting.GetModelPrice("table-model", false)
	if !exists || tablePrice != 0.42 {
		t.Fatalf("expected table model price to win, got price=%v exists=%v", tablePrice, exists)
	}
	if !strings.Contains(fmt.Sprintf("%s", common.OptionMap["ModelPrice"]), "table-model") {
		t.Fatalf("expected synthesized option map to include table-backed model price, got %s", common.OptionMap["ModelPrice"])
	}
}
