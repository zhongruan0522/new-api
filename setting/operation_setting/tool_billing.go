package operation_setting

import (
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/config"
)

// ToolBillingMode defines how a tool is billed.
// Currently only "per_call" is supported – fixed price per invocation.
const (
	ToolBillingModePerCall = "per_call"
)

// ToolBillingRule is a single pricing rule for one tool.
type ToolBillingRule struct {
	// Unique identifier, e.g. "web_search_openai", "image_generation_high_1024x1024"
	ID string `json:"id"`
	// Human-readable name shown in the UI
	Name string `json:"name"`
	// Which tool this rule applies to: "web_search", "image_generation"
	ToolType string `json:"tool_type"`
	// Billing mode: "per_call"
	BillingMode string `json:"billing_mode"`
	// Price in USD per call.
	Price float64 `json:"price"`
	// Optional model prefix filter (comma-separated). Empty means all models.
	// e.g. "gpt-4o*,gpt-4.1*" means this rule only applies to those models.
	ModelFilter string `json:"model_filter,omitempty"`
	// Optional quality filter (for image_generation): "low", "medium", "high"
	Quality string `json:"quality,omitempty"`
	// Optional size filter (for image_generation): "1024x1024", "1024x1536", "1536x1024"
	Size string `json:"size,omitempty"`
	// Optional provider filter: "openai", "claude", "gemini". Empty means all providers.
	Provider string `json:"provider,omitempty"`
	// Whether this rule is enabled
	Enabled bool `json:"enabled"`
}

// ToolBillingSetting holds all tool billing configuration.
// Stored in DB via config.GlobalConfig under key "tool_billing_setting".
type ToolBillingSetting struct {
	Rules []ToolBillingRule `json:"rules"`
}

var toolBillingSetting = ToolBillingSetting{
	Rules: defaultToolBillingRules(),
}

func init() {
	config.GlobalConfig.Register("tool_billing_setting", &toolBillingSetting)
}

func defaultToolBillingRules() []ToolBillingRule {
	return []ToolBillingRule{
		// --- Web Search (price = USD per call) ---
		{
			ID:          "web_search_openai_reasoning",
			Name:        "OpenAI Web Search (o系列/gpt-5)",
			ToolType:    "web_search",
			BillingMode: ToolBillingModePerCall,
			Price:       0.01, // $10/1K calls = $0.01/call
			ModelFilter: "o3*,o4*,gpt-5*",
			Provider:    "openai",
			Enabled:     true,
		},
		{
			ID:          "web_search_openai_standard",
			Name:        "OpenAI Web Search (gpt-4o/gpt-4.1)",
			ToolType:    "web_search",
			BillingMode: ToolBillingModePerCall,
			Price:       0.025, // $25/1K calls = $0.025/call
			ModelFilter: "gpt-4o*,gpt-4.1*",
			Provider:    "openai",
			Enabled:     true,
		},
		{
			ID:          "web_search_claude",
			Name:        "Claude Web Search",
			ToolType:    "web_search",
			BillingMode: ToolBillingModePerCall,
			Price:       0.01, // $10/1K calls = $0.01/call
			Provider:    "claude",
			Enabled:     true,
		},
		{
			ID:          "web_search_gemini",
			Name:        "Gemini Google Search",
			ToolType:    "web_search",
			BillingMode: ToolBillingModePerCall,
			Price:       0.01, // $10/1K calls = $0.01/call
			Provider:    "gemini",
			Enabled:     true,
		},
		// --- Image Generation (price = USD per call) ---
		{
			ID:          "image_gen_low_1024x1024",
			Name:        "Image Gen Low 1024x1024",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.011,
			Quality:     "low",
			Size:        "1024x1024",
			Enabled:     true,
		},
		{
			ID:          "image_gen_low_1024x1536",
			Name:        "Image Gen Low 1024x1536",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.016,
			Quality:     "low",
			Size:        "1024x1536",
			Enabled:     true,
		},
		{
			ID:          "image_gen_low_1536x1024",
			Name:        "Image Gen Low 1536x1024",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.016,
			Quality:     "low",
			Size:        "1536x1024",
			Enabled:     true,
		},
		{
			ID:          "image_gen_medium_1024x1024",
			Name:        "Image Gen Medium 1024x1024",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.042,
			Quality:     "medium",
			Size:        "1024x1024",
			Enabled:     true,
		},
		{
			ID:          "image_gen_medium_1024x1536",
			Name:        "Image Gen Medium 1024x1536",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.063,
			Quality:     "medium",
			Size:        "1024x1536",
			Enabled:     true,
		},
		{
			ID:          "image_gen_medium_1536x1024",
			Name:        "Image Gen Medium 1536x1024",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.063,
			Quality:     "medium",
			Size:        "1536x1024",
			Enabled:     true,
		},
		{
			ID:          "image_gen_high_1024x1024",
			Name:        "Image Gen High 1024x1024",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.167,
			Quality:     "high",
			Size:        "1024x1024",
			Enabled:     true,
		},
		{
			ID:          "image_gen_high_1024x1536",
			Name:        "Image Gen High 1024x1536",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.25,
			Quality:     "high",
			Size:        "1024x1536",
			Enabled:     true,
		},
		{
			ID:          "image_gen_high_1536x1024",
			Name:        "Image Gen High 1536x1024",
			ToolType:    "image_generation",
			BillingMode: ToolBillingModePerCall,
			Price:       0.25,
			Quality:     "high",
			Size:        "1536x1024",
			Enabled:     true,
		},
	}
}

// GetToolBillingSetting returns the current tool billing configuration.
func GetToolBillingSetting() *ToolBillingSetting {
	return &toolBillingSetting
}

// matchModelFilter checks if a model name matches a comma-separated prefix filter.
// Empty filter matches all models.
func matchModelFilter(modelName, filter string) bool {
	if filter == "" {
		return true
	}
	for _, prefix := range splitComma(filter) {
		if prefix == "" {
			continue
		}
		// Support wildcard suffix
		p := prefix
		hasWildcard := len(p) > 0 && p[len(p)-1] == '*'
		if hasWildcard {
			p = p[:len(p)-1]
		}
		if hasWildcard {
			if len(modelName) >= len(p) && modelName[:len(p)] == p {
				return true
			}
		} else {
			if modelName == p {
				return true
			}
		}
	}
	return false
}

// splitComma splits a comma-separated string, trimming whitespace.
func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range splitByRune(s, ',') {
		parts = append(parts, p)
	}
	return parts
}

func splitByRune(s string, r rune) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == r {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// GetToolBillingPrice looks up the price for a tool call.
// toolType: "web_search" or "image_generation"
// modelName: the model being used (for model-specific pricing)
// provider: "openai", "claude", "gemini" (for provider-specific pricing)
// quality: image quality ("low", "medium", "high") – only for image_generation
// size: image size ("1024x1024", etc.) – only for image_generation
//
// Returns (price, true) if a matching rule is found, (0, false) otherwise.
// price is in USD per call.
func GetToolBillingPrice(toolType, modelName, provider, quality, size string) (float64, bool) {
	for i := range toolBillingSetting.Rules {
		rule := &toolBillingSetting.Rules[i]
		if !rule.Enabled {
			continue
		}
		if rule.ToolType != toolType {
			continue
		}
		// Check provider filter
		if rule.Provider != "" && provider != "" && rule.Provider != provider {
			continue
		}
		// Check model filter
		if rule.ModelFilter != "" && !matchModelFilter(modelName, rule.ModelFilter) {
			continue
		}
		// Check quality filter (image_generation)
		if rule.Quality != "" && quality != "" && rule.Quality != quality {
			continue
		}
		// Check size filter (image_generation)
		if rule.Size != "" && size != "" && rule.Size != size {
			continue
		}
		return rule.Price, true
	}
	return 0, false
}

// GetToolBillingRules returns all configured rules.
func GetToolBillingRules() []ToolBillingRule {
	return toolBillingSetting.Rules
}

// UpdateToolBillingRules replaces all rules. Called from the admin API.
func UpdateToolBillingRules(rules []ToolBillingRule) {
	toolBillingSetting.Rules = rules
}

// ValidateToolBillingRules validates a JSON string of tool billing rules.
func ValidateToolBillingRules(jsonStr string) error {
	var rules []ToolBillingRule
	if err := common.Unmarshal([]byte(jsonStr), &rules); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	for i, rule := range rules {
		if rule.ID == "" {
			return fmt.Errorf("rule %d: id is required", i)
		}
		if rule.ToolType == "" {
			return fmt.Errorf("rule %d (%s): tool_type is required", i, rule.ID)
		}
		rule.ToolType = strings.ToLower(rule.ToolType)
		if rule.ToolType != "web_search" && rule.ToolType != "image_generation" {
			return fmt.Errorf("rule %d (%s): unsupported tool_type %q", i, rule.ID, rule.ToolType)
		}
		if rule.BillingMode != ToolBillingModePerCall {
			return fmt.Errorf("rule %d (%s): unsupported billing_mode %q, only per_call is supported", i, rule.ID, rule.BillingMode)
		}
		if rule.Price < 0 {
			return fmt.Errorf("rule %d (%s): price cannot be negative", i, rule.ID)
		}
	}
	return nil
}
