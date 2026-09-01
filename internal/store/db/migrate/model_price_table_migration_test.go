package dbmigrate

import (
	"context"
	"fmt"
	"testing"

	"github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestModelPriceTableIsRegisteredForMainSchemaAndPreMigration(t *testing.T) {
	dbHandle, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := autoMigrateTargetMainSchema(dbHandle); err != nil {
		t.Fatalf("auto migrate target schema: %v", err)
	}
	if !dbHandle.Migrator().HasTable(&pricingstore.ModelPricePlan{}) {
		t.Fatal("target main schema is missing model_price_plans")
	}
	if !dbHandle.Migrator().HasTable(&pricingstore.ModelPriceComponent{}) {
		t.Fatal("target main schema is missing model_price_components")
	}

	steps := make(map[string]bool, len(dbPreMigrateMainSteps))
	for _, step := range dbPreMigrateMainSteps {
		steps[step.Name()] = true
	}
	for _, table := range []string{"model_price_plans", "model_price_components"} {
		if !steps[table] {
			t.Fatalf("pre-migrate main steps are missing %s", table)
		}
	}
}

func TestModelPriceTablePreMigrationSkipsMissingHistoricalSourceTables(t *testing.T) {
	sourceDB := openModelPriceTableMigrationDB(t, "source-missing")
	targetDB := openModelPriceTableMigrationDB(t, "target-missing")
	if err := autoMigrateTargetMainSchema(targetDB); err != nil {
		t.Fatalf("migrate target schema: %v", err)
	}

	if err := runModelPriceTableCopySteps(sourceDB, targetDB); err != nil {
		t.Fatalf("copy missing historical price tables: %v", err)
	}
	for _, model := range []any{&pricingstore.ModelPricePlan{}, &pricingstore.ModelPriceComponent{}} {
		var count int64
		if err := targetDB.Model(model).Count(&count).Error; err != nil {
			t.Fatalf("count target %T: %v", model, err)
		}
		if count != 0 {
			t.Fatalf("target %T count = %d, want 0", model, count)
		}
	}
}

func TestModelPriceTablePreMigrationCopiesPlansBeforeComponents(t *testing.T) {
	sourceDB := openModelPriceTableMigrationDB(t, "source-present")
	targetDB := openModelPriceTableMigrationDB(t, "target-present")
	if err := sourceDB.AutoMigrate(&pricingstore.ModelPricePlan{}, &pricingstore.ModelPriceComponent{}); err != nil {
		t.Fatalf("migrate source price tables: %v", err)
	}
	if err := autoMigrateTargetMainSchema(targetDB); err != nil {
		t.Fatalf("migrate target schema: %v", err)
	}

	plan := pricingstore.ModelPricePlan{
		ModelName:             "migrated-price-model",
		BillingMode:           "token",
		Currency:              "USD",
		ExchangeRate:          "1",
		PricePrecision:        12,
		RoundingMode:          "half_up",
		GroupMultiplierSource: "inherit_group_ratio",
		CreatedAt:             1,
		UpdatedAt:             1,
	}
	if err := sourceDB.Create(&plan).Error; err != nil {
		t.Fatalf("create source price plan: %v", err)
	}
	component := pricingstore.ModelPriceComponent{
		PlanID:    plan.ID,
		Component: "input",
		Unit:      "per_1m_tokens",
		UnitPrice: "1.25",
	}
	if err := sourceDB.Create(&component).Error; err != nil {
		t.Fatalf("create source price component: %v", err)
	}

	if err := runModelPriceTableCopySteps(sourceDB, targetDB); err != nil {
		t.Fatalf("copy component price tables: %v", err)
	}

	var copied pricingstore.ModelPricePlan
	if err := targetDB.Preload("Components").First(&copied, "model_name = ?", plan.ModelName).Error; err != nil {
		t.Fatalf("load copied price plan: %v", err)
	}
	if len(copied.Components) != 1 || copied.Components[0].PlanID != copied.ID || copied.Components[0].UnitPrice != component.UnitPrice {
		t.Fatalf("copied price plan components = %+v", copied.Components)
	}
}

func openModelPriceTableMigrationDB(t *testing.T, suffix string) *gorm.DB {
	t.Helper()
	dbHandle, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s-%s?mode=memory&cache=shared", t.Name(), suffix)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open %s sqlite: %v", suffix, err)
	}
	return dbHandle
}

func runModelPriceTableCopySteps(sourceDB, targetDB *gorm.DB) error {
	job := newDBPreMigrateJob("price-table-test", "sqlite", "sqlite", DBPreMigrateStartParams{})
	for _, step := range dbPreMigrateMainSteps {
		if step.Name() != "model_price_plans" && step.Name() != "model_price_components" {
			continue
		}
		if err := step.Run(context.Background(), job, sourceDB, targetDB, DBPreMigrateStartParams{}); err != nil {
			return err
		}
	}
	return nil
}
