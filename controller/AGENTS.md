# controller/AGENTS.md

`controller/` 是 HTTP 边界层，负责输入校验、权限检查、调用 service/model、组织响应。

## 规则

- 外部输入必须在这里或更近边界校验：path/query/body/form/file/header。
- 控制器不要沉淀复杂业务逻辑；可复用业务放 `service/`，持久化放 `model/`。
- 响应结构保持现有 `{ success, message, data }` 风格，避免为单个前端新增不兼容格式。
- 不要为了前端改后端业务 API。字段不匹配时优先改前端适配本项目接口。
- 安全相关控制器要保留二次验证、角色校验、限速和审计日志。

## 审计日志埋点

管理员对系统资源的增删改操作必须接入审计日志。调用
`service.RecordAudit(c, module, actionType, description, before, after)`，
参数说明见 `service/AGENTS.md`。

### 何时埋点

- **新增**（create）：渠道、用户、令牌、兑换码、模型、供应商、动态倍率规则、预填充分组、系统初始化等资源的创建接口。
- **修改**（update）：上述资源的更新、启停、状态变更、额度调整、密钥重置等接口。
- **删除**（delete）：上述资源的单删、批量删除、清空等接口。
- 不审计只读操作（查询、测试、余额查询、模型拉取预览）。
- 不审计普通用户的自助操作（改自己信息、签到、充值、Passkey/2FA 自助管理）。

### 埋点位置

审计调用必须在 **业务操作成功之后、HTTP 响应之前** 插入。如果操作有多个成功
分支（如 `ManageUser` 的 `enable/disable/delete/add_quota`），每个分支都要埋点。

### before / after 数据约定

- **before**：操作前的原始数据。如果代码已在更新前查询了原记录（如 `GetChannelById`），
  直接传该对象；否则在更新前查一次。值类型传值拷贝（`origin := *ptr`）或 JSON map 快照，
  避免指针修改导致 before 数据被污染。
- **after**：操作后的新数据。传请求体或更新后的对象均可。`status_only` 等部分更新
  场景应基于 origin 构造 after（如 `afterModel := origin; afterModel.Status = newStatus`），
  避免请求体零值字段产生噪声 diff。
- 纯新增操作 before 传 `nil`；纯删除操作 after 传 `nil` 或简要元数据 map。
- 敏感字段（key、password、token、secret 等）由 `service.RecordAudit` 自动脱敏，
  controller 无需额外处理；但系统设置更新（`UpdateOption`）需要对值本身判断是否敏感。

### 审计配置变更的特殊处理

修改 `audit_setting.*` 配置时，由于 `model.UpdateOption` 先于 `RecordAudit` 执行，
新配置会立即生效。如果管理员关闭了审计总开关或 option 模块，后续的 `RecordAudit`
会被跳过。因此 `UpdateOption` 中对 `audit_setting.*` 的变更必须传 `forceRecord=true`。

### 新增资源类型时的检查清单

新增一个需要审计的资源类型时，确认：
1. 定义 `model.AuditModule*` 常量，加入 `model.AuditModuleList`。
2. 在 `setting/operation_setting/audit_setting.go` 的 `defaultAuditModules()` 中注册。
3. 在 `web/src/features/audit-logs/constants.ts` 的 `AUDIT_MODULES` 中注册。
4. 在 `web/src/i18n/static-keys.ts` 中注册模块标签 key。
5. 在 `web/src/i18n/locales/{en,zh}.json` 中添加翻译。
6. 在涉及的 controller 函数中添加 `service.RecordAudit` 调用。

## 使用日志字段可见性

管理员可在"系统设置 → 控制台内容 → 使用日志字段"中配置详情弹窗各字段对管理员/普通用户的可见性。

- 后端配置：`setting/console_setting/config.go` 的 `UsageLogFields`（JSON map）、`UsageLogFieldsAdminEnabled`、`UsageLogFieldsUserEnabled`。
- 字段定义：`UsageLogFieldsDefaults()` 返回所有字段的 key、默认值、中文名和描述。
- 校验：`setting/console_setting/validation.go` 的 `validateUsageLogFields`。
- 查询：`console_setting.IsUsageLogFieldVisible(field, isAdmin)` 和 `console_setting.IsUsageLogDetailsEnabled(isAdmin)`。
- 公开接口：`GET /api/user/self/usage_log_fields`（UserAuth），返回当前角色可见的字段列表。
- 审计：通过 `UpdateOption` 的通用 `RecordAudit` 自动覆盖，复用 `AuditModuleOption`。

**新增或删除字段时需同步：**

1. 后端 `UsageLogFieldsDefaults()` 添加/移除条目。
2. 后端 `UsageLogField*` 常量。
3. 前端 `web/src/features/usage-logs/lib/field-visibility.ts` 的 `USAGE_LOG_FIELD_KEYS`、`USAGE_LOG_FIELDS`。
4. 前端 `web/src/features/usage-logs/components/dialogs/details-dialog.tsx` 中对应字段的条件渲染改为 `isVisible('<fieldKey>')`。

## 验证

- 改请求校验、权限或响应字段后执行对应 controller 测试。
- 影响系统设置、登录、渠道、令牌、计费、文件或 relay 入口时执行 `go test ./controller/... ./service/...`。
- 新增或修改审计埋点后执行 `go build .` 确认编译通过。
