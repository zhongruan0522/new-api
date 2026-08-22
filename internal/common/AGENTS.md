# internal/common/AGENTS.md

`internal/common/` 是业务全局变量与零碎工具的共享内核层（阶段 4 拆解后），
改动影响后端所有包。基础设施能力（db/redis/cache/email/security/runtime 等）
已迁至 `internal/infra/`，HTTP 边界工具已迁至 `internal/httpapi/` 根包，
本包只剩全局业务状态、跨层常量注册表与零碎纯工具。

## 当前内容

| 类别 | 文件 |
|---|---|
| 业务全局变量 | `constants.go`（开关、限流、SMTP/OAuth、角色/状态常量等）、`quota.go`、`topup_ratio.go`、`performance_config.go`、`timezone.go` |
| 跨层上下文键注册表 | `context_key.go`（`ContextKey` 类型与全部键常量，原 `internal/constant`，阶段 4 并入） |
| 环境变量读取工具 | `env.go`（`GetEnvOrDefault*`，供 app/infra/store 各层使用，不得移入 app —— infra 反向 import app 会成环） |
| 系统日志输出 | `sys_log.go`（`SysLog`/`SysError`/`FatalLog`/`LogStartupSuccess`，写 gin writer；与 `internal/infra/log` 的业务日志区分） |
| 模型/端点工具 | `model.go`、`api_type.go`、`endpoint_type.go`、`endpoint_defaults.go`、`audio.go` |
| 零碎工具 | `str.go`、`utils.go`、`hash.go`、`copy.go`、`rate_limit.go`、`page_info.go`、`custom_event.go` |

## 规则

- JSON 序列化/反序列化调用必须走 `pkg/jsonx` 的包装函数（原 `common/json.go` 与
  `common.StringToByteSlice` 已迁至 `pkg/jsonx`，`common` 包内部同样调用 `jsonx.*`）。
  可以引用 `encoding/json` 的类型（如 `json.RawMessage`），但不要直接调用
  `json.Marshal`/`json.Unmarshal`/`json.NewDecoder` 等序列化函数。
- 本包**只允许依赖** `pkg/`、`internal/domain/channel/constant` 与标准库；
  禁止 import `internal/infra/`、`internal/httpapi/`、`internal/domain/shared/`
  及任何业务包（`shared → common` 与 `infra/* → common` 是既定单向方向，反向即成环）。
- `sys_log.go` 留守本包而非并入 `internal/infra/log`：`infra/log` 的业务日志
  （`LogInfo` 等）依赖本包 `DebugEnabled`/`QuotaPerUnit` 与 `infra/runtime.RelayGo`，
  若 `SysLog` 家族迁入会形成 `common ↔ infra/log` 环（timezone/topup_ratio 使用 `SysError`）。
- `ContextKey` 键常量被 domain/store/infra/relay/httpapi 全层引用，注册表落位本包；
  新增键先确认归属（渠道域键考虑 `domain/channel`，relay 域键考虑 `relay/constant`）。
- 共享工具不要引入 controller/domain/store 的反向依赖。

## 验证

- 改全局变量、`ContextKey` 或工具后执行 `go test ./internal/common/...`。
- 影响全局行为时执行 `go test ./...`。
