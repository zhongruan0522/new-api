# internal/relay/AGENTS.md

`internal/relay/` 是 AI API 中继和供应商适配核心，改动风险高。

## 包结构

- `relay.go`：对外门面（facade），仅 re-export 各子包入口（`GetAdaptor`、各 `*Helper`、`OpenAIWireHelper` 等），保持 `internal/relay` 历史 import 路径稳定；不要在门面里写业务逻辑。
- `core/`：adaptor 调度（`GetAdaptor`）、websocket 中继（`WssHelper`）、存储资产签名 URL（`VerifyStoredAssetSignature`、`BuildStoredImageURL` 等）。
- `wire/`：OpenAI wire 协议族。`auto_convert.go` 是 Chat ↔ Responses 自动转换调度；`wire/convert/` 是请求/响应/usage 转换与 tool 代理上下文（原 `relay/common/openai_wire_*`），其中 `Converter` 是 chat ↔ responses 双向转换的单点入口；`wire/stream/` 是流式转换器与 SSE 改写 writer。
- `handler/`：各模态中继 handler（chat/claude/gemini/audio/image/embedding/rerank/responses/stored media）。
- `channel/`：供应商适配，按 `<provider>/` 一级子包。
- `common/`：RelayInfo、`BillingSettler` 消费侧接口（`billing.go`，实现在 domain/billing.BillingSession）、参数覆写、reasoning effort、请求转换链等中继横切状态；OpenAI wire 协议转换已迁 `wire/`，勿再往回加。
- `helper/`：Claude/Gemini ↔ OpenAI 协议转换（阶段 5.3 自原 `service/` 迁入）、relay 错误包装、响应透传工具。
- `common_handler/`、`reasonmap/`、`constant/`：原位保留。

依赖方向（无环）：`relay(门面) → {core, wire, handler}`（门面签名另引用 `channel.Adaptor` 与 `*relaycommon.RelayInfo` 类型）；`wire → {handler, common, helper, wire/convert, wire/stream}`；`handler → {core, common, helper, constant}`；`common → wire/convert`（`RelayInfo.OpenAIResponsesToolContext`）；`wire/stream → wire/convert`；`wire/convert` 不依赖任何 relay 内部包（依赖 `domain/shared`、`internal/common` 的 `Interface2String` 工具与 `pkg/jsonx`）。

## 规则

- 保持协议边界清晰：OpenAI wire、Responses、Chat Completions、Claude、Gemini、AWS 等转换不要混写。OpenAI wire 协议族的转换一律放 `wire/convert/`、`wire/stream/`，不要放回 `common/` 或 `helper/`。
- chat ↔ responses 转换必须经单点入口进入：请求与非流式响应走 `convert.NewConverter(...)` 的方法（会话承载方向与 tool 代理上下文），流式逐帧走 `stream.NewChatToResponsesStreamConverter` / `NewResponsesToChatStreamConverter` 两个构造器；不要绕过它们直调 `wire/convert` 内部具体函数或自建转换器。
- 计费边界：usage → quota 的计算与落账只经 `domain/billing.CalculateUsage(ctx, relayInfo, rawUsage, extraContent...)` + `ApplyQuota(ctx, relayInfo, settlement)`（通用文本路径）或既有 `PostClaudeConsumeQuota` / `PostAudioConsumeQuota` / Pre&PostWss 等领域侧入口完成；阶段 2 起这些入口的 quota 一律由归一化 `BillingUsage` 计算（`billing.CalculateNormalizedQuota`，PRD 第 3 章公式），不再从聚合字段反推普通输入；relay 各层不得 import store 的 token/user/channel 包读写配额。
- usage 语义来源必须显式标识：各渠道响应解析点（`claude.HandleClaudeResponseData`/`HandleStreamResponseData`、`openai` 各 Handler、`gemini` 各 Handler、AWS Nova、rerank handler 等）在解析 usage 时写入 `relayInfo.UsageSource`（`relay/constant` 的 `UsageSource*`，覆盖 Claude / OpenAI Chat / OpenAI Responses / Gemini 或对应兼容渠道来源），计费归一化（`billing.BuildBillingUsage`）据此选择语义规则；禁止用 `FinalRequestRelayFormat` 等请求侧格式反推。Gemini 解析点还须把原始 `usageMetadata` 暂存到 `relayInfo.UsageGeminiMetadata`（toolUse 拆分只审计，不进计价输入）。新增渠道解析 usage 时必须同步打标：携带真实 token 用量但来源未打标的请求在计费侧显式失败（不可重试、预扣退还，cause=normalization_failed），不再回退旧公式。
- 响应侧实际生效的 `service_tier` / Gemini `usageMetadata.serviceTier` 解析后写入 `relayInfo.SetEffectiveServiceTier`，由计费快照统一落到 `Other.service_tier`；不要把请求侧 tier 写成生效 tier，也不要把 tier 放入 `billing_details`。
- 本地估算/伪 token 用量（`ResponseText2Usage`、TTS 字符数与无 usage 事件的 TTS 流、STT 估算、Realtime 本地计数计费分支、按张数充数、Gemini imagen 固定 258 等）必须设置 `ContextKeyLocalCountTokens`，这类入口不生成 `billing_details`；未打 UsageSource 的伪 usage 按聚合口径构造归一化用量计费。
- 计费可观测三类原因（PRD 阶段 2）必须可区分：计费配置缺失（模型倍率/价格未配置在请求准入口 `ModelPriceHelper` 报错，分段档位匹配失败在计费侧记 cause=billing_config_missing）、归一化失败（来源未打标或上游数据非法，cause=normalization_failed，显式报错并退还预扣）、上游无 usage（rawUsage 为 nil 走估算兜底或本地计数打标）。迁移期保留旧公式影子对拍（`billing/billing_shadow.go`）：quota 差异输出告警并按已知类别标注，禁止吸收差异。
- 流式输出必须保护 chunk 顺序、错误事件、finish reason、usage 和连接关闭行为。
- 计费、预扣、补扣、缓存倍率、音频/图片/视频/embedding/rerank 的 usage 不能随意近似。
- 供应商适配放在 `internal/relay/channel/<provider>/`，跨模态共享转换放 `internal/relay/helper/`。
- 新增 channel 时确认供应商是否支持 `stream_options` 等流式选项，并同步相关 capability 判断。
- 请求 DTO 中需要重新 marshal 给上游的可选标量字段，使用 `*int`、`*uint`、`*float64`、`*bool` 等指针类型加 `omitempty`，避免显式零值被丢弃。
- 不要吞掉上游错误；错误类型、HTTP 状态和用户可见信息要保持可诊断。
- 请求身份、渠道 key、用户 token、签名和敏感 header 不得写入日志。
- JSON marshal/unmarshal 调用遵守根目录规则：统一走 `pkg/jsonx` 包装函数，不直接调用 `encoding/json`。

## 验证

- 改协议转换执行相关 roundtrip、stream、provider 测试，例如 `go test ./internal/relay/...`。
- 改某个 provider 时至少执行该 provider 包测试和 relay 公共转换测试。
- 影响 usage 或 billing 时补充 `internal/domain/billing`（含 `usage_test.go` 的 CalculateUsage 用例）相关测试。
