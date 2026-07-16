package operation_setting

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
)

func TestGetToolBillingPrice_WebSearchOpenAI(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		wantPrice float64
	}{
		{"o3 series gets low price", "o3-mini", 0.01},
		{"o4 series gets low price", "o4-mini", 0.01},
		{"gpt-5 gets low price", "gpt-5", 0.01},
		{"gpt-4o gets high price", "gpt-4o", 0.025},
		{"gpt-4o-mini gets high price", "gpt-4o-mini", 0.025},
		{"gpt-4.1 gets high price", "gpt-4.1", 0.025},
		{"gpt-4.1-mini gets high price", "gpt-4.1-mini", 0.025},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, ok := GetToolBillingPrice("web_search", map[string]string{
				"model":    tt.modelName,
				"provider": "openai",
			})
			if !ok {
				t.Errorf("GetToolBillingPrice(web_search, %s, openai) expected to match a rule", tt.modelName)
			}
			if price != tt.wantPrice {
				t.Errorf("GetToolBillingPrice(web_search, %s, openai) = %v, want %v", tt.modelName, price, tt.wantPrice)
			}
		})
	}
}

func TestGetToolBillingPrice_WebSearchClaude(t *testing.T) {
	price, ok := GetToolBillingPrice("web_search", map[string]string{
		"model":    "claude-3.5-sonnet",
		"provider": "claude",
	})
	if !ok {
		t.Fatal("expected to match a rule")
	}
	if price != 0.01 {
		t.Errorf("GetToolBillingPrice(web_search, claude) = %v, want 0.01", price)
	}
}

func TestGetToolBillingPrice_WebSearchGemini(t *testing.T) {
	price, ok := GetToolBillingPrice("web_search", map[string]string{
		"model":    "gemini-2.5-flash",
		"provider": "gemini",
	})
	if !ok {
		t.Fatal("expected to match a rule")
	}
	if price != 0.01 {
		t.Errorf("GetToolBillingPrice(web_search, gemini) = %v, want 0.01", price)
	}
}

func TestGetToolBillingPrice_ImageGeneration(t *testing.T) {
	tests := []struct {
		name      string
		quality   string
		size      string
		wantPrice float64
	}{
		{"low 1024x1024", "low", "1024x1024", 0.011},
		{"low 1024x1536", "low", "1024x1536", 0.016},
		{"low 1536x1024", "low", "1536x1024", 0.016},
		{"medium 1024x1024", "medium", "1024x1024", 0.042},
		{"medium 1024x1536", "medium", "1024x1536", 0.063},
		{"medium 1536x1024", "medium", "1536x1024", 0.063},
		{"high 1024x1024", "high", "1024x1024", 0.167},
		{"high 1024x1536", "high", "1024x1536", 0.25},
		{"high 1536x1024", "high", "1536x1024", 0.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, ok := GetToolBillingPrice("image_generation", map[string]string{
				"model":    "gpt-4o",
				"provider": "openai",
				"quality":  tt.quality,
				"size":     tt.size,
			})
			if !ok {
				t.Fatalf("expected to match a rule for quality=%s size=%s", tt.quality, tt.size)
			}
			if price != tt.wantPrice {
				t.Errorf("GetToolBillingPrice(image_generation, %s, %s) = %v, want %v", tt.quality, tt.size, price, tt.wantPrice)
			}
		})
	}
}

func TestGetToolBillingPrice_ImageGeneration_UnknownReturnsFalse(t *testing.T) {
	_, ok := GetToolBillingPrice("image_generation", map[string]string{
		"model":    "gpt-4o",
		"provider": "openai",
		"quality":  "ultra",
		"size":     "2048x2048",
	})
	if ok {
		t.Error("GetToolBillingPrice(unknown quality/size) should return false")
	}
}

func TestGetToolBillingPrice_DisabledRule(t *testing.T) {
	// Temporarily disable a rule
	original := toolBillingSetting.Rules
	defer func() { toolBillingSetting.Rules = original }()

	rules := make([]ToolBillingRule, len(original))
	copy(rules, original)
	// Disable the claude web search rule
	for i := range rules {
		if rules[i].ToolType == "web_search" {
			for _, cond := range rules[i].Conditions {
				if cond.Field == "provider" && cond.Value == "claude" {
					rules[i].Enabled = false
				}
			}
		}
	}
	toolBillingSetting.Rules = rules

	_, ok := GetToolBillingPrice("web_search", map[string]string{
		"model":    "claude-3.5-sonnet",
		"provider": "claude",
	})
	if ok {
		t.Error("GetToolBillingPrice(disabled rule) should return false")
	}
}

func TestGetToolBillingPrice_ZeroPriceRule(t *testing.T) {
	original := toolBillingSetting.Rules
	defer func() { toolBillingSetting.Rules = original }()

	// Add a zero-price rule
	rules := append(original, ToolBillingRule{
		ID:          "test_zero_price",
		Name:        "Test Zero Price",
		ToolType:    "web_search",
		BillingMode: ToolBillingModePerCall,
		Price:       0,
		Conditions: []common.Condition{
			{Field: "provider", Mode: common.ConditionModeEq, Value: "test_provider"},
		},
		Logic:   common.ConditionLogicAnd,
		Enabled: true,
	})
	toolBillingSetting.Rules = rules

	price, ok := GetToolBillingPrice("web_search", map[string]string{
		"model":    "any-model",
		"provider": "test_provider",
	})
	if !ok {
		t.Fatal("expected to match the zero-price rule")
	}
	if price != 0 {
		t.Errorf("expected price 0, got %v", price)
	}
}

func TestValidateToolBillingRules(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		wantErr bool
	}{
		{
			"valid rules with conditions",
			`[{"id":"test","name":"Test","tool_type":"web_search","billing_mode":"per_call","price":10.0,"conditions":[{"field":"provider","mode":"eq","value":"openai"}]}]`,
			false,
		},
		{
			"valid rules without conditions",
			`[{"id":"test","name":"Test","tool_type":"web_search","billing_mode":"per_call","price":10.0}]`,
			false,
		},
		{
			"missing id",
			`[{"id":"","name":"Test","tool_type":"web_search","billing_mode":"per_call","price":10.0}]`,
			true,
		},
		{
			"missing tool_type",
			`[{"id":"test","name":"Test","tool_type":"","billing_mode":"per_call","price":10.0}]`,
			true,
		},
		{
			"invalid tool_type",
			`[{"id":"test","name":"Test","tool_type":"invalid","billing_mode":"per_call","price":10.0}]`,
			true,
		},
		{
			"invalid billing_mode",
			`[{"id":"test","name":"Test","tool_type":"web_search","billing_mode":"per_token","price":10.0}]`,
			true,
		},
		{
			"negative price",
			`[{"id":"test","name":"Test","tool_type":"web_search","billing_mode":"per_call","price":-1.0}]`,
			true,
		},
		{
			"zero price is valid",
			`[{"id":"test","name":"Test","tool_type":"web_search","billing_mode":"per_call","price":0}]`,
			false,
		},
		{
			"invalid condition mode",
			`[{"id":"test","name":"Test","tool_type":"web_search","billing_mode":"per_call","price":1.0,"conditions":[{"field":"x","mode":"bogus","value":"y"}]}]`,
			true,
		},
		{
			"empty condition field",
			`[{"id":"test","name":"Test","tool_type":"web_search","billing_mode":"per_call","price":1.0,"conditions":[{"field":"","mode":"eq","value":"y"}]}]`,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolBillingRules(tt.jsonStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToolBillingRules() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateToolBillingRules_InvalidJSON(t *testing.T) {
	err := ValidateToolBillingRules("not json")
	if err == nil {
		t.Error("ValidateToolBillingRules(invalid json) should return error")
	}
}

func TestDefaultRulesAreValid(t *testing.T) {
	for _, rule := range toolBillingSetting.Rules {
		if rule.ID == "" {
			t.Errorf("default rule has empty ID: %+v", rule)
		}
		if rule.ToolType == "" {
			t.Errorf("default rule %s has empty ToolType", rule.ID)
		}
		if rule.Price < 0 {
			t.Errorf("default rule %s has negative price", rule.ID)
		}
		if rule.BillingMode != ToolBillingModePerCall {
			t.Errorf("default rule %s has unsupported billing_mode %q", rule.ID, rule.BillingMode)
		}
	}
}

// 测试旧格式自动迁移
func TestMigrateLegacyRules(t *testing.T) {
	legacyJSON := `[
		{
			"id": "test_legacy",
			"name": "Test Legacy",
			"tool_type": "image_generation",
			"billing_mode": "per_call",
			"price": 0.05,
			"quality": "high",
			"size": "1024x1024",
			"provider": "openai",
			"enabled": true
		}
	]`

	migrated, didMigrate, err := MigrateLegacyRules(legacyJSON)
	if err != nil {
		t.Fatalf("MigrateLegacyRules failed: %v", err)
	}
	if !didMigrate {
		t.Fatal("expected migration to occur for legacy format")
	}

	// 验证迁移后规则仍然有效
	if err := ValidateToolBillingRules(migrated); err != nil {
		t.Fatalf("migrated rules failed validation: %v", err)
	}

	// 验证迁移后能正确匹配
	var rules []ToolBillingRule
	if err := common.Unmarshal([]byte(migrated), &rules); err != nil {
		t.Fatalf("failed to parse migrated rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if len(rules[0].Conditions) != 3 {
		t.Fatalf("expected 3 conditions (quality+size+provider), got %d", len(rules[0].Conditions))
	}

	// 迁移后的规则应该能正确匹配
	original := toolBillingSetting.Rules
	defer func() { toolBillingSetting.Rules = original }()
	toolBillingSetting.Rules = rules

	price, ok := GetToolBillingPrice("image_generation", map[string]string{
		"quality":  "high",
		"size":     "1024x1024",
		"provider": "openai",
	})
	if !ok {
		t.Fatal("migrated rule should match")
	}
	if price != 0.05 {
		t.Errorf("migrated rule price = %v, want 0.05", price)
	}
}

func TestMigrateLegacyRules_ModelFilter(t *testing.T) {
	legacyJSON := `[
		{
			"id": "test_model_filter",
			"name": "Test Model Filter",
			"tool_type": "web_search",
			"billing_mode": "per_call",
			"price": 0.02,
			"model_filter": "o3*,o4*,gpt-5*",
			"provider": "openai",
			"enabled": true
		}
	]`

	migrated, didMigrate, err := MigrateLegacyRules(legacyJSON)
	if err != nil {
		t.Fatalf("MigrateLegacyRules failed: %v", err)
	}
	if !didMigrate {
		t.Fatal("expected migration to occur")
	}

	var rules []ToolBillingRule
	if err := common.Unmarshal([]byte(migrated), &rules); err != nil {
		t.Fatalf("failed to parse migrated rules: %v", err)
	}

	// 应该有 2 个 condition: model(regex) + provider(eq)
	if len(rules[0].Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(rules[0].Conditions))
	}

	original := toolBillingSetting.Rules
	defer func() { toolBillingSetting.Rules = original }()
	toolBillingSetting.Rules = rules

	// o3-mini 应该匹配
	price, ok := GetToolBillingPrice("web_search", map[string]string{
		"model":    "o3-mini",
		"provider": "openai",
	})
	if !ok || price != 0.02 {
		t.Errorf("o3-mini should match with price 0.02, got ok=%v price=%v", ok, price)
	}

	// gpt-4o 不应该匹配（因为 model_filter 是 o3*,o4*,gpt-5*）
	_, ok = GetToolBillingPrice("web_search", map[string]string{
		"model":    "gpt-4o",
		"provider": "openai",
	})
	if ok {
		t.Error("gpt-4o should not match")
	}
}

func TestMigrateLegacyRules_AlreadyNewFormat(t *testing.T) {
	newFormatJSON := `[
		{
			"id": "already_new",
			"name": "Already New",
			"tool_type": "web_search",
			"billing_mode": "per_call",
			"price": 0.03,
			"conditions": [{"field":"provider","mode":"eq","value":"openai"}],
			"enabled": true
		}
	]`

	_, didMigrate, err := MigrateLegacyRules(newFormatJSON)
	if err != nil {
		t.Fatalf("MigrateLegacyRules failed: %v", err)
	}
	if didMigrate {
		t.Error("new format rules should not trigger migration")
	}
}

func TestModelFilterToRegex(t *testing.T) {
	tests := []struct {
		filter string
		want   string
	}{
		{"o3*", "^(o3)"},
		{"o3*,o4*", "^(o3|o4)"},
		{"o3*,o4*,gpt-5*", "^(o3|o4|gpt-5)"},
		{"gpt-4o,gpt-4o-mini", "^(gpt-4o|gpt-4o-mini)"},
		{"", ""},
		// 验证正则元字符被转义
		{"gpt-4.1*", "^(gpt-4\\.1)"},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			got := modelFilterToRegex(tt.filter)
			if got != tt.want {
				t.Errorf("modelFilterToRegex(%q) = %q, want %q", tt.filter, got, tt.want)
			}
		})
	}
}
