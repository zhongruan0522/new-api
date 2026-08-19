package console

import (
	"github.com/NookMux/NookMux/internal/config/manager"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

// UsageLogFieldKey 使用日志详情弹窗中可独立控制可见性的字段标识。
// 新增或删除字段时，需同步：
//  1. 此处的 UsageLogFieldDefaults 默认值表
//  2. 前端 web/src/features/usage-logs/lib/field-visibility.ts 的 DEFAULT_FIELDS
//  3. 前端配置页 usage-log-fields-section.tsx 的字段元数据
//  4. details-dialog.tsx 中对应字段的 isVisible(fieldKey) 判断
//
// 详见根 AGENTS.md "使用日志字段可见性" 章节。
const (
	UsageLogFieldRequestID           = "request_id"
	UsageLogFieldUpstreamRequestID   = "upstream_request_id"
	UsageLogFieldChannel             = "channel"
	UsageLogFieldRetryChain          = "retry_chain"
	UsageLogFieldToken               = "token"
	UsageLogFieldGroup               = "group"
	UsageLogFieldIPAddress           = "ip_address"
	UsageLogFieldResponseTime        = "response_time"
	UsageLogFieldClientHeaders       = "client_headers"
	UsageLogFieldRequestConversion   = "request_conversion"
	UsageLogFieldReasoningEffort     = "reasoning_effort"
	UsageLogFieldSystemPrompt        = "system_prompt_override"
	UsageLogFieldModelMapping        = "model_mapping"
	UsageLogFieldParameterOverride   = "parameter_override"
	UsageLogFieldBillingSource       = "billing_source"
	UsageLogFieldBillingDetails      = "billing_details"
	UsageLogFieldPriceTable          = "price_table"
	UsageLogFieldTieredPricing       = "tiered_pricing"
	UsageLogFieldViolationFee        = "violation_fee"
	UsageLogFieldRefundDetails       = "refund_details"
	UsageLogFieldSubscriptionBilling = "subscription_billing"
	UsageLogFieldTokenBreakdown      = "token_breakdown"
	UsageLogFieldAudioTokens         = "audio_tokens"
	UsageLogFieldTopupAudit          = "topup_audit"
	UsageLogFieldOperatorAdmin       = "operator_admin"
	UsageLogFieldStreamStatus        = "stream_status"
	UsageLogFieldContent             = "content"
)

// UsageLogFieldDefault 描述单个字段的默认可见性。
type UsageLogFieldDefault struct {
	Key         string `json:"key"`
	Admin       bool   `json:"admin"`
	User        bool   `json:"user"`
	NameZH      string `json:"name_zh"`
	Description string `json:"description"`
	Group       string `json:"group"`
}

// UsageLogFieldsDefaults 返回详情弹窗可独立配置可见性的字段默认值列表。
//
// 仅包含详情弹窗独有字段。同时出现在列表表格列和详情弹窗中的字段
// （channel/token/group/response_time/content）不在配置范围内——表格列字段始终
// 对普通用户可见，不需要也无法配置隐藏。详见根 AGENTS.md "使用日志字段可见性"。
func UsageLogFieldsDefaults() []UsageLogFieldDefault {
	return []UsageLogFieldDefault{
		// 基本信息
		{Key: UsageLogFieldRequestID, Admin: true, User: true, NameZH: "请求ID", Description: "本次请求的唯一标识", Group: "basic"},
		{Key: UsageLogFieldUpstreamRequestID, Admin: true, User: true, NameZH: "上游请求ID", Description: "上游供应商返回的请求ID", Group: "basic"},
		{Key: UsageLogFieldRetryChain, Admin: true, User: false, NameZH: "重试链路", Description: "请求在多渠道间的重试路径", Group: "basic"},
		{Key: UsageLogFieldIPAddress, Admin: true, User: true, NameZH: "IP地址", Description: "请求来源的客户端IP", Group: "basic"},
		// 请求信息
		{Key: UsageLogFieldClientHeaders, Admin: true, User: true, NameZH: "客户端请求头", Description: "HTTP-Referer、X-Title、UA", Group: "request"},
		{Key: UsageLogFieldRequestConversion, Admin: true, User: false, NameZH: "请求转换", Description: "协议转换路径与实际请求路径", Group: "request"},
		{Key: UsageLogFieldReasoningEffort, Admin: true, User: true, NameZH: "推理强度", Description: "模型的推理强度设置", Group: "request"},
		{Key: UsageLogFieldSystemPrompt, Admin: true, User: true, NameZH: "系统提示覆盖", Description: "是否覆盖了系统提示词", Group: "request"},
		{Key: UsageLogFieldModelMapping, Admin: true, User: true, NameZH: "模型映射", Description: "请求模型与实际上游模型的映射", Group: "request"},
		{Key: UsageLogFieldParameterOverride, Admin: true, User: true, NameZH: "参数覆盖", Description: "请求中被覆盖的参数列表", Group: "request"},
		// 计费
		{Key: UsageLogFieldBillingSource, Admin: true, User: false, NameZH: "计费来源", Description: "本地计费或上游响应计费", Group: "billing"},
		{Key: UsageLogFieldBillingDetails, Admin: true, User: true, NameZH: "计费详情", Description: "计费模式、倍率与总费用", Group: "billing"},
		{Key: UsageLogFieldPriceTable, Admin: true, User: true, NameZH: "当前价格表格", Description: "各计费项的数量、单价、小计", Group: "billing"},
		{Key: UsageLogFieldTieredPricing, Admin: true, User: true, NameZH: "阶梯定价详情", Description: "动态阶梯计费的匹配详情", Group: "billing"},
		{Key: UsageLogFieldViolationFee, Admin: true, User: true, NameZH: "违规费用", Description: "违规扣费的代码、标记与金额", Group: "billing"},
		{Key: UsageLogFieldRefundDetails, Admin: true, User: true, NameZH: "退款详情", Description: "退款的任务ID与原因", Group: "billing"},
		{Key: UsageLogFieldSubscriptionBilling, Admin: true, User: true, NameZH: "订阅计费", Description: "订阅实例的计费详情", Group: "billing"},
		// Token
		{Key: UsageLogFieldTokenBreakdown, Admin: true, User: true, NameZH: "Token明细", Description: "标准/缓存/多模态Token细分", Group: "token"},
		{Key: UsageLogFieldAudioTokens, Admin: true, User: true, NameZH: "音频Token", Description: "音频/文本的输入输出统计", Group: "token"},
		// 系统/管理
		{Key: UsageLogFieldTopupAudit, Admin: true, User: false, NameZH: "充值审计", Description: "充值订单的支付方式、回调IP等", Group: "system"},
		{Key: UsageLogFieldOperatorAdmin, Admin: true, User: false, NameZH: "操作管理员", Description: "执行管理操作的管理员信息", Group: "system"},
		{Key: UsageLogFieldStreamStatus, Admin: true, User: false, NameZH: "流式状态", Description: "流式响应的状态与错误信息", Group: "system"},
	}
}

type ConsoleSetting struct {
	ApiInfo              string `json:"api_info"`              // 控制台 API 信息 (JSON 数组字符串)
	UptimeKumaGroups     string `json:"uptime_kuma_groups"`    // Uptime Kuma 分组配置 (JSON 数组字符串)
	Announcements        string `json:"announcements"`         // 系统公告 (JSON 数组字符串)
	FAQ                  string `json:"faq"`                   // 常见问题 (JSON 数组字符串)
	ApiInfoEnabled       bool   `json:"api_info_enabled"`      // 是否启用 API 信息面板
	UptimeKumaEnabled    bool   `json:"uptime_kuma_enabled"`   // 是否启用 Uptime Kuma 面板
	AnnouncementsEnabled bool   `json:"announcements_enabled"` // 是否启用系统公告面板
	FAQEnabled           bool   `json:"faq_enabled"`           // 是否启用常见问答面板

	// 使用日志详情弹窗字段可见性配置 (JSON 字符串，格式见 UsageLogFieldsSchema)
	UsageLogFields string `json:"usage_log_fields"`
	// 管理员总开关：关闭后管理员无法访问使用日志详情弹窗
	UsageLogFieldsAdminEnabled bool `json:"usage_log_fields_admin_enabled"`
	// 普通用户总开关：关闭后普通用户无法访问使用日志详情弹窗
	UsageLogFieldsUserEnabled bool `json:"usage_log_fields_user_enabled"`
}

// 默认配置
var defaultConsoleSetting = ConsoleSetting{
	ApiInfo:                    "",
	UptimeKumaGroups:           "",
	Announcements:              "",
	FAQ:                        "",
	ApiInfoEnabled:             true,
	UptimeKumaEnabled:          true,
	AnnouncementsEnabled:       true,
	FAQEnabled:                 true,
	UsageLogFields:             "",
	UsageLogFieldsAdminEnabled: true,
	UsageLogFieldsUserEnabled:  true,
}

// 全局实例
var consoleSetting = defaultConsoleSetting

func init() {
	// 注册到全局配置管理器，键名为 console_setting
	manager.GlobalConfig.Register("console_setting", &consoleSetting)
}

// GetConsoleSetting 获取 ConsoleSetting 配置实例
func GetConsoleSetting() *ConsoleSetting {
	return &consoleSetting
}

// UsageLogFieldVisibleConfig 表示一个字段在管理员/普通用户视角的可见性。
type UsageLogFieldVisibleConfig struct {
	Admin bool `json:"admin"`
	User  bool `json:"user"`
}

// parseUsageLogFields 解析 UsageLogFields JSON 字符串为 map。
// 如果配置为空或解析失败，返回 nil 和 error，不静默回退。
func parseUsageLogFields(raw string) (map[string]UsageLogFieldVisibleConfig, error) {
	if raw == "" {
		// 空配置视为未配置，由调用方决定是否使用默认值
		return nil, nil
	}
	m := make(map[string]UsageLogFieldVisibleConfig)
	if err := jsonx.UnmarshalJsonStr(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// removedUsageLogFields 是已从配置范围移除的字段集合。
// 这些字段同时出现在列表表格列和详情弹窗中，产品决策是表格列字段始终可见、
// 不可配置隐藏。旧配置 JSON 可能仍包含这些 key，运行时必须剔除，
// 否则 IsUsageLogFieldVisible 会读到遗留值而非回退到 return false。
var removedUsageLogFields = map[string]bool{
	UsageLogFieldChannel:      true,
	UsageLogFieldToken:        true,
	UsageLogFieldGroup:        true,
	UsageLogFieldResponseTime: true,
	UsageLogFieldContent:      true,
}

// GetUsageLogFieldsVisible 返回当前配置的字段可见性 map。
// 如果配置为空（未配置），返回基于默认值的 map。
// 如果配置解析失败，返回 error。
// 已从配置范围移除的字段（removedUsageLogFields）会被剔除，不返回给调用方。
func GetUsageLogFieldsVisible() (map[string]UsageLogFieldVisibleConfig, error) {
	m, err := parseUsageLogFields(consoleSetting.UsageLogFields)
	if err != nil {
		return nil, err
	}
	if m == nil {
		// 配置为空，使用默认值
		defaults := UsageLogFieldsDefaults()
		m = make(map[string]UsageLogFieldVisibleConfig, len(defaults))
		for _, d := range defaults {
			m[d.Key] = UsageLogFieldVisibleConfig{Admin: d.Admin, User: d.User}
		}
	}
	// 剔除已移除字段，防止旧配置的遗留值绕过安全规则
	for key := range removedUsageLogFields {
		delete(m, key)
	}
	return m, nil
}

// IsUsageLogFieldVisible 查询指定字段在指定角色下的可见性。
// 如果字段不存在于配置中，回退到默认值。
func IsUsageLogFieldVisible(field string, isAdmin bool) bool {
	m, err := GetUsageLogFieldsVisible()
	if err != nil {
		// 解析失败时保守返回默认值，避免因配置损坏导致弹窗空白
		defaults := UsageLogFieldsDefaults()
		for _, d := range defaults {
			if d.Key == field {
				if isAdmin {
					return d.Admin
				}
				return d.User
			}
		}
		return false
	}
	cfg, ok := m[field]
	if !ok {
		// 字段不在配置中，回退到默认值
		defaults := UsageLogFieldsDefaults()
		for _, d := range defaults {
			if d.Key == field {
				if isAdmin {
					return d.Admin
				}
				return d.User
			}
		}
		return false
	}
	if isAdmin {
		return cfg.Admin
	}
	return cfg.User
}

// IsUsageLogDetailsEnabled 返回指定角色是否可以访问使用日志详情弹窗。
func IsUsageLogDetailsEnabled(isAdmin bool) bool {
	if isAdmin {
		return consoleSetting.UsageLogFieldsAdminEnabled
	}
	return consoleSetting.UsageLogFieldsUserEnabled
}
