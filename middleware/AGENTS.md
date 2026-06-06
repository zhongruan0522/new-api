# middleware/AGENTS.md

`middleware/` 处理认证、限速、分发、安全校验、日志和请求上下文。

## 规则

- 认证、角色、用户上下文、请求 ID、限速 key 和安全校验必须显式失败，不要静默放行。
- 读取请求 body 后必须恢复给后续处理器，避免破坏 relay、文件上传或签名校验。
- 不要在全局 middleware 中引入会破坏 SSE、websocket、流式输出的压缩或缓存行为。
- 分发和限速逻辑要保持 Redis 与内存缓存路径一致。
- 日志不要写入 token、API key、OAuth secret、完整请求体等敏感信息。

## 验证

- 改认证、限速、分发、缓存或请求体处理后执行 `go test ./middleware/... ./controller/... ./relay/...`。
- 影响 streaming 时手动或测试覆盖 SSE/relay 流式路径。
