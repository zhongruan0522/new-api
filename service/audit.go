package service

import (
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// sensitiveFieldSuffixes 字段名包含这些子串（小写匹配）时需要脱敏。
// 覆盖精确名称（如 key、password）和变体（如 api_key、x-goog-api-key、proxy-authorization）。
var sensitiveFieldSubstrings = []string{
	"key",
	"password",
	"token",
	"secret",
	"credential",
	"authorization",
	"private_key",
}

// isSensitiveField 判断字段名是否匹配敏感模式。
func isSensitiveField(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, sub := range sensitiveFieldSubstrings {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

// RecordAudit 记录审计日志。
//
// 当审计总开关关闭、目标模块未启用、或上下文为空（非 HTTP 请求路径）时，直接返回不记录。
// before/after 会被尝试归一化为 map[string]interface{} 以计算字段差异；
// 如果二者无法归一化或差异计算无意义，则按各自原始 JSON 序列化保存。
// 数据库写入通过 gopool 异步执行，审计失败不影响业务，仅通过 common.SysError 记录。
//
// forceRecord 为 true 时跳过审计开关检查，用于确保"关闭审计"这类关键操作本身被记录。
func RecordAudit(c *gin.Context, module, actionType, description string, before, after interface{}, forceRecord ...bool) {
	if c == nil {
		return
	}
	// forceRecord 变长参数：仅当显式传 true 时才跳过开关检查。
	force := len(forceRecord) > 0 && forceRecord[0]
	if !force {
		if !operation_setting.IsAuditEnabled() {
			return
		}
		if !operation_setting.IsAuditModuleEnabled(module) {
			return
		}
	}

	username := c.GetString("username")
	ip := ""
	if force || operation_setting.IsAuditRecordIp() {
		ip = c.ClientIP()
	}

	recordDiff := force || operation_setting.IsAuditRecordDiff()
	beforeStr, afterStr := serializeAuditDiff(before, after, recordDiff)

	auditLog := &model.AuditLog{
		CreatedAt:   common.GetTimestamp(),
		Username:    username,
		Ip:          ip,
		Module:      module,
		ActionType:  actionType,
		Description: description,
		BeforeData:  beforeStr,
		AfterData:   afterStr,
	}

	gopool.Go(func() {
		if err := model.CreateAuditLog(auditLog); err != nil {
			common.SysError(fmt.Sprintf("failed to record audit log (module=%s, action=%s): %s", module, actionType, err.Error()))
		}
	})
}

// serializeAuditDiff 将 before/after 序列化为审计日志所需的差异 JSON 字符串。
//
// 行为：
//   - 若 recordDiff 为 false，返回空串，不记录差异内容。
//   - 若 before 与 after 都能归一化为 map，则使用 computeDiff 仅保留变化字段。
//   - 否则按原始值整体序列化，before/after 为 nil 时返回空串。
func serializeAuditDiff(before, after interface{}, recordDiff bool) (string, string) {
	if !recordDiff {
		return "", ""
	}

	beforeMap, beforeOk := normalizeToMap(before)
	afterMap, afterOk := normalizeToMap(after)

	// 脱敏：移除敏感字段（密钥、密码等），防止泄露到审计日志。
	if beforeOk {
		sanitizeSecrets(beforeMap)
	}
	if afterOk {
		sanitizeSecrets(afterMap)
	}

	// 双方都是 map 时计算字段差异，避免保存未变化字段。
	if beforeOk && afterOk {
		diffBefore, diffAfter := computeDiff(beforeMap, afterMap)
		return mapToJsonStr(diffBefore), mapToJsonStr(diffAfter)
	}

	// 如果某一方能归一化为 map，使用脱敏后的 map 序列化；否则直接序列化原始值。
	return serializeSide(before, beforeMap, beforeOk), serializeSide(after, afterMap, afterOk)
}

// serializeSide 将单侧数据序列化为 JSON 字符串。如果能归一化为 map 则用脱敏后的 map，否则用原始值。
func serializeSide(original interface{}, m map[string]interface{}, ok bool) string {
	if !ok {
		return marshalIfPresent(original)
	}
	return mapToJsonStr(m)
}

// normalizeToMap 将任意值归一化为 map[string]interface{}。
// 支持 map[string]interface{}、map[string]bool 等具体类型；struct 或 JSON 字符串通过 common.Marshal 再 Unmarshal 转换。
// 第二个返回值表示是否成功归一化为非空 map。
func normalizeToMap(v interface{}) (map[string]interface{}, bool) {
	if v == nil {
		return nil, false
	}
	// 直接类型断言，覆盖 JSON 反序列化产生的 map[string]interface{}。
	if m, ok := v.(map[string]interface{}); ok {
		return m, true
	}
	// 通过反射无法直接判断 map[string]XXX，借助 JSON 往返转换统一处理。
	bytes, err := common.Marshal(v)
	if err != nil {
		return nil, false
	}
	var m map[string]interface{}
	if err := common.Unmarshal(bytes, &m); err != nil {
		return nil, false
	}
	return m, true
}

// computeDiff 计算两个 map 之间的字段差异。
//
// 规则：
//   - 遍历 after：值与 before 不同（或 before 缺失），在 beforeDiff/afterDiff 中同时记录双方值。
//   - 遍历 before 中存在但 after 中缺失的 key：标记为已删除，afterDiff 中以 nil 表示。
//   - 未变化的字段不会出现在任何结果中。
//
// 当 before 为空（创建场景）时，beforeDiff 为空 map，afterDiff 等于 after 的浅拷贝。
func computeDiff(before, after map[string]interface{}) (map[string]interface{}, map[string]interface{}) {
	beforeDiff := make(map[string]interface{})
	afterDiff := make(map[string]interface{})

	for k, afterVal := range after {
		beforeVal, exists := before[k]
		if !exists {
			beforeDiff[k] = nil
			afterDiff[k] = afterVal
			continue
		}
		if !valueEqual(beforeVal, afterVal) {
			beforeDiff[k] = beforeVal
			afterDiff[k] = afterVal
		}
	}

	for k, beforeVal := range before {
		if _, exists := after[k]; !exists {
			beforeDiff[k] = beforeVal
			afterDiff[k] = nil
		}
	}

	return beforeDiff, afterDiff
}

// valueEqual 比较两个 interface{} 是否相等。
// 由于 JSON 反序列化后数字统一为 float64，这里借助 JSON 序列化做深度比较，避免 map/slice 引用差异。
func valueEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	ab, _ := common.Marshal(a)
	bb, _ := common.Marshal(b)
	return string(ab) == string(bb)
}

// marshalIfPresent 将非 nil 值序列化为 JSON 字符串，nil 返回空串。
func marshalIfPresent(v interface{}) string {
	if v == nil {
		return ""
	}
	bytes, err := common.Marshal(v)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// mapToJsonStr 将 map 序列化为 JSON 字符串，空 map 返回空串以保持与 omitempty 语义一致。
func mapToJsonStr(m map[string]interface{}) string {
	if len(m) == 0 {
		return ""
	}
	bytes, err := common.Marshal(m)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// sanitizeSecrets 递归清除 map 中名称匹配敏感字段列表的条目。
// 对嵌套的 map[string]interface{} 和 []interface{} 递归处理。
// 当字段值是 JSON 字符串时（如渠道的 other、header_override 等字符串化配置），
// 尝试解析后递归脱敏再序列化回去，确保深层结构中的敏感字段也被清除。
func sanitizeSecrets(m map[string]interface{}) {
	for k, v := range m {
		if isSensitiveField(k) {
			m[k] = "[REDACTED]"
			continue
		}
		m[k] = sanitizeValue(v)
	}
}

// sanitizeValue 对任意值递归脱敏，返回处理后的值。
func sanitizeValue(v interface{}) interface{} {
	switch child := v.(type) {
	case map[string]interface{}:
		sanitizeSecrets(child)
		return child
	case []interface{}:
		sanitizeSecretsInSlice(child)
		return child
	case string:
		// 尝试解析 JSON 字符串，如果成功则递归脱敏后序列化回去。
		// 只对看起来像 JSON 对象/数组的字符串处理，避免对普通文本做无意义解析。
		if len(child) > 0 && (child[0] == '{' || child[0] == '[') {
			var parsed interface{}
			if err := common.Unmarshal([]byte(child), &parsed); err == nil {
				sanitizeValue(parsed)
				if bytes, err := common.Marshal(parsed); err == nil {
					return string(bytes)
				}
			}
		}
		return child
	default:
		return child
	}
}

func sanitizeSecretsInSlice(s []interface{}) {
	for i, item := range s {
		s[i] = sanitizeValue(item)
	}
}
