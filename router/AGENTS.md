# router/AGENTS.md

`router/` 只负责挂载路由和静态资源，不承载业务逻辑。

## Web 路由

- `SetWebRouter` 使用 `WebAssets` 和 `common.EmbedFolder` 服务 `web/dist`。
- `NoRoute` 中 `/v1`、`/api`、`/assets` 仍应返回 relay/API 404，不要误返回前端 HTML。
- 分析脚本注入在 `main.go` 中直接修改 `indexPage` 字节。
- `FRONTEND_BASE_URL` 只在非 master 节点生效，保持现有重定向行为。

## 路由边界

- API、dashboard、relay、web 路由分层清晰；新增业务接口优先放对应 router 文件。
- 不要在路由层解析复杂业务参数或访问数据库。
- 不要添加会破坏 SSE/streaming 的全局 gzip；web 静态资源 gzip 只在 web router 中处理。

## 验证

- 改 web router 或 embed 资源后执行 `go test ./router/... ./common/...` 和 `go build`。
- 如果影响 relay 路由，执行相关 `relay` 测试并检查流式响应路径。
