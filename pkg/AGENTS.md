# pkg/AGENTS.md

`pkg/` 存放可独立复用、无业务依赖的基础库。进入门槛是依赖核查通过：
包内任何文件（含测试）不得 import `internal/` 下任何业务包（含
`internal/common`、`internal/domain/` 等）或其他业务代码。

## 现有包

- `jsonx/`: JSON 序列化/反序列化包装函数（`Marshal` / `Unmarshal` /
  `UnmarshalJsonStr` / `DecodeJson` / `GetJsonType`）及零拷贝 `StringToByteSlice`。
  全仓 JSON 调用统一走 `jsonx`，业务代码不直接调用 `encoding/json` 的
  marshal/unmarshal/decode。
- `cachex/`: 底层缓存原语。当前唯一调用方是 `internal/domain/channel/channel_affinity.go`（阶段 5.3 迁入），
  依赖干净故保留在 `pkg/`；一旦长出业务依赖即降级 `internal/`。

## 规则

- 依赖核查必须看符号级依赖而非仅 import 语句：候选文件即使直接 import 只有
  标准库，若引用业务包内定义的函数/变量/错误值（如磁盘缓存配置、`SysError`、
  `ErrRequestBodyTooLarge`），也不得进 `pkg/`。先例见
  [docs/PRD/prd-architecture-migration.md](../docs/PRD/prd-architecture-migration.md)
  阶段 2 对 `internal/common/body_storage.go` 的核查结论（暂留，阶段 4 已随迁
  `internal/infra/cache/`）。
- 禁止为通过核查而在迁移时改写业务逻辑（依赖注入等重构不在 `pkg/` 抽离范围内）。

## 验证

- `go build ./... && go test ./pkg/...`
- `go list -deps ./pkg/...` 输出不出现 `github.com/NookMux/NookMux/` 前缀下的
  `internal/` 任何路径（含 `domain`、`common`、`infra`）。
