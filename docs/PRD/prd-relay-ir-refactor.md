# PRD：Relay 层 IR 重构

> 版本：v1
> 日期：2026-07-16
> 前置文档：[prd-relay-sdk-migration.md](./prd-relay-sdk-migration.md)（SDK 迁移，必须先完成并合并）

## 概述

对 new-api 的 relay 层进行 IR（中间表示）重构，解决当前架构的核心痛点：

1. **协议转换与计费深度耦合**：多个 `PostXxxConsumeQuota` 函数按协议分裂（Claude / Audio / Realtime / 通用），usage 提取逻辑无法复用。
2. **协议转换逻辑分散**：handler / adaptor / service 三层各自维护协议转换，难以线性扩展新协议。

移除现有协议转换和计费逻辑，对齐 AxonHub 独立 IR（中间表示）模式，实现 N+M 线性扩展。

> 本 PRD 只覆盖 IR 重构（协议转换 IR 化 + 计费 IR 化）。SDK 迁移、BaseURL bug 修复、adaptor 接口 4 个 Convert 方法改造等内容见 [prd-relay-sdk-migration.md](./prd-relay-sdk-migration.md)。

---

## 前置条件

- [prd-relay-sdk-migration.md](./prd-relay-sdk-migration.md) 已合并到 main
- SDK 迁移合并后的版本已通过完整测试

## 目标

移除现有协议转换和计费逻辑，对齐 AxonHub 独立 IR（中间表示）模式，实现 N+M 线性扩展。

## 范围

### Step 1：协议转换 IR 化

#### 1.1 定义 IR 模型

新建 `relay/ir/` 包，定义统一的中间模型。参考 AxonHub 的 `llm.Request` / `llm.Response`，结合 new-api 项目实际需求定制。

核心结构：

| 结构 | 职责 | 参考 |
|---|---|---|
| `ir.Request` | 统一请求模型（messages / model / tools / parameters） | AxonHub `llm.Request` |
| `ir.Response` | 统一响应模型（content / choices / finish_reason） | AxonHub `llm.Response` |
| `ir.Usage` | 统一 usage 模型（见下方完整字段） | AxonHub `llm.ResponseMeta` |
| `ir.StreamEvent` | 统一流式事件模型 | AxonHub `httpclient.StreamEvent` |
| `ir.ResponseError` | 统一错误模型 | AxonHub `llm.ResponseError` |

**`ir.Usage` 字段定义（审查 P0-3 修正）**：

```
prompt_tokens                         // 输入 token
completion_tokens                     // 输出 token
reasoning_tokens                      // 推理 token（OpenAI o-series）

// Claude 缓存字段（三档，倍率不同，不可合并）
cache_read_tokens                     // 缓存命中（读）
cache_creation_tokens                 // 缓存创建（默认档）
cache_creation_5min_tokens            // 缓存创建（5 分钟档）
cache_creation_1hour_tokens           // 缓存创建（1 小时档）

// Audio / Realtime 字段（审查 P0-1）
audio_input_tokens                    // 音频输入 token
audio_output_tokens                   // 音频输出 token

// 扩展字段（预留，防 IR 不够宽）
extra map[string]any                  // 上游独有字段透传
```

> `cache_creation_tokens`（默认档）= 总缓存创建 token - 5min - 1hour。对应现有代码 `PostClaudeConsumeQuota` 中 `remainingCacheCreationTokens = cacheCreationTokens - cacheCreationTokens5m - cacheCreationTokens1h`。

#### 1.2 建立 Inbound 注册表

Inbound 负责 `客户端协议 ↔ IR`：

| Inbound | 客户端协议 | 对应入口路由 |
|---|---|---|
| `OpenAIChatInbound` | OpenAI Chat Completions | `/v1/chat/completions` |
| `OpenAIResponsesInbound` | OpenAI Responses | `/v1/responses` |
| `ClaudeInbound` | Anthropic Messages | `/v1/messages` |
| `GeminiInbound` | Gemini | `/v1beta/models/*` |

每个 Inbound 实现（审查 P1-1 补充 `AggregateStreamChunks`，P1-2 修正签名）：

```go
// 客户端协议 → IR
TransformRequest(ctx, *httpclient.Request) (*ir.Request, error)

// IR → 客户端协议（非流式）
TransformResponse(ctx, *ir.Response) (*httpclient.Response, error)

// IR → 客户端协议（流式）
TransformStream(ctx, Stream[*ir.Response]) (Stream[*httpclient.StreamEvent], error)

// 错误转换
TransformError(ctx, error) *httpclient.Error

// 流式 chunk 聚合（用于计费 usage 提取）
AggregateStreamChunks(ctx, chunks []*httpclient.StreamEvent) ([]byte, ir.ResponseMeta, error)
```

> `AggregateStreamChunks` 是流式计费的数据来源：流式请求结束后，将所有 chunk 聚合成完整 response 来提取 usage。

#### 1.3 建立 Outbound 注册表

Outbound 负责 `IR ↔ 上游协议`。基于 SDK 迁移 PRD 已有的 SDK params adaptor 改造：

| Outbound | 上游协议 | 底层 SDK |
|---|---|---|
| `OpenAIChatOutbound` | OpenAI Chat Completions | `openai-go` |
| `OpenAIResponsesOutbound` | OpenAI Responses | `openai-go` |
| `ClaudeOutbound` | Anthropic Messages | `anthropic-sdk-go` |
| `GeminiOutbound` | Gemini | 待定 |
| `AWSOutbound` | AWS Bedrock | `aws-sdk-go-v2` / `anthropic-sdk-go` |
| 其他渠道 Outbound | 各厂商协议 | SDK 或自定义 HTTP |

每个 Outbound 实现（审查 P1-2 修正 `TransformStream` 签名）：

```go
// APIFormat 标识
APIFormat() ir.APIFormat

// IR → 上游请求
TransformRequest(ctx, *ir.Request) (*httpclient.Request, error)

// 上游响应 → IR（非流式）
TransformResponse(ctx, *httpclient.Response) (*ir.Response, error)

// 上游响应 → IR（流式）—— 注意：比 Inbound 多了 req 参数
TransformStream(ctx, req *httpclient.Request, Stream[*httpclient.StreamEvent]) (Stream[*ir.Response], error)

// 错误转换
TransformError(ctx, *httpclient.Error) *ir.ResponseError

// 流式 chunk 聚合（用于计费 usage 提取）
AggregateStreamChunks(ctx, req *httpclient.Request, chunks []*httpclient.StreamEvent) ([]byte, ir.ResponseMeta, error)
```

> Outbound 的 `TransformStream` 和 `AggregateStreamChunks` 比 Inbound 多了 `req *httpclient.Request` 参数，用于复合 transformer 路由（如 DeepSeek 用 `RequestType` 区分 chat-completion 和 completion 子 transformer）。

#### 1.4 Pipeline 串联

新建 `relay/pipeline/` 包，参考 AxonHub 的 pipeline 架构：

```
客户端 HTTP 请求
   ↓ Inbound.TransformRequest
ir.Request（统一中间模型）
   ↓ [中间件链：日志 / 限速 / 改写 / 审计…]
   ↓ Outbound.TransformRequest
上游 HTTP 请求（通过 SDK client 发送）
   ↓ ====== 调用上游 ======
上游 HTTP 响应（流式 or 非流式）
   ↓ Outbound.TransformResponse / TransformStream
ir.Response（统一中间模型）
   ↓ [中间件链反向]
   ↓ Inbound.TransformResponse / TransformStream
客户端 HTTP 响应
```

#### 1.5 计费临时适配层（审查 P2-3 补充数据流）

Step 1 完成时，计费逻辑暂时不改。通过临时适配层把 `ir.Usage` 转换回现有的 `dto.Usage`，继续调用旧的计费函数。

**数据流与时序**：

```
// 非流式
上游响应 → Outbound.TransformResponse → ir.Response.Usage → 适配层 ir.Usage→dto.Usage → PostConsumeQuota

// 流式
上游流 → Outbound.TransformStream → ir.Response 流
   → Inbound.TransformStream → 客户端 SSE
   → 流结束后 Inbound.AggregateStreamChunks → ir.ResponseMeta.Usage
   → 适配层 ir.Usage→dto.Usage → PostConsumeQuota
```

#### 1.6 删除旧代码

Step 1 验证通过后，删除：
- 旧的 8 个 Convert 方法
- 旧 handler（`claude_handler.go` / `gemini_handler.go` / `responses_handler.go` / `compatible_handler.go` 等）
- `relay/openai_wire_auto_convert.go` 及相关文件
- `service/` 层的协议转换函数（`ClaudeToOpenAIRequest` / `GeminiToOpenAIRequest` 等）
- `relay/common/` 中的 wire 转换辅助函数

> **删除策略（审查 P2-1）**：不直接物理删除。先标记 deprecated，用 build tag 或 feature flag 控制新旧路径切换，跑完完整回归后物理删除。new-api 有线上用户，硬删风险大。

#### WebSocket / Realtime 路径处理（审查 P0-1）

**决策**：Step 1 将 Realtime 纳入 IR pipeline。

`relay/websocket.go` 的 `WssHelper` 目前是独立入口，不走 adaptor Convert 方法，usage 类型是 `dto.RealtimeUsage`（含 `InputTokenDetails.TextTokens` / `AudioTokens`）。

`ir.Usage` 已增加 `audio_input_tokens` / `audio_output_tokens` 字段覆盖 Realtime 的 audio token。Step 1 需要新增 `RealtimeInbound`（WebSocket 入站适配器），将 Realtime 纳入 IR pipeline。

如果 Realtime IR 化的技术难度超出预期，回退方案是：Realtime 保留旁路（不走 IR pipeline），但必须在"不在范围内"章节显式列出，且 Step 2 的计费统一保留 `PostWssConsumeQuota`。

#### Step 1 验证标准

- IR 包自己的单元测试 + roundtrip 测试（`Claude→IR→Claude` 保真）
- 每个渠道迁移后通过该渠道全套测试
- 流式输出正确（chunk 顺序 / finish_reason / usage / 连接关闭）
- 计费结果与迁移前一致（通过临时适配层保证）
- Realtime 计费回归通过（如果纳入 IR）

### Step 2：计费 IR 化

#### 2.1 统一 usage 模型

将 `ir.Usage`（含 Step 1 定义的所有字段）作为唯一的 usage 数据结构，所有 Outbound 都把上游 usage 翻成 `ir.Usage`。

#### 2.2 合并计费函数

将现有计费函数合并为一个基于 `ir.Usage` 的统一函数：

```go
func PostConsumeQuota(relayInfo *RelayInfo, usage *ir.Usage, ...) error
```

需合并的函数（含 Realtime，因为 Step 1 已将 Realtime 纳入 IR）：

| 旧函数 | 对应的 `ir.Usage` 字段 |
|---|---|
| `PostConsumeQuota`（通用） | `prompt_tokens` / `completion_tokens` / `reasoning_tokens` |
| `PostClaudeConsumeQuota` | + `cache_read_tokens` / `cache_creation_tokens` / `cache_creation_5min_tokens` / `cache_creation_1hour_tokens` |
| `PostAudioConsumeQuota` | + `audio_input_tokens` / `audio_output_tokens` |
| `PostWssConsumeQuota` | + `audio_input_tokens` / `audio_output_tokens` |

内部根据 `relayInfo.ChannelType` / `relayInfo.RelayMode` 选择对应的倍率计算逻辑（这部分差异是合理的，因为不同渠道的计费倍率确实不同），但 usage 提取本身不再按协议分裂。

#### 2.3 删除旧计费函数

- `PostClaudeConsumeQuota`
- `PostAudioConsumeQuota`
- `PostWssConsumeQuota`
- 旧的通用 `PostConsumeQuota`（被新统一函数替代）

> 同样走 deprecated 过渡，不直接物理删除。

#### Step 2 验证标准

- 计费回归测试：所有渠道的计费结果与 Step 1 一致
- `ir.Usage` 字段覆盖率：所有现有 usage 字段都能正确映射
- Claude 三档缓存计费回归通过（5m / 1h 倍率不同）
- Realtime audio token 计费回归通过
- `go test ./...` 全绿

## 风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| IR 设计不够"宽"，丢失上游独有能力 | 部分字段无法透传 | 参考 AxonHub IR 字段设计，预留 `extra map[string]any` 扩展字段 |
| Claude→IR→Claude roundtrip 不保真 | 客户端收到失真的响应 | roundtrip 测试覆盖 |
| 流式转换 chunk 顺序错乱 | 客户端接收到的 SSE 流异常 | 流式测试覆盖 |
| 计费统一后边缘场景遗漏 | 计费偏差 | 逐渠道对比新旧计费结果 |
| Claude 三档缓存计费遗漏 | Claude 计费回归失败 | `ir.Usage` 已覆盖三档字段 |
| Realtime IR 化技术难度超预期 | Realtime 功能不可用 | 回退方案：Realtime 保留旁路 |
| 删除旧代码后发现问题需要回退 | 代码丢失 | deprecated 过渡 + 分支版本控制 |

## 完成标准

- IR 模型定义完整，覆盖所有现有协议的能力（含 Claude 三档缓存、Realtime audio token）
- Inbound + Outbound 注册表建立，所有渠道已迁移（含 Realtime Inbound）
- Pipeline 串联完成，中间件在 IR 层统一介入
- 计费统一到 `ir.Usage`，旧计费函数标记 deprecated 后物理删除
- 旧代码标记 deprecated 后物理删除
- 全部测试通过
- 合并回 main，发布版本

---

## 分支与版本控制

```
main（已合并 SDK 迁移 PRD）
  └── IR 重构分支 ← 从 SDK 迁移合并后的 main 切出
       ├── Step 1：协议转换 IR 化
       │    ├── IR 模型定义（含 Claude 三档缓存 + Realtime audio 字段）
       │    ├── Inbound + Outbound 注册表
       │    ├── Pipeline 串联
       │    ├── 计费临时适配层
       │    ├── 旧协议转换代码标记 deprecated
       │    └── 阶段测试
       │
       ├── Step 2：计费 IR 化
       │    ├── 统一 usage 模型
       │    ├── 合并计费函数
       │    ├── 旧计费函数标记 deprecated
       │    └── 阶段测试
       │
       ├── 物理删除所有 deprecated 代码
       └── 合并回 main → 发布版本
```

**衔接规则**：
- 本 PRD 从 SDK 迁移 PRD 合并后的 main 切出
- SDK 迁移 PRD 对 adaptor 接口的改造在本 PRD 中会被 IR 架构推倒重来，这是预期行为
- 合并后发布一个可用版本，保证版本迭代过程中始终有可发布状态

---

## 不在范围内

以下内容明确不在本 PRD 范围内：

- 前端 UI 改动（Playground 走独立路由，不受影响）
- 路由注册层改动（`router/relay_router.go` 的对外路径不变）
- 数据库 schema 变更
- 新增渠道类型
- 模型倍率配置逻辑变更（倍率差异是合理的，不变）
- 审计日志逻辑变更
- 模型倍率本身的数值调整

以下内容由 [prd-relay-sdk-migration.md](./prd-relay-sdk-migration.md) 处理，本 PRD 不重复：

- adaptor 接口 4 个核心 Convert 方法迁移到 SDK params
- `GetRequestURL` 语义变更为返回路径后缀
- BaseURL 双重版本前缀 bug 修复
- Cloudflare Gateway 死代码删除
- SDK client 初始化与 middleware 注入

> 错误重试策略（`RelayErrorHandler` / `ResetStatusCode` / `NewEmptyUsageRetryError`）的重试策略本身不变，本 PRD 只适配 IR 错误模型。
