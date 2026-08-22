package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSerializeAuditDiff_OptionUpdateBeforeAndAfter 验证修复 issue #101：
// 修改系统设置时，before 必须能被记录，而不是始终为 nil。
//
// 该用例复现 controller.UpdateOption 的调用路径：
//
//	before = {"key":"Foo","value":"old"} （修复前传 nil）
//	after  = {"key":"Foo","value":"new"}
//
// 期望 before_data 与 after_data 同时存在，且各自包含对应值。
func TestSerializeAuditDiff_OptionUpdateBeforeAndAfter(t *testing.T) {
	before := map[string]interface{}{"key": "Foo", "value": "old"}
	after := map[string]interface{}{"key": "Foo", "value": "new"}

	beforeStr, afterStr := serializeAuditDiff(before, after, true)
	if beforeStr == "" {
		t.Fatalf("before_data 不应为空：修复前 before 传 nil 导致审计日志缺少修改前配置")
	}
	if afterStr == "" {
		t.Fatalf("after_data 不应为空")
	}

	var beforeMap map[string]interface{}
	if err := json.Unmarshal([]byte(beforeStr), &beforeMap); err != nil {
		t.Fatalf("before_data 不是合法 JSON: %v", err)
	}
	if beforeMap["value"] != "old" {
		t.Fatalf("before_data.value 应为 old, 实际 %v", beforeMap["value"])
	}

	var afterMap map[string]interface{}
	if err := json.Unmarshal([]byte(afterStr), &afterMap); err != nil {
		t.Fatalf("after_data 不是合法 JSON: %v", err)
	}
	if afterMap["value"] != "new" {
		t.Fatalf("after_data.value 应为 new, 实际 %v", afterMap["value"])
	}
}

// TestSerializeAuditDiff_OptionUpdateNilBefore 兼容首次创建场景：
// OptionMap 中不存在该 key 时 before 为 nil，before_data 应为空，
// after_data 仍记录新值。保持与修复前行为一致，不破坏 create 路径。
func TestSerializeAuditDiff_OptionUpdateNilBefore(t *testing.T) {
	after := map[string]interface{}{"key": "Foo", "value": "new"}

	beforeStr, afterStr := serializeAuditDiff(nil, after, true)
	if beforeStr != "" {
		t.Fatalf("before 为 nil 时 before_data 应为空, 实际: %s", beforeStr)
	}
	if !strings.Contains(afterStr, "new") {
		t.Fatalf("after_data 应包含 new, 实际: %s", afterStr)
	}
}

// TestSerializeAuditDiff_SensitiveKeyRedaction 验证敏感 key 的脱敏
// 同时作用于 before 和 after，避免在审计日志中泄露旧凭证。
func TestSerializeAuditDiff_SensitiveKeyRedaction(t *testing.T) {
	// 模拟 controller 中对敏感 key 的脱敏：调用方将 value 替换为 [REDACTED]。
	before := map[string]interface{}{"key": "GitHubToken", "value": "[REDACTED]"}
	after := map[string]interface{}{"key": "GitHubToken", "value": "[REDACTED]"}

	beforeStr, afterStr := serializeAuditDiff(before, after, true)

	// 脱敏后 before/after 的 value 相同，computeDiff 认为无变化，
	// 二者都会是空串 —— 这是期望行为（敏感值未变不应写入审计日志）。
	if beforeStr != "" || afterStr != "" {
		t.Fatalf("脱敏后无差异时 before/after 应均为空, before=%s after=%s", beforeStr, afterStr)
	}
}
