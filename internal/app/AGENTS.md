# internal/app/AGENTS.md

`internal/app/` 是启动装配层，负责资源初始化、后台任务装配、HTTP server 装配和分析脚本注入，不承载具体业务规则。

## 结构

- `bootstrap.go`：环境、日志、数据库、配置、缓存、监控和 i18n 初始化；保持原启动顺序。
- `env.go`：启动 flag（`--port`/`--version`/`--help`/`--log-dir`）与 `InitEnv` 环境装配
  （阶段 4 自 `common/init.go` 迁入）。`--log-dir` 解析结果通过 `infra/log.Dir` 注入
  日志包（infra 不得反向 import app）。
- `server.go`：Gin server、session、后台任务和路由装配；进程退出码在这里统一返回给 `cmd/server`。
- `analytics.go`：Umami / Google Analytics 注入；函数接收并返回 index 字节，不持有可变全局状态。
- `webdist/`：给应用层提供前端嵌入资产的门面，含静态资源 `EmbedFolder`
  （阶段 4 自 `common/embed_file_system.go` 迁入，`httpapi/router` 经此构建 web FS）。
  Go 的 `//go:embed` 不支持 `..`，因此实际声明必须位于 `web/embed.go`，由本包转成
  `internal` API；不要把 `web/dist` 复制到本目录或改回根入口。

## 规则

- 阶段一内的业务包 import 路径保持不变，不在本层顺手迁移业务代码。
- 启动顺序变更必须逐项说明原因，特别不要破坏数据库初始化、配置热更新、渠道缓存、i18n 和监控的依赖关系。
- 分析脚本注入必须保持占位符替换语义，并继续保护 API/SSE 路由不被前端静态路由吞掉。
- 修改前端嵌入载体时同步检查 `web/dist/index.html`、`internal/httpapi/router/web_router.go` 和 Docker 构建检查。

## 验证

- `go test ./internal/app/... ./internal/httpapi/router/... ./internal/common/...`
- `go build ./...`
- 修改启动或嵌入路径后执行 `go run ./cmd/server` 冒烟验证。
