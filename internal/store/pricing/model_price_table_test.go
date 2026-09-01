package pricingstore

import (
	"fmt"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestReplaceModelPricePlansPersistsAtomicallyAndInvalidatesCache(t *testing.T) {
	setupModelPriceTableTestDB(t)

	initial := testStoredPricePlan("model-initial", []contract.ModelPriceComponent{
		{
			Component: contract.PriceComponentInput,
			Unit:      contract.PriceUnitPerMillionTokens,
			UnitPrice: "1.25",
		},
		{
			Component: contract.PriceComponentCacheRead,
			Unit:      contract.PriceUnitPerMillionTokens,
			UnitPrice: "0.25",
		},
	})
	if err := ReplaceModelPricePlans([]contract.ModelPricePlan{initial}); err != nil {
		t.Fatalf("replace initial plans: %v", err)
	}

	cachedInitial, err := GetModelPricePlans()
	if err != nil {
		t.Fatalf("get initial plans: %v", err)
	}
	if len(cachedInitial) != 1 || cachedInitial[0].ID == 0 || cachedInitial[0].ModelName != "model-initial" {
		t.Fatalf("unexpected initial plans: %+v", cachedInitial)
	}
	if cachedInitial[0].Source != contract.PricePlanSourceExplicit || cachedInitial[0].ReadOnly {
		t.Fatalf("persisted explicit metadata was not restored: %+v", cachedInitial[0])
	}
	if len(cachedInitial[0].Components) != 2 || cachedInitial[0].Components[0].Component != contract.PriceComponentCacheRead {
		t.Fatalf("components were not loaded/sorted: %+v", cachedInitial[0].Components)
	}

	replacement := testStoredPricePlan("model-replacement", []contract.ModelPriceComponent{{
		Component: contract.PriceComponentOutput,
		Unit:      contract.PriceUnitPerMillionTokens,
		UnitPrice: "2.5",
	}})
	if err := ReplaceModelPricePlans([]contract.ModelPricePlan{replacement}); err != nil {
		t.Fatalf("replace plans: %v", err)
	}
	afterReplacement, err := GetModelPricePlans()
	if err != nil {
		t.Fatalf("get replacement plans: %v", err)
	}
	if len(afterReplacement) != 1 || afterReplacement[0].ModelName != "model-replacement" {
		t.Fatalf("cache was not invalidated after replacement: %+v", afterReplacement)
	}

	invalid := testStoredPricePlan("invalid", []contract.ModelPriceComponent{
		{
			Component: contract.PriceComponentInput,
			Unit:      contract.PriceUnitPerMillionTokens,
			UnitPrice: "1",
		},
		{
			Component: contract.PriceComponentInput,
			Unit:      contract.PriceUnitPerMillionTokens,
			UnitPrice: "2",
		},
	})
	if err := ReplaceModelPricePlans([]contract.ModelPricePlan{invalid}); err == nil {
		t.Fatal("duplicate component write unexpectedly succeeded")
	}
	afterFailure, err := GetModelPricePlans()
	if err != nil {
		t.Fatalf("get plans after failed replacement: %v", err)
	}
	if len(afterFailure) != 1 || afterFailure[0].ModelName != "model-replacement" {
		t.Fatalf("failed replacement must roll back completely, got %+v", afterFailure)
	}
}

func setupModelPriceTableTestDB(t *testing.T) {
	t.Helper()

	oldDB := dbstore.DB
	dbHandle, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := dbHandle.AutoMigrate(&ModelPricePlan{}, &ModelPriceComponent{}); err != nil {
		t.Fatalf("migrate price table schema: %v", err)
	}
	dbstore.DB = dbHandle
	InvalidateModelPricePlanCache()

	t.Cleanup(func() {
		InvalidateModelPricePlanCache()
		if sqlDB, err := dbHandle.DB(); err == nil {
			_ = sqlDB.Close()
		}
		dbstore.DB = oldDB
	})
}

func testStoredPricePlan(modelName string, components []contract.ModelPriceComponent) contract.ModelPricePlan {
	return contract.ModelPricePlan{
		ModelName:             modelName,
		BillingMode:           contract.BillingModeToken,
		Currency:              "USD",
		ExchangeRate:          "1",
		PricePrecision:        12,
		RoundingMode:          contract.PriceRoundingHalfUp,
		GroupMultiplierSource: contract.GroupMultiplierSourceInherit,
		Components:            components,
	}
}
