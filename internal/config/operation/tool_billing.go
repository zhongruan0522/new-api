package operation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/manager"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

// ToolBillingMode defines how a tool is billed.
// Currently only "per_call" is supported – fixed price per invocation.
const (
	ToolBillingModePerCall = "per_call"
)

// toolBillingLegacyRule 用于反序列化旧格式规则（带 quality/size/model_filter/provider 字段）。
// 当从 DB 加载的 JSON 中不包含 "conditions" 字段时，使用该结构读取并迁移为新格式。
type toolBillingLegacyRule struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ToolType    string  `json:"tool_type"`
	BillingMode string  `json:"billing_mode"`
	Price       float64 `json:"price"`
	ModelFilter string  `json:"model_filter,omitempty"`
	Quality     string  `json:"quality,omitempty"`
	Size        string  `json:"size,omitempty"`
	Provider    string  `json:"provider,omitempty"`
	Enabled     bool    `json:"enabled"`
}

// ToolBillingRule is a single pricing rule for one tool.
// 匹配维度通过 conditions 表达，不再使用硬编码的 quality/size 字段。
type ToolBillingRule struct {
	// Unique identifier, e.g. "web_search_openai", "image_gen_high_1024x1024"
	ID string `json:"id"`
	// Human-readable name shown in the UI
	Name string `json:"name"`
	// Which tool this rule applies to: "web_search", "image_generation"
	ToolType string `json:"tool_type"`
	// Billing mode: "per_call"
	BillingMode string `json:"billing_mode"`
	// Price in USD per call.
	Price float64 `json:"price"`
	// Conditions 是该规则的匹配条件列表。
	// 常见 field: "model", "provider", "quality", "size" 以及任意自定义属性。
	// model_filter 的旧前缀通配符语义通过 regex 模式表达。
	Conditions []Condition `json:"conditions,omitempty"`
	// Logic 控制 conditions 之间的 AND/OR 关系，默认 AND。
	Logic ConditionLogic `json:"logic,omitempty"`
	// Whether this rule is enabled
	Enabled bool `json:"enabled"`
}

// ToolBillingSetting holds all tool billing configuration.
// Stored in DB via manager.GlobalConfig under key "tool_billing_setting".
type ToolBillingSetting struct {
	Rules []ToolBillingRule `json:"rules"`
}

var toolBillingSetting = ToolBillingSetting{
	Rules: defaultToolBillingRules(),
}

func init() {
	manager.GlobalConfig.Register("tool_billing_setting", &toolBillingSetting)
}

func defaultToolBillingRules() []ToolBillingRule {
	return []ToolBillingRule{
		// --- Web Search (price = USD per call) ---
		{
			ID:          "web_search_openai_reasoning",
			Name:        "OpenAI Web Search (o系列/gpt-5)",
			ToolType:    "web_search",
			BillingMode: ToolBillingModePerCall,
			Price:       0.01,
			Conditions: []Condition{
				{Field: "model", Mode: ConditionModeRegex, Value: "^(o3|o4|gpt-5)"},
				{Field: "provider", Mode: ConditionModeEq, Value: "openai"},
			},
			Logic:   ConditionLogicAnd,
			Enabled: true,
		},
		{
			ID:          "web_search_openai_standard",
			Name:        "OpenAI Web Search (gpt-4o/gpt-4.1)",
			ToolType:    "web_search",
			BillingMode: ToolBillingModePerCall,
			Price:       0.025,
			Conditions: []Condition{
				{Field: "model", Mode: ConditionModeRegex, Value: "^(gpt-4o|gpt-4\\.1)"},
				{Field: "provider", Mode: ConditionModeEq, Value: "openai"},
			},
			Logic:   ConditionLogicAnd,
			Enabled: true,
		},
		{
			ID:          "web_search_claude",
			Name:        "Claude Web Search",
			ToolType:    "web_search",
			BillingMode: ToolBillingModePerCall,
			Price:       0.01,
			Conditions: []Condition{
				{Field: "provider", Mode: ConditionModeEq, Value: "claude"},
			},
			Logic:   ConditionLogicAnd,
			Enabled: true,
		},
		{
			ID:          "web_search_gemini",
			Name:        "Gemini Google Search",
			ToolType:    "web_search",
			BillingMode: ToolBillingModePerCall,
			Price:       0.01,
			Conditions: []Condition{
				{Field: "provider", Mode: ConditionModeEq, Value: "gemini"},
			},
			Logic:   ConditionLogicAnd,
			Enabled: true,
		},
		// --- Image Generation (price = USD per call) ---
		// quality/size 作为 conditions 维度表达，无需硬编码字段。
		imageGenRule("low", "1024x1024", 0.011),
		imageGenRule("low", "1024x1536", 0.016),
		imageGenRule("low", "1536x1024", 0.016),
		imageGenRule("medium", "1024x1024", 0.042),
		imageGenRule("medium", "1024x1536", 0.063),
		imageGenRule("medium", "1536x1024", 0.063),
		imageGenRule("high", "1024x1024", 0.167),
		imageGenRule("high", "1024x1536", 0.25),
		imageGenRule("high", "1536x1024", 0.25),
	}
}

// imageGenRule 构造一条 image_generation 规则的便捷函数。
func imageGenRule(quality, size string, price float64) ToolBillingRule {
	return ToolBillingRule{
		ID:          fmt.Sprintf("image_gen_%s_%s", quality, size),
		Name:        fmt.Sprintf("Image Gen %s %s", quality, size),
		ToolType:    "image_generation",
		BillingMode: ToolBillingModePerCall,
		Price:       price,
		Conditions: []Condition{
			{Field: "quality", Mode: ConditionModeEq, Value: quality},
			{Field: "size", Mode: ConditionModeEq, Value: size},
		},
		Logic:   ConditionLogicAnd,
		Enabled: true,
	}
}

// GetToolBillingSetting returns the current tool billing configuration.
func GetToolBillingSetting() *ToolBillingSetting {
	return &toolBillingSetting
}

// GetToolBillingPrice looks up the price for a tool call.
//
// toolType: "web_search", "image_generation" 或任意自定义工具类型
// attrs: 调用上下文属性，例如 {"model":"gpt-4o", "provider":"openai", "quality":"high", "size":"1024x1024"}
//
// 返回 (price, true) 表示匹配到一条规则，(0, false) 表示无匹配。
// price 的单位是 USD per call。
func GetToolBillingPrice(toolType string, attrs map[string]string) (float64, bool) {
	for i := range toolBillingSetting.Rules {
		rule := &toolBillingSetting.Rules[i]
		if !rule.Enabled {
			continue
		}
		if rule.ToolType != toolType {
			continue
		}
		ok, err := EvaluateConditions(rule.Conditions, attrs, rule.Logic)
		if err != nil {
			common.SysError(fmt.Sprintf("tool billing rule %s evaluation failed: %v", rule.ID, err))
			continue
		}
		if ok {
			return rule.Price, true
		}
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

// DefaultToolBillingRules2JSONString marshals the built-in default rule set
// (the one used when tool_billing_setting.rules has never been customized)
// to a JSON string. Used by the reset endpoint to restore defaults.
func DefaultToolBillingRules2JSONString() string {
	jsonBytes, err := jsonx.Marshal(ToolBillingSetting{Rules: defaultToolBillingRules()})
	if err != nil {
		common.SysError("error marshalling default tool billing rules: " + err.Error())
		return "[]"
	}
	return string(jsonBytes)
}

// ValidateToolBillingRules validates a JSON string of tool billing rules.
// 支持新格式（带 conditions）和旧格式（带 quality/size/model_filter/provider），
// 旧格式会自动迁移为 conditions。
func ValidateToolBillingRules(jsonStr string) error {
	// 先尝试新格式
	var rules []ToolBillingRule
	if err := jsonx.Unmarshal([]byte(jsonStr), &rules); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}

	// 检测是否是旧格式：尝试解析为 legacy，看是否有 model_filter/quality/size/provider 非空
	var legacyRules []toolBillingLegacyRule
	_ = jsonx.Unmarshal([]byte(jsonStr), &legacyRules)

	for i, rule := range rules {
		if strings.TrimSpace(rule.ID) == "" {
			return fmt.Errorf("rule %d: id is required", i)
		}
		if strings.TrimSpace(rule.ToolType) == "" {
			return fmt.Errorf("rule %d (%s): tool_type is required", i, rule.ID)
		}
		if strings.TrimSpace(rule.ToolType) != rule.ToolType {
			return fmt.Errorf("rule %d (%s): tool_type cannot contain leading or trailing whitespace", i, rule.ID)
		}
		if rule.BillingMode != ToolBillingModePerCall {
			return fmt.Errorf("rule %d (%s): unsupported billing_mode %q, only per_call is supported", i, rule.ID, rule.BillingMode)
		}
		if rule.Price < 0 {
			return fmt.Errorf("rule %d (%s): price cannot be negative", i, rule.ID)
		}
	}

	// 验证 logic 和 conditions 的合法性，避免运行时静默跳过计费规则。
	for i, rule := range rules {
		if !isValidConditionLogic(rule.Logic) {
			return fmt.Errorf("rule %d (%s): unsupported logic %q", i, rule.ID, rule.Logic)
		}
		for j, cond := range rule.Conditions {
			if strings.TrimSpace(cond.Field) == "" {
				return fmt.Errorf("rule %d (%s): condition %d has empty field", i, rule.ID, j)
			}
			if strings.TrimSpace(cond.Field) != cond.Field {
				return fmt.Errorf("rule %d (%s): condition %d field cannot contain leading or trailing whitespace", i, rule.ID, j)
			}
			mode := ConditionMode(strings.ToLower(string(cond.Mode)))
			if cond.Mode == "" {
				return fmt.Errorf("rule %d (%s): condition %d has empty mode", i, rule.ID, j)
			}
			if strings.TrimSpace(string(cond.Mode)) != string(cond.Mode) {
				return fmt.Errorf("rule %d (%s): condition %d mode cannot contain leading or trailing whitespace", i, rule.ID, j)
			}
			if !isValidConditionMode(mode) {
				return fmt.Errorf("rule %d (%s): condition %d has unsupported mode %q", i, rule.ID, j, cond.Mode)
			}
			if mode == ConditionModeRegex {
				v, ok := cond.Value.(string)
				if !ok {
					return fmt.Errorf("rule %d (%s): condition %d regex value must be a string", i, rule.ID, j)
				}
				if v == "" {
					return fmt.Errorf("rule %d (%s): condition %d has empty regex value", i, rule.ID, j)
				}
				if _, err := regexp.Compile(v); err != nil {
					return fmt.Errorf("rule %d (%s): condition %d has invalid regex: %w", i, rule.ID, j, err)
				}
			}
			if isNumericConditionMode(mode) {
				if err := validateConditionNumericValue(cond.Value); err != nil {
					return fmt.Errorf("rule %d (%s): condition %d has invalid numeric value: %w", i, rule.ID, j, err)
				}
			}
		}
	}

	return nil
}

func isValidConditionLogic(logic ConditionLogic) bool {
	if logic == "" {
		return true
	}
	trimmed := strings.TrimSpace(string(logic))
	if trimmed != string(logic) {
		return false
	}
	switch ConditionLogic(strings.ToUpper(trimmed)) {
	case ConditionLogicAnd, ConditionLogicOr:
		return true
	}
	return false
}

func isValidConditionMode(mode ConditionMode) bool {
	switch mode {
	case ConditionModeEq, ConditionModeNeq,
		ConditionModePrefix, ConditionModeSuffix,
		ConditionModeContains, ConditionModeRegex,
		ConditionModeGt, ConditionModeGte,
		ConditionModeLt, ConditionModeLte:
		return true
	}
	return false
}

func isNumericConditionMode(mode ConditionMode) bool {
	switch mode {
	case ConditionModeGt, ConditionModeGte, ConditionModeLt, ConditionModeLte:
		return true
	}
	return false
}

func validateConditionNumericValue(value interface{}) error {
	switch v := value.(type) {
	case float64, float32, int, int64:
		return nil
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("empty string")
		}
		if _, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported value type %T", value)
	}
}

// MigrateLegacyRules 检测并迁移旧格式规则（带 quality/size/model_filter/provider 字段）为新 conditions 格式。
// 如果 JSON 中所有规则都已经是新格式（无 legacy 字段），则原样返回。
// 返回迁移后的 JSON 字符串和是否有迁移发生的标志。
func MigrateLegacyRules(jsonStr string) (string, bool, error) {
	// 解析为新格式
	var rules []ToolBillingRule
	if err := jsonx.Unmarshal([]byte(jsonStr), &rules); err != nil {
		return "", false, fmt.Errorf("invalid JSON: %v", err)
	}

	// 同时解析为 legacy 格式以检测旧字段
	var legacyRules []toolBillingLegacyRule
	if err := jsonx.Unmarshal([]byte(jsonStr), &legacyRules); err != nil {
		// legacy 解析失败说明不是旧格式，直接返回新格式
		return jsonStr, false, nil
	}

	if len(rules) != len(legacyRules) {
		return jsonStr, false, nil
	}

	migrated := false
	for i := range rules {
		if i >= len(legacyRules) {
			break
		}
		legacy := legacyRules[i]

		// 检测是否有旧格式字段需要迁移
		hasLegacy := legacy.ModelFilter != "" || legacy.Quality != "" ||
			legacy.Size != "" || legacy.Provider != ""

		// 如果新格式已经有 conditions 且有 legacy 字段，不重复迁移
		if hasLegacy && len(rules[i].Conditions) == 0 {
			rules[i] = migrateLegacyRule(legacy, rules[i])
			migrated = true
		} else if hasLegacy && len(rules[i].Conditions) > 0 {
			// 有 conditions 说明已是新格式，legacy 字段是多余的，标记迁移以清理
			migrated = true
		}
	}

	if !migrated {
		return jsonStr, false, nil
	}

	data, err := jsonx.Marshal(rules)
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal migrated rules: %v", err)
	}
	return string(data), true, nil
}

// migrateLegacyRule 将一条旧格式规则转换为新 conditions 格式。
func migrateLegacyRule(legacy toolBillingLegacyRule, current ToolBillingRule) ToolBillingRule {
	var conds []Condition

	// model_filter → regex condition
	if legacy.ModelFilter != "" {
		regexPattern := modelFilterToRegex(legacy.ModelFilter)
		if regexPattern != "" {
			conds = append(conds, Condition{
				Field: "model",
				Mode:  ConditionModeRegex,
				Value: regexPattern,
			})
		}
	}

	// provider → eq condition
	if legacy.Provider != "" {
		conds = append(conds, Condition{
			Field: "provider",
			Mode:  ConditionModeEq,
			Value: legacy.Provider,
		})
	}

	// quality → eq condition
	if legacy.Quality != "" {
		conds = append(conds, Condition{
			Field: "quality",
			Mode:  ConditionModeEq,
			Value: legacy.Quality,
		})
	}

	// size → eq condition
	if legacy.Size != "" {
		conds = append(conds, Condition{
			Field: "size",
			Mode:  ConditionModeEq,
			Value: legacy.Size,
		})
	}

	current.Conditions = conds
	current.Logic = ConditionLogicAnd
	return current
}

// modelFilterToRegex 将旧的逗号分隔模型过滤列表转换为正则表达式。
// 带尾部 * 的条目保留前缀匹配语义；不带 * 的条目保留精确匹配语义。
func modelFilterToRegex(filter string) string {
	parts := strings.Split(filter, ",")
	var alternatives []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		hasWildcard := strings.HasSuffix(p, "*")
		if hasWildcard {
			p = p[:len(p)-1]
		}
		if p == "" {
			// 旧格式 "*" 表示不限制模型；迁移时不生成 model condition。
			return ""
		}
		escaped := regexp.QuoteMeta(p)
		if hasWildcard {
			alternatives = append(alternatives, escaped)
			continue
		}
		alternatives = append(alternatives, escaped+"$")
	}
	if len(alternatives) == 0 {
		return ""
	}
	return "^(?:" + strings.Join(alternatives, "|") + ")"
}
