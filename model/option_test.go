package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/setting/operation_setting"
	"gorm.io/gorm"
)

func TestLoadOptionsMigratesToolBillingRulesAndRefreshesRuntimeConfig(t *testing.T) {
	oldDB := DB
	oldRules := append([]operation_setting.ToolBillingRule(nil), operation_setting.GetToolBillingRules()...)
	common.OptionMapRWMutex.Lock()
	oldOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&Option{}); err != nil {
		t.Fatalf("migrate option table: %v", err)
	}
	DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
		operation_setting.UpdateToolBillingRules(oldRules)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	legacyRules := `[
		{
			"id": "legacy_web_search_exact",
			"name": "Legacy Web Search Exact",
			"tool_type": "web_search",
			"billing_mode": "per_call",
			"price": 0.02,
			"model_filter": "gpt-4o",
			"provider": "openai",
			"enabled": true
		}
	]`
	if err := DB.Create(&Option{Key: "tool_billing_setting.rules", Value: legacyRules}).Error; err != nil {
		t.Fatalf("insert legacy option: %v", err)
	}

	loadOptionsFromDatabase()

	price, ok := operation_setting.GetToolBillingPrice("web_search", map[string]string{
		"model":    "gpt-4o",
		"provider": "openai",
	})
	if !ok || price != 0.02 {
		t.Fatalf("migrated runtime rule should match gpt-4o with price 0.02, got ok=%v price=%v", ok, price)
	}

	_, ok = operation_setting.GetToolBillingPrice("web_search", map[string]string{
		"model":    "gpt-4o-mini",
		"provider": "openai",
	})
	if ok {
		t.Fatal("migrated exact legacy model_filter should not match gpt-4o-mini")
	}

	var stored Option
	if err := DB.First(&stored, "key = ?", "tool_billing_setting.rules").Error; err != nil {
		t.Fatalf("query migrated option: %v", err)
	}
	if strings.Contains(stored.Value, "model_filter") || !strings.Contains(stored.Value, "conditions") {
		t.Fatalf("stored option was not migrated to conditions format: %s", stored.Value)
	}
}

func TestMigrateLegacyToolBillingRulesInOptionsBeforeRuntimeLoad(t *testing.T) {
	oldRules := append([]operation_setting.ToolBillingRule(nil), operation_setting.GetToolBillingRules()...)
	common.OptionMapRWMutex.Lock()
	oldOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		operation_setting.UpdateToolBillingRules(oldRules)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	options := []*Option{{
		Key: "tool_billing_setting.rules",
		Value: `[
			{
				"id": "legacy_web_search_exact",
				"name": "Legacy Web Search Exact",
				"tool_type": "web_search",
				"billing_mode": "per_call",
				"price": 0.02,
				"model_filter": "gpt-4o",
				"provider": "openai",
				"enabled": true
			}
		]`,
	}}

	didMigrate, _ := migrateLegacyToolBillingRulesInOptions(options)
	if !didMigrate {
		t.Fatal("expected legacy tool billing rules to migrate before runtime load")
	}
	if strings.Contains(options[0].Value, "model_filter") || !strings.Contains(options[0].Value, "conditions") {
		t.Fatalf("option value was not migrated before runtime load: %s", options[0].Value)
	}
	if err := updateOptionMap(options[0].Key, options[0].Value); err != nil {
		t.Fatalf("load migrated tool billing rules: %v", err)
	}

	_, ok := operation_setting.GetToolBillingPrice("web_search", map[string]string{
		"model":    "gpt-4o-mini",
		"provider": "openai",
	})
	if ok {
		t.Fatal("legacy exact model_filter should not become an unconditional runtime rule")
	}
}
