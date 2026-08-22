# cmd/server/AGENTS.md

`cmd/server/` 只保留可执行入口。

## 规则

- `main.go` 只允许处理进程退出码并调用 `internal/app.Run()`。
- 启动参数解析、资源初始化、HTTP 装配、后台任务和路由挂载都不要放入该目录。
- 新增可执行命令时，先确认是否确有独立进程边界；不要为库代码创建空命令。

## 验证

- 修改入口后执行 `go test ./cmd/server/... ./internal/app/...` 和 `go build ./cmd/server`。
- 修改启动装配时额外执行 `go run ./cmd/server` 的启动冒烟验证，并确认进程可正常退出。
