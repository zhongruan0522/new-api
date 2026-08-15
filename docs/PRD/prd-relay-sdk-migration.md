# PRD：Relay 层 SDK 迁移

> 版本：v1
> 日期：2026-07-16
> 关联文档：[prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md)（IR 重构，本 PRD 完成后启动）

## 概述

对 new-api 的 relay 层进行 SDK 迁移，解决当前架构的核心痛点：

1. **万物基于 OpenAI HTTP 手动构造**：所有渠道适配都围绕 `GeneralOpenAIRequest` 展开，协议转换逻辑分散在 handler / adaptor / service 三层，8 个 Convert 方法持续膨胀。
2. **BaseURL 双重版本前缀 bug**：用户填写带版本前缀的 BaseURL（如 `https://ark.cn-beijing.volces.com/api/v3`）后，系统拼出的上游 URL 变成 `.../api/v3/v1/chat/completions`，导致请求失败。

通过本分支将渠道适配从"万物基于 OpenAI HTTP 手动构造"迁移到"优先使用渠道官方 SDK"，降低维护成本，提升协议保真度。

> 本 PRD 只覆盖 SDK 迁移。协议转换 IR 化、计费统一、旧代码删除等内容见 [prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md)。

---

## 目标

将渠道适配从"万物基于 OpenAI HTTP 手动构造"迁移到"优先使用渠道官方 SDK"，降低维护成本，提升协议保真度。

## 范围

### 1. adaptor 接口改造

**现状**：adaptor 接口有 8 个 Convert 方法，返回 new-api 自定义的 DTO。

**目标**：本 PRD 只改以下 4 个方法，返回官方 SDK 的 params 类型：

| 方法 | 改造后返回类型 |
|---|---|
| `ConvertOpenAIRequest` | `openai.ChatCompletionNewParams` |
| `ConvertClaudeRequest` | `anthropic.MessageNewParams` |
| `ConvertGeminiRequest` | 待定（Gemini SDK 选型后确定） |
| `ConvertOpenAIResponsesRequest` | `responses.ResponseNewParams` |

以下 4 个方法在本 PRD 中**不改**，保持返回旧 DTO 类型（审查 P1-4）：

- `ConvertRerankRequest` — 返回 `dto.RerankRequest`
- `ConvertEmbeddingRequest` — 返回 `dto.EmbeddingRequest`
- `ConvertAudioRequest` — 返回 `io.Reader`（签名特殊，保留）
- `ConvertImageRequest` — 返回 `dto.ImageRequest`

> **中间态说明**：本 PRD 完成后 adaptor 接口处于混合态（4 个方法返回 SDK params，4 个返回旧 DTO）。这是预期行为，IR 重构 PRD 会将整个接口推倒重来。

### 2. `GetRequestURL` 接口语义变更（审查 P1-3）

**现状**：`GetRequestURL` 返回完整的上游 URL string，由各 adaptor 自行拼接 `ChannelBaseUrl + RequestURLPath`。

**改造方案**：`GetRequestURL` 保留在接口中，但语义从"返回完整 URL"变为"返回路径后缀"（如 `/chat/completions`、`/messages`）。BaseURL 由 SDK client 初始化时通过 `option.WithBaseURL` 注入。

| 渠道 | `GetRequestURL` 返回值 | SDK BaseURL |
|---|---|---|
| OpenAI | `/chat/completions` 或 `/responses` | 用户填写的 ChannelBaseUrl |
| Anthropic | SDK 内部管理（`v1/messages`） | 用户填写的 ChannelBaseUrl（不含 `/v1`） |
| AWS ApiKey | `/model/{id}/converse` | Bedrock region endpoint |
| AWS AKSK | `""`（走 SDK，不用 URL） | N/A |
| ByteDance | `/chat/completions` | `https://ark.cn-beijing.volces.com/api/v3` |

> 各 adaptor 如果需要完整 URL（如 Custom 渠道的模板替换），可在 `GetRequestURL` 内部自行拼接完整 URL 返回。调用方根据返回值是否以 `http://` / `https://` 开头判断是完整 URL 还是路径后缀。

### 3. SDK client 初始化与 middleware 注入

使用 SDK 的完整 client，通过以下 option 注入 new-api 现有的 HTTP 层能力：

| 现有能力 | 注入方式 | 对应 SDK option |
|---|---|---|
| 自定义代理（proxyURL） | 自定义 `*http.Client`（Transport 注入 proxy） | `option.WithHTTPClient` |
| Header 覆盖（渠道级 / 运行时） | 请求前修改 header | `option.WithMiddleware` |
| 请求超时 / 重试 | SDK 内置 + middleware 覆盖 | `option.WithRequestTimeout` |
| 流式 usage 提取（计费） | 响应后解析最后一个 chunk 的 usage | SDK 流式聚合器 |
| 请求体清理（BodyStorageCleanup） | 请求前清理 | `option.WithMiddleware` |
| 请求解压（Decompress） | Transport 层处理 | `option.WithHTTPClient` |

**两个 SDK 的 middleware 签名一致**：

```go
type Middleware = func(req *http.Request, next MiddlewareNext) (*http.Response, error)
```

### 4. Chat Completions + Responses 双 API 迁移

`openai-go` v3 同时支持：
- `client.Chat.Completions.New` → `/chat/completions`
- `client.Responses.New` → `/responses`

两个 API 都迁移到 SDK。

### 5. BaseURL 双重版本前缀 bug 修复

**根因**：`info.RequestURLPath` 来自 `c.Request.URL.String()`，客户端发 `/v1/chat/completions` 时，`/v1` 被透传到上游 URL。当用户 BaseURL 填 `https://ark.cn-beijing.volces.com/api/v3` 时，拼出 `.../api/v3/v1/chat/completions`。

**修复方案**：
- BaseURL 保持用户原样输入（含 `/v1` 或 `/api/v3`）
- SDK 的 `option.WithBaseURL` 只填用户输入的 BaseURL
- 路径后缀（`/chat/completions`、`/messages`）由 `GetRequestURL` 返回
- 删除 `GetFullRequestURL` 中 Cloudflare Gateway 残留死代码（渠道 type 已移除）

**涉及文件**：
- `relay/common/relay_utils.go` — 删除 Cloudflare 特殊处理分支，删除 `GetFullRequestURL`（功能由 SDK BaseURL + `GetRequestURL` 路径后缀替代）
- `relay/channel/openai/adaptor.go:181` — 删除 `fmt.Sprintf("%s/v1/chat/completions", info.ChannelBaseUrl)` 硬编码
- 各 adaptor 的 `GetRequestURL` — 改为返回路径后缀

### 6. 不动的部分

以下代码在本 PRD 中**不修改**，留到 IR 重构 PRD 处理：

- `relay/openai_wire_auto_convert.go`（chat↔responses wire 转换）
- `relay/compatible_handler.go`（协议兼容处理）
- `service/` 层的协议转换函数（`ClaudeToOpenAIRequest` 等）
- 计费逻辑（`PostClaudeConsumeQuota` 等）
- Audio / Rerank / Embedding / Image 的 Convert 方法（审查 P1-4）
- **PassThrough Body 模式**（审查 P0-2）：`info.ChannelSetting.PassThroughBodyEnabled` 为 true 时，handler 直接从 `BodyStorage` 取原始 body 发送，**不调用 Convert 方法**，因此 adaptor 返回类型变更对 PassThrough 路径无影响
- **WebSocket / Realtime 路径**（审查 P0-1）：`relay/websocket.go` 的 `WssHelper` 是独立入口，不走 Convert 方法，直接调用 `adaptor.DoRequest` + `adaptor.DoResponse`，本 PRD 不动
- 错误处理和重试逻辑（`RelayErrorHandler` / `ResetStatusCode` / `NewEmptyUsageRetryError`）：不改变重试策略本身，IR 重构 PRD 适配 IR 模式

## 渠道优先级

原则：**优先使用渠道官方 SDK，SDK 支持不够则退回自定义 HTTP**。

建议的迁移顺序（按风险和收益排序）：

1. **OpenAI**（`openai-go`）：最核心，SDK 最成熟，验证 middleware 机制
2. **Anthropic 原生**（`anthropic-sdk-go`）：验证 Claude 流式 usage 提取
3. **AWS ApiKey 模式**：见下方 AWS Bedrock 专项说明
4. **OpenRouter / DeepSeek**（`openai-go` + `WithBaseURL`）：验证国内/第三方兼容度
5. **Gemini**（待定）：确认 `google.golang.org/genai` 是否满足需求
6. **其他国内厂商**（Moonshot / ByteDance / Xiaomi / MiniMax / SiliconFlow / ZhipuV4）：逐个验证 OpenAI 兼容度，不兼容的退回自定义 HTTP

### AWS Bedrock ApiKey 模式说明（审查 P2-4）

AWS Bedrock ApiKey 模式不是简单的"覆盖 baseURL"。它需要走 Bedrock 的 invoke endpoint + SigV4 签名。

**方案**：
- 如果 `anthropic-sdk-go` 支持 Bedrock runtime 集成（通过 AWS credentials chain + Bedrock baseURL），优先使用 SDK 原生支持
- 否则通过自定义 middleware 注入 SigV4 签名 + 覆盖 baseURL
- AKSK 模式保持现有 `aws-sdk-go-v2` 的 `bedrockruntime.Client`，不改动

## 每个渠道迁移完成的 Definition of Done（审查 P2-2）

- 该渠道 `relay/channel/<provider>/` 测试包全绿
- 手动验证流式 + 非流式各一次（至少确认 usage 字段正确）
- BaseURL 带 `/api/v3` 等版本前缀的 URL 拼接正确
- PassThrough 模式不受影响（如果该渠道支持 PassThrough）

## 验证标准

- `go test ./...` 全绿
- 每个迁移完成的渠道通过该渠道的全套测试
- 流式 usage 提取正确（prompt_tokens / completion_tokens / cache 相关字段）
- BaseURL bug 修复验证：填写带 `/api/v3` 的 BaseURL 后拼出的上游 URL 正确
- middleware 注入验证：proxyURL / header override / timeout / 请求体清理 均生效
- PassThrough 模式行为不变

## 风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| SDK 流式聚合与现有手动解析行为不一致 | usage 计费偏差 | 每个渠道迁移后对比新旧 usage 提取结果 |
| 国内渠道 OpenAI 兼容度不足 | 请求失败或字段丢失 | 逐个验证，不兼容的退回自定义 HTTP |
| anthropic-sdk-go 内部 path 硬编码 `v1/messages` | Bedrock baseURL 拼接问题 | 验证 Bedrock 兼容性，必要时用 `WithBaseURL` + middleware 重写 path |
| adaptor 接口变更影响面大 | 编译错误密集 | 分渠道迁移，每迁一个就修编译 + 跑测试 |
| Audio/Rerank/Embedding/Image 的 Convert 方法保留旧签名 | 接口混合态 | 预期行为，IR 重构 PRD 推倒重来 |

## 完成标准

- 4 个核心 Convert 方法已迁移到返回 SDK params 类型
- 4 个非核心 Convert 方法保持旧 DTO（预期中间态）
- `GetRequestURL` 语义变更为返回路径后缀
- BaseURL 双重版本前缀 bug 已修复
- Cloudflare Gateway 死代码已删除
- 不支持 SDK 的渠道退回自定义 HTTP 且功能完整
- 全部测试通过
- 合并回 main，发布版本

---

## 不在范围内

以下内容明确不在本 PRD 范围内，由 [prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md) 处理：

- IR 模型定义（`relay/ir/` 包）
- Inbound / Outbound 注册表
- Pipeline 串联
- 协议转换 IR 化（删除旧 8 个 Convert 方法、旧 handler、wire 转换）
- 计费统一（`PostClaudeConsumeQuota` / `PostAudioConsumeQuota` / `PostWssConsumeQuota` 合并）
- Realtime 纳入 IR pipeline
- 旧代码 deprecated 标记与物理删除

以下内容在两个 PRD 中均不在范围内：

- 前端 UI 改动（Playground 走独立路由，不受影响）
- 路由注册层改动（`router/relay_router.go` 的对外路径不变）
- 数据库 schema 变更
- 新增渠道类型
- 模型倍率配置逻辑变更（倍率差异是合理的，不变）
- 审计日志逻辑变更
- 错误重试策略变更（`RelayErrorHandler` / `ResetStatusCode` / `NewEmptyUsageRetryError` 的重试策略不变，IR 重构 PRD 适配 IR 错误模型）
- 模型倍率本身的数值调整
