package model

// AuditLog 审计日志，记录管理员对系统资源的增删改操作。
// 仅持久化到主库（DB），不写入 LOG_DB，避免与消费日志混淆。
type AuditLog struct {
	Id          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index:idx_audit_created_at"`
	Username    string `json:"username" gorm:"type:varchar(255);index:idx_audit_username"`
	Ip          string `json:"ip" gorm:"type:varchar(64);index:idx_audit_ip"`
	Module      string `json:"module" gorm:"type:varchar(32);index:idx_audit_module"`
	ActionType  string `json:"action_type" gorm:"type:varchar(16);index:idx_audit_action_type"`
	Description string `json:"description" gorm:"type:varchar(255)"`
	BeforeData  string `json:"before_data,omitempty" gorm:"type:text"`
	AfterData   string `json:"after_data,omitempty" gorm:"type:text"`
}

// 审计模块常量。取值需稳定，前端与控制器通过常量引用，避免拼写不一致。
const (
	AuditModuleOption          = "option"           // 系统设置
	AuditModuleChannel         = "channel"          // 渠道
	AuditModuleUser            = "user"             // 用户
	AuditModuleToken           = "token"            // 令牌
	AuditModuleRedemption      = "redemption"       // 兑换码
	AuditModuleModel           = "model"            // 模型
	AuditModuleVendor          = "vendor"           // 供应商
	AuditModuleDynamicRatio    = "dynamic_ratio"    // 动态倍率
	AuditModulePrefillGroup    = "prefill_group"    // 预填充分组
	AuditModuleDB              = "db"               // 数据库迁移
	AuditModulePerformance     = "performance"      // 性能管理
	AuditModuleLog             = "log"              // 日志清理
	AuditModuleSetup           = "setup"            // 系统初始化
	AuditModuleDashboardConfig = "dashboard_config" // 仪表板配置
	AuditModuleVoice           = "voice"            // 音色管理
)

// 审计操作类型常量。
const (
	AuditActionCreate = "create"
	AuditActionUpdate = "update"
	AuditActionDelete = "delete"
)

// AuditModuleInfo 描述一个审计模块的值与中文显示名，供前端渲染筛选项。
type AuditModuleInfo struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// AuditModuleList 按业务约定顺序列出全部审计模块，前端可直接消费。
var AuditModuleList = []AuditModuleInfo{
	{AuditModuleOption, "系统设置"},
	{AuditModuleChannel, "渠道"},
	{AuditModuleUser, "用户"},
	{AuditModuleToken, "令牌"},
	{AuditModuleRedemption, "兑换码"},
	{AuditModuleModel, "模型"},
	{AuditModuleVendor, "供应商"},
	{AuditModuleDynamicRatio, "动态倍率"},
	{AuditModulePrefillGroup, "预填充分组"},
	{AuditModuleDB, "数据库迁移"},
	{AuditModulePerformance, "性能管理"},
	{AuditModuleLog, "日志清理"},
	{AuditModuleSetup, "系统初始化"},
	{AuditModuleDashboardConfig, "仪表板配置"},
	{AuditModuleVoice, "音色管理"},
}

// CreateAuditLog 插入一条审计日志。
func CreateAuditLog(auditLog *AuditLog) error {
	return DB.Create(auditLog).Error
}

// GetAllAuditLogs 分页查询审计日志。所有过滤参数可选，零值表示不限。
// 返回日志列表、符合条件的总条数和错误。
func GetAllAuditLogs(username, module, actionType string, startTime, endTime int64, page, pageSize int) ([]*AuditLog, int64, error) {
	var logs []*AuditLog
	var total int64

	tx := DB.Model(&AuditLog{})
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if module != "" {
		tx = tx.Where("module = ?", module)
	}
	if actionType != "" {
		tx = tx.Where("action_type = ?", actionType)
	}
	if startTime != 0 {
		tx = tx.Where("created_at >= ?", startTime)
	}
	if endTime != 0 {
		tx = tx.Where("created_at <= ?", endTime)
	}

	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := tx.Order("id desc").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// DeleteOldAuditLogs 删除 created_at 早于 targetTimestamp 的审计日志。
// 已被 DeleteAuditLogsInRange 取代，保留用于兼容。
// 返回被删除的行数。
func DeleteOldAuditLogs(targetTimestamp int64) (int64, error) {
	return DeleteAuditLogsInRange(0, targetTimestamp)
}

// DeleteAuditLogsInRange 删除 created_at 在 [startTimestamp, endTimestamp] 区间内的审计日志。
// startTimestamp 为 0 表示不限下界，endTimestamp 为 0 表示不限上界。
// 返回被删除的行数。
func DeleteAuditLogsInRange(startTimestamp, endTimestamp int64) (int64, error) {
	tx := DB.Model(&AuditLog{})
	if startTimestamp > 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	result := tx.Delete(&AuditLog{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
