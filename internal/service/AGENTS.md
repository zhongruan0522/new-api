# internal/service/AGENTS.md

`internal/service/` 承载业务逻辑、外部请求、计费、配额、迁移编排和文件处理。

## 规则

- 控制器输入已校验也不能假设内部状态可信；跨系统边界继续校验。
- 数据库写入通过 `internal/store/` 各资源子包或 GORM 约定处理，不要在 service 中散落数据库专有 SQL。
- 计费、quota、usage、渠道选择、动态倍率、违规费用等逻辑必须保持可追踪和可测试。
- 外部 HTTP 调用复用现有客户端和超时配置；不要无超时请求或吞掉上游错误。
- 文件下载、解析、存储要校验大小、类型、来源和错误路径。
- 不要模拟成功或生成假 usage 来掩盖上游/业务失败。

## 审计日志服务

`service.RecordAudit` 是审计日志的唯一入口，controller 调用它记录管理员操作。

### 签名

```go
func RecordAudit(c *gin.Context, module, actionType, description string, before, after interface{}, forceRecord ...bool)
```

- `module`：`auditstore.AuditModule*` 常量（如 `auditstore.AuditModuleChannel`，包路径 `internal/store/audit`）。
- `actionType`：`auditstore.AuditActionCreate` / `auditstore.AuditActionUpdate` / `auditstore.AuditActionDelete`。
- `before`/`after`：操作前后的数据，传 struct、map 或 nil 均可。service 层会：
  - 自动归一化为 map 并计算字段级 diff，只保留变化字段。
  - 自动递归脱敏敏感字段（字段名包含 `key`/`password`/`token`/`secret`/`credential`/`authorization`/`private_key` 的值替换为 `[REDACTED]`）。
  - 自动解析字符串化的 JSON 字段（如渠道的 `other`、`header_override`）并递归脱敏。
- `forceRecord`：传 `true` 时跳过审计开关检查。仅用于审计配置本身的变更
  （`audit_setting.*`），确保"关闭审计"操作本身被记录。

### 行为约定

- 审计总开关关闭（`audit_setting.enabled=false`）或目标模块未启用时，`RecordAudit` 直接返回。
- `record_ip=false` 时不记录 IP；`record_diff=false` 时不记录 before/after 数据。
- 数据库写入通过 `gopool.Go` 异步执行，失败仅记录 `common.SysError`，不影响业务。
- 无鉴权接口（如 `PostSetup`）需要在调用前通过 `c.Set("username", ...)` 设置操作人。

## 验证

- 改计费、quota、倍率、渠道选择或迁移逻辑后执行对应 service/model 测试。
- 影响跨包行为时执行 `go test ./internal/service/... ./internal/store/... ./internal/relay/...`。
- 改审计服务（`internal/service/audit.go`）后执行 `go build .` 和 `go vet ./internal/service/...`。
