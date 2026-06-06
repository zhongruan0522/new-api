# common/AGENTS.md

`common/` 是全局共享工具层，改动影响后端所有包。

## 规则

- JSON 序列化/反序列化调用必须走 `common/json.go` 的包装函数。
- 可以引用 `encoding/json` 的类型，例如 `json.RawMessage`，但不要直接调用 `json.Marshal`、
  `json.Unmarshal`、`json.NewDecoder` 等业务序列化函数。
- `GetTheme`/`SetTheme` 只接受 `default` 和 `classic`。不要在调用方绕过校验写入其他主题值。
- `EmbedFolder` 和 `NewThemeAwareFS` 是双 UI 静态资源服务的基础，改动后必须检查 `router/web-router.go`
  和两套 dist。
- URL、IP、SSRF、TLS、Redis、缓存、限流等工具处在安全边界，外部输入必须显式校验。
- 共享工具不要引入 controller/service/model 的反向依赖。

## 验证

- 改 JSON、主题、静态文件、URL 或缓存工具后执行 `go test ./common/... ./router/...`。
- 影响全局行为时执行 `go test ./...`。
