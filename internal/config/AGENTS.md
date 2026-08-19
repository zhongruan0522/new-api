# internal/config/AGENTS.md

`internal/config/` 管理系统、运营、模型、倍率、性能和控制台配置（原 `setting/`）。

## 规则

- 配置通过现有 registry 模型（`internal/config/manager/` 的 ConfigManager）注册和读取，避免散落全局变量写入。
- 配置值必须有明确默认值、合法值校验和错误信息。
- 修改配置结构时同步 controller、前端系统设置页面和持久化逻辑。
- 倍率、价格、状态码区间、限流等配置解析失败必须显式返回错误，不能回落到看似成功的默认值。

## 审计配置

审计日志配置注册在 `internal/config/operation/audit_setting.go`，通过
`manager.GlobalConfig.Register("audit_setting", ...)` 注册。

- `Enabled`：审计总开关，默认关闭。
- `Modules`：各模块开关，JSON 字符串 `map[string]bool`，默认全部启用。
- `RecordIp`：是否记录操作 IP。
- `RecordDiff`：是否记录前后差异。

新增审计模块时必须在 `defaultAuditModules()` 中注册，详见
`internal/controller/AGENTS.md` 的"新增资源类型时的检查清单"。

## 验证

- 改配置解析或校验后执行 `go test ./internal/config/...`。
- 影响前端系统设置时同时构建前端。
