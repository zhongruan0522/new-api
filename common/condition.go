package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ConditionMode 定义条件匹配的模式。
type ConditionMode string

const (
	// ConditionModeEq 精确匹配（字符串或数值）。
	ConditionModeEq ConditionMode = "eq"
	// ConditionModeNeq 不等于。
	ConditionModeNeq ConditionMode = "neq"
	// ConditionModePrefix 前缀匹配。
	ConditionModePrefix ConditionMode = "prefix"
	// ConditionModeSuffix 后缀匹配。
	ConditionModeSuffix ConditionMode = "suffix"
	// ConditionModeContains 包含子串。
	ConditionModeContains ConditionMode = "contains"
	// ConditionModeRegex 正则表达式匹配。
	ConditionModeRegex ConditionMode = "regex"
	// ConditionModeGt 数值大于。
	ConditionModeGt ConditionMode = "gt"
	// ConditionModeGte 数值大于等于。
	ConditionModeGte ConditionMode = "gte"
	// ConditionModeLt 数值小于。
	ConditionModeLt ConditionMode = "lt"
	// ConditionModeLte 数值小于等于。
	ConditionModeLte ConditionMode = "lte"
)

// ConditionLogic 定义多条件之间的逻辑关系。
type ConditionLogic string

const (
	ConditionLogicAnd ConditionLogic = "AND"
	ConditionLogicOr  ConditionLogic = "OR"
)

// Condition 表示一条针对属性 map 的匹配条件。
// 与 relay/common/override.go 的 ConditionOperation 不同，这里面向的是
// key-value 属性映射（map[string]string），而非 JSON body 路径。
type Condition struct {
	// Field 是属性 map 中的键名，例如 "model"、"provider"、"quality"、"size"。
	Field string `json:"field"`
	// Mode 是匹配模式，取值见 ConditionMode 常量。
	Mode ConditionMode `json:"mode"`
	// Value 是匹配的目标值。eq/neq/gt/gte/lt/lte 模式下支持数值比较。
	Value interface{} `json:"value"`
	// Invert 为 true 时将匹配结果取反。
	Invert bool `json:"invert,omitempty"`
	// PassMissingKey 为 true 时，当属性 map 中不存在该 Field 时视为匹配通过；
	// 为 false（默认）时缺失 key 视为不匹配。
	PassMissingKey bool `json:"pass_missing_key,omitempty"`
}

// EvaluateConditions 针对一组属性评估条件列表。
// logic 决定多条件之间的 AND/OR 关系，空值默认 AND。
// 空条件列表表示无条件通过（返回 true）。
func EvaluateConditions(conditions []Condition, attrs map[string]string, logic ConditionLogic) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}
	effectiveLogic := ConditionLogicAnd
	if strings.ToUpper(string(logic)) == string(ConditionLogicOr) {
		effectiveLogic = ConditionLogicOr
	}

	results := make([]bool, len(conditions))
	for i, cond := range conditions {
		result, err := evaluateSingleCondition(cond, attrs)
		if err != nil {
			return false, fmt.Errorf("condition %d (field=%s, mode=%s): %w", i, cond.Field, cond.Mode, err)
		}
		results[i] = result
	}

	if effectiveLogic == ConditionLogicAnd {
		for _, r := range results {
			if !r {
				return false, nil
			}
		}
		return true, nil
	}
	// OR
	for _, r := range results {
		if r {
			return true, nil
		}
	}
	return false, nil
}

func evaluateSingleCondition(cond Condition, attrs map[string]string) (bool, error) {
	raw, exists := attrs[cond.Field]
	if !exists {
		if cond.PassMissingKey {
			return applyInvert(true, cond.Invert), nil
		}
		return applyInvert(false, cond.Invert), nil
	}

	mode := strings.ToLower(string(cond.Mode))
	result, err := compareValues(raw, cond.Value, mode)
	if err != nil {
		return false, err
	}
	return applyInvert(result, cond.Invert), nil
}

func applyInvert(result, invert bool) bool {
	if invert {
		return !result
	}
	return result
}

// compareValues 将实际值（字符串）与目标值按 mode 比较。
func compareValues(actual string, target interface{}, mode string) (bool, error) {
	switch mode {
	case string(ConditionModeEq):
		return equalsValue(actual, target), nil
	case string(ConditionModeNeq):
		return !equalsValue(actual, target), nil
	case string(ConditionModePrefix):
		return strings.HasPrefix(actual, toString(target)), nil
	case string(ConditionModeSuffix):
		return strings.HasSuffix(actual, toString(target)), nil
	case string(ConditionModeContains):
		return strings.Contains(actual, toString(target)), nil
	case string(ConditionModeRegex):
		re, err := regexp.Compile(toString(target))
		if err != nil {
			return false, fmt.Errorf("invalid regex: %w", err)
		}
		return re.MatchString(actual), nil
	case string(ConditionModeGt), string(ConditionModeGte), string(ConditionModeLt), string(ConditionModeLte):
		return compareNumeric(actual, target, mode)
	default:
		return false, fmt.Errorf("unsupported condition mode: %s", mode)
	}
}

// equalsValue 比较字符串实际值与目标值，目标可能是字符串、数值等。
func equalsValue(actual string, target interface{}) bool {
	if target == nil {
		return actual == ""
	}
	return actual == toString(target)
}

// toString 将任意目标值转为字符串用于字符串比较模式。
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// JSON 反序列化的数字默认是 float64
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case float32:
		return fmt.Sprintf("%g", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// compareNumeric 对数值比较模式做处理。
func compareNumeric(actual string, target interface{}, operator string) (bool, error) {
	actualNum, ok := parseFloat(actual)
	if !ok {
		return false, fmt.Errorf("actual value %q is not a number", actual)
	}
	targetNum, ok := parseFloatFromInterface(target)
	if !ok {
		return false, fmt.Errorf("target value %v is not a number", target)
	}

	switch operator {
	case string(ConditionModeGt):
		return actualNum > targetNum, nil
	case string(ConditionModeGte):
		return actualNum >= targetNum, nil
	case string(ConditionModeLt):
		return actualNum < targetNum, nil
	case string(ConditionModeLte):
		return actualNum <= targetNum, nil
	default:
		return false, fmt.Errorf("unsupported numeric operator: %s", operator)
	}
}

func parseFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func parseFloatFromInterface(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		return parseFloat(val)
	default:
		return 0, false
	}
}
