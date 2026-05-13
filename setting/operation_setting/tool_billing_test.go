package operation_setting

import (
	"testing"
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
			price, ok := GetToolBillingPrice("web_search", tt.modelName, "openai", "", "")
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
	price, ok := GetToolBillingPrice("web_search", "claude-3.5-sonnet", "claude", "", "")
	if !ok {
		t.Fatal("expected to match a rule")
	}
	if price != 0.01 {
		t.Errorf("GetToolBillingPrice(web_search, claude) = %v, want 0.01", price)
	}
}

func TestGetToolBillingPrice_WebSearchGemini(t *testing.T) {
	price, ok := GetToolBillingPrice("web_search", "gemini-2.5-flash", "gemini", "", "")
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
			price, ok := GetToolBillingPrice("image_generation", "gpt-4o", "openai", tt.quality, tt.size)
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
	_, ok := GetToolBillingPrice("image_generation", "gpt-4o", "openai", "ultra", "2048x2048")
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
		if rules[i].Provider == "claude" && rules[i].ToolType == "web_search" {
			rules[i].Enabled = false
		}
	}
	toolBillingSetting.Rules = rules

	_, ok := GetToolBillingPrice("web_search", "claude-3.5-sonnet", "claude", "", "")
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
		Provider:    "test_provider",
		Enabled:     true,
	})
	toolBillingSetting.Rules = rules

	price, ok := GetToolBillingPrice("web_search", "any-model", "test_provider", "", "")
	if !ok {
		t.Fatal("expected to match the zero-price rule")
	}
	if price != 0 {
		t.Errorf("expected price 0, got %v", price)
	}
}

func TestMatchModelFilter(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		filter    string
		wantMatch bool
	}{
		{"empty filter matches all", "gpt-4o", "", true},
		{"single prefix match", "gpt-4o", "gpt-4o*", true},
		{"single prefix no match", "gpt-4o", "claude*", false},
		{"multiple prefix match first", "gpt-4o", "gpt-4o*,claude*", true},
		{"multiple prefix match second", "claude-3", "gpt-4o*,claude*", true},
		{"exact match without wildcard", "gpt-4o", "gpt-4o", true},
		{"exact no match", "gpt-4o-mini", "gpt-4o", false},
		{"wildcard match", "o3-mini", "o3*", true},
		{"wildcard no match", "gpt-4o", "o3*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchModelFilter(tt.model, tt.filter)
			if got != tt.wantMatch {
				t.Errorf("matchModelFilter(%q, %q) = %v, want %v", tt.model, tt.filter, got, tt.wantMatch)
			}
		})
	}
}

func TestValidateToolBillingRules(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		wantErr bool
	}{
		{
			"valid rules",
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
