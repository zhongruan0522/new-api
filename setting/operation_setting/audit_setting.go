package operation_setting

import (
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/config"
)

// AuditSetting 审计日志全局配置。
// Modules 为 JSON 字符串（map[string]bool），key 是模块名，value 表示是否记录该模块。
// 采用字符串存储与 console_setting.ApiInfo 一致，便于跨 SQLite/MySQL/PostgreSQL 持久化。
type AuditSetting struct {
	Enabled    bool   `json:"enabled"`     // 审计总开关，默认关闭
	Modules    string `json:"modules"`     // 各模块开关 JSON 字符串，map[string]bool
	RecordIp   bool   `json:"record_ip"`   // 是否记录操作 IP
	RecordDiff bool   `json:"record_diff"` // 是否记录前后差异内容
}

// defaultAuditModules 构造默认模块开关：所有模块默认开启。
func defaultAuditModules() string {
	m := map[string]interface{}{
		"option":        true,
		"channel":       true,
		"user":          true,
		"token":         true,
		"redemption":    true,
		"model":         true,
		"vendor":        true,
		"dynamic_ratio": true,
		"prefill_group": true,
		"db":            true,
		"performance":   true,
		"log":           true,
		"setup":         true,
	}
	return common.MapToJsonStr(m)
}

// 默认配置：总开关关闭，其余字段为启用后即可生效的合理默认。
var auditSetting = AuditSetting{
	Enabled:    false,
	Modules:    defaultAuditModules(),
	RecordIp:   true,
	RecordDiff: true,
}

func init() {
	config.GlobalConfig.Register("audit_setting", &auditSetting)
}

// GetAuditSetting 返回审计配置实例指针。
func GetAuditSetting() *AuditSetting {
	return &auditSetting
}

// IsAuditEnabled 返回审计总开关状态。
func IsAuditEnabled() bool {
	return auditSetting.Enabled
}

// IsAuditModuleEnabled 检查指定模块是否需要记录审计。
// 当总开关关闭时直接返回 false；当模块映射解析失败时，保守地返回 false 并记录错误，
// 避免因为配置损坏而意外写入审计日志（违反"配置解析失败必须显式暴露"的原则）。
func IsAuditModuleEnabled(module string) bool {
	if !auditSetting.Enabled {
		return false
	}
	m := make(map[string]bool)
	if err := common.Unmarshal([]byte(auditSetting.Modules), &m); err != nil {
		common.SysError("failed to parse audit modules setting: " + err.Error())
		return false
	}
	enabled, ok := m[module]
	return ok && enabled
}

// IsAuditRecordIp 返回是否记录操作 IP。
func IsAuditRecordIp() bool {
	return auditSetting.RecordIp
}

// IsAuditRecordDiff 返回是否记录前后数据差异。
func IsAuditRecordDiff() bool {
	return auditSetting.RecordDiff
}
