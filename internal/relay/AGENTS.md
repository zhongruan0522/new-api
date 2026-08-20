# internal/relay/AGENTS.md

`internal/relay/` 是 AI API 中继和供应商适配核心，改动风险高。

## 规则

- 保持协议边界清晰：OpenAI wire、Responses、Chat Completions、Claude、Gemini、AWS 等转换不要混写。
- 流式输出必须保护 chunk 顺序、错误事件、finish reason、usage 和连接关闭行为。
- 计费、预扣、补扣、缓存倍率、音频/图片/视频/embedding/rerank 的 usage 不能随意近似。
- 供应商适配放在 `internal/relay/channel/<provider>/`，共享转换放 `internal/relay/common/` 或 `internal/relay/helper/`（`helper/` 含 Claude/Gemini ↔ OpenAI 协议转换、relay 错误包装与响应透传工具，阶段 5.3 自原 `service/` 迁入）。
- 新增 channel 时确认供应商是否支持 `stream_options` 等流式选项，并同步相关 capability 判断。
- 请求 DTO 中需要重新 marshal 给上游的可选标量字段，使用 `*int`、`*uint`、`*float64`、`*bool` 等指针类型加 `omitempty`，避免显式零值被丢弃。
- 不要吞掉上游错误；错误类型、HTTP 状态和用户可见信息要保持可诊断。
- 请求身份、渠道 key、用户 token、签名和敏感 header 不得写入日志。
- JSON marshal/unmarshal 调用遵守根目录规则：统一走 `pkg/jsonx` 包装函数，不直接调用 `encoding/json`。

## 验证

- 改协议转换执行相关 roundtrip、stream、provider 测试，例如 `go test ./internal/relay/...`。
- 改某个 provider 时至少执行该 provider 包测试和 relay 公共转换测试。
- 影响 usage 或 billing 时补充 `internal/domain/billing`/`relay/common` 相关测试。
