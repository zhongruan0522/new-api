package common

import (
	"testing"
)

func TestEvaluateConditions_EmptyConditions(t *testing.T) {
	ok, err := EvaluateConditions(nil, map[string]string{"model": "gpt-4o"}, ConditionLogicAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("empty conditions should match")
	}
}

func TestEvaluateConditions_Eq(t *testing.T) {
	conds := []Condition{
		{Field: "provider", Mode: ConditionModeEq, Value: "openai"},
	}
	ok, _ := EvaluateConditions(conds, map[string]string{"provider": "openai"}, ConditionLogicAnd)
	if !ok {
		t.Error("eq match should pass")
	}
	ok, _ = EvaluateConditions(conds, map[string]string{"provider": "claude"}, ConditionLogicAnd)
	if ok {
		t.Error("non-equal should fail")
	}
}

func TestEvaluateConditions_Neq(t *testing.T) {
	conds := []Condition{
		{Field: "provider", Mode: ConditionModeNeq, Value: "openai"},
	}
	ok, _ := EvaluateConditions(conds, map[string]string{"provider": "claude"}, ConditionLogicAnd)
	if !ok {
		t.Error("neq should pass when different")
	}
	ok, _ = EvaluateConditions(conds, map[string]string{"provider": "openai"}, ConditionLogicAnd)
	if ok {
		t.Error("neq should fail when equal")
	}
}

func TestEvaluateConditions_Prefix(t *testing.T) {
	conds := []Condition{
		{Field: "model", Mode: ConditionModePrefix, Value: "gpt-4o"},
	}
	cases := map[string]bool{
		"gpt-4o":      true,
		"gpt-4o-mini": true,
		"gpt-5":       false,
		"":            false,
	}
	for model, want := range cases {
		ok, _ := EvaluateConditions(conds, map[string]string{"model": model}, ConditionLogicAnd)
		if ok != want {
			t.Errorf("prefix(model=%s) = %v, want %v", model, ok, want)
		}
	}
}

func TestEvaluateConditions_Suffix(t *testing.T) {
	conds := []Condition{
		{Field: "model", Mode: ConditionModeSuffix, Value: "-mini"},
	}
	ok, _ := EvaluateConditions(conds, map[string]string{"model": "gpt-4o-mini"}, ConditionLogicAnd)
	if !ok {
		t.Error("suffix should match")
	}
}

func TestEvaluateConditions_Contains(t *testing.T) {
	conds := []Condition{
		{Field: "model", Mode: ConditionModeContains, Value: "4o"},
	}
	ok, _ := EvaluateConditions(conds, map[string]string{"model": "gpt-4o-mini"}, ConditionLogicAnd)
	if !ok {
		t.Error("contains should match")
	}
}

func TestEvaluateConditions_Regex(t *testing.T) {
	// 模拟旧 model_filter "o3*,o4*,gpt-5*" 的 OR 语义
	conds := []Condition{
		{Field: "model", Mode: ConditionModeRegex, Value: "^(o3|o4|gpt-5)"},
	}
	cases := map[string]bool{
		"o3-mini": true,
		"o4-mini": true,
		"gpt-5":   true,
		"gpt-4o":  false,
		"":        false,
	}
	for model, want := range cases {
		ok, _ := EvaluateConditions(conds, map[string]string{"model": model}, ConditionLogicAnd)
		if ok != want {
			t.Errorf("regex(model=%s) = %v, want %v", model, ok, want)
		}
	}
}

func TestEvaluateConditions_Regex_Invalid(t *testing.T) {
	conds := []Condition{
		{Field: "model", Mode: ConditionModeRegex, Value: "[invalid"},
	}
	_, err := EvaluateConditions(conds, map[string]string{"model": "test"}, ConditionLogicAnd)
	if err == nil {
		t.Error("invalid regex should return error")
	}
}

func TestEvaluateConditions_Gt(t *testing.T) {
	conds := []Condition{
		{Field: "count", Mode: ConditionModeGt, Value: float64(5)},
	}
	ok, _ := EvaluateConditions(conds, map[string]string{"count": "10"}, ConditionLogicAnd)
	if !ok {
		t.Error("10 > 5 should pass")
	}
	ok, _ = EvaluateConditions(conds, map[string]string{"count": "3"}, ConditionLogicAnd)
	if ok {
		t.Error("3 > 5 should fail")
	}
}

func TestEvaluateConditions_Lte(t *testing.T) {
	conds := []Condition{
		{Field: "count", Mode: ConditionModeLte, Value: float64(5)},
	}
	ok, _ := EvaluateConditions(conds, map[string]string{"count": "5"}, ConditionLogicAnd)
	if !ok {
		t.Error("5 <= 5 should pass")
	}
}

func TestEvaluateConditions_Gt_NonNumeric(t *testing.T) {
	conds := []Condition{
		{Field: "model", Mode: ConditionModeGt, Value: float64(5)},
	}
	_, err := EvaluateConditions(conds, map[string]string{"model": "abc"}, ConditionLogicAnd)
	if err == nil {
		t.Error("non-numeric actual should error")
	}
}

func TestEvaluateConditions_Invert(t *testing.T) {
	conds := []Condition{
		{Field: "provider", Mode: ConditionModeEq, Value: "openai", Invert: true},
	}
	ok, _ := EvaluateConditions(conds, map[string]string{"provider": "claude"}, ConditionLogicAnd)
	if !ok {
		t.Error("inverted eq should pass when different")
	}
}

func TestEvaluateConditions_PassMissingKey(t *testing.T) {
	// PassMissingKey=true 时缺失 key 视为通过
	conds := []Condition{
		{Field: "optional", Mode: ConditionModeEq, Value: "x", PassMissingKey: true},
	}
	ok, _ := EvaluateConditions(conds, map[string]string{"model": "gpt-4o"}, ConditionLogicAnd)
	if !ok {
		t.Error("pass_missing_key should pass when key absent")
	}

	// PassMissingKey=false（默认）时缺失 key 视为不通过
	conds[0].PassMissingKey = false
	ok, _ = EvaluateConditions(conds, map[string]string{"model": "gpt-4o"}, ConditionLogicAnd)
	if ok {
		t.Error("missing key without pass_missing_key should fail")
	}
}

func TestEvaluateConditions_And(t *testing.T) {
	conds := []Condition{
		{Field: "provider", Mode: ConditionModeEq, Value: "openai"},
		{Field: "model", Mode: ConditionModePrefix, Value: "gpt-4o"},
	}
	// 两个都满足
	ok, _ := EvaluateConditions(conds, map[string]string{"provider": "openai", "model": "gpt-4o"}, ConditionLogicAnd)
	if !ok {
		t.Error("AND: both match should pass")
	}
	// 只满足一个
	ok, _ = EvaluateConditions(conds, map[string]string{"provider": "openai", "model": "claude"}, ConditionLogicAnd)
	if ok {
		t.Error("AND: one match should fail")
	}
}

func TestEvaluateConditions_Or(t *testing.T) {
	conds := []Condition{
		{Field: "provider", Mode: ConditionModeEq, Value: "openai"},
		{Field: "provider", Mode: ConditionModeEq, Value: "claude"},
	}
	// 第一个满足
	ok, _ := EvaluateConditions(conds, map[string]string{"provider": "openai"}, ConditionLogicOr)
	if !ok {
		t.Error("OR: first match should pass")
	}
	// 第二个满足
	ok, _ = EvaluateConditions(conds, map[string]string{"provider": "claude"}, ConditionLogicOr)
	if !ok {
		t.Error("OR: second match should pass")
	}
	// 都不满足
	ok, _ = EvaluateConditions(conds, map[string]string{"provider": "gemini"}, ConditionLogicOr)
	if ok {
		t.Error("OR: none match should fail")
	}
}

func TestEvaluateConditions_DefaultLogicIsAnd(t *testing.T) {
	conds := []Condition{
		{Field: "provider", Mode: ConditionModeEq, Value: "openai"},
		{Field: "model", Mode: ConditionModeEq, Value: "gpt-4o"},
	}
	// 空逻辑默认 AND
	ok, _ := EvaluateConditions(conds, map[string]string{"provider": "openai", "model": "claude"}, "")
	if ok {
		t.Error("empty logic defaults to AND, should fail")
	}
}

func TestEvaluateConditions_ErrorPropagation(t *testing.T) {
	conds := []Condition{
		{Field: "x", Mode: "invalid_mode", Value: "y"},
	}
	_, err := EvaluateConditions(conds, map[string]string{"x": "val"}, ConditionLogicAnd)
	if err == nil {
		t.Error("unsupported mode should error")
	}
}

func TestCompareValues_NumericEquality(t *testing.T) {
	// JSON 数字 float64(1024) 与字符串 "1024" 应相等
	ok, err := compareValues("1024", float64(1024), string(ConditionModeEq))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("1024 == 1024 should match")
	}
}

func TestToString(t *testing.T) {
	cases := []struct {
		input interface{}
		want  string
	}{
		{"hello", "hello"},
		{float64(1024), "1024"},
		{float64(3.14), "3.14"},
		{int(42), "42"},
		{int64(100), "100"},
		{true, "true"},
		{false, "false"},
		{nil, ""},
	}
	for _, tt := range cases {
		got := toString(tt.input)
		if got != tt.want {
			t.Errorf("toString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
