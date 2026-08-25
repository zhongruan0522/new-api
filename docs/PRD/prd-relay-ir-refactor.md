# PRD：Relay 协议转换层 IR 重构（以 Chat 为中心）

> 版本：v2（整文重写：v1 的「点对点」定性有误，实测现状已是隐式 Hub；阶段划分按协议对拆分重排）
> 日期：2026-08-25
> 状态：待评审
> 关联文档：`prd-relay-sdk-migration.md`（SDK 迁移，已延期搁置，见 §1.4）

## 1. 背景

### 1.1 现状不是「点对点」，而是「隐式 Hub + 每渠道手工拼装」

重构动机常被描述为「N×N 点对点转换难维护」。实测代码后这个定性需要修正：**OpenAI Chat 早已是事实上的中间协议**，只是没有被形式化，也没有纪律约束。

回程转换全部走 `上游协议 → OpenAI Chat → 下游协议` 两跳：


| 转换路径                           | 实现位置                                       | 实际做法                                                                 |
| ------------------------------ | ------------------------------------------ | -------------------------------------------------------------------- |
| Claude 上游 → Gemini 客户端         | `channel/claude/relay_claude.go:1007,1017` | `StreamResponseClaude2OpenAI` 再 `helper.StreamResponseOpenAI2Gemini` |
| Claude 上游 → Responses 客户端（流式）  | `channel/claude/relay_claude.go:993,1089`  | `StreamResponseClaude2OpenAI` 再 `NewChatToResponsesStreamConverter`  |
| Claude 上游 → Responses 客户端（非流式） | `channel/claude/relay_claude.go:1205-1211` | `ResponseClaude2OpenAI` 再 `convert.NewConverter(Chat, Responses)`    |
| Responses 客户端 → Claude 上游（去程）  | `channel/claude/adaptor.go:94-113`         | `ConvertResponsesToChatRequest` 再 `ConvertOpenAIRequest`             |
| Gemini/Claude 客户端 → OpenAI 上游  | `channel/openai/adaptor.go:40,62`          | `GeminiToOpenAIRequest` / `ClaudeToOpenAIRequest`                    |


**结论：本次重构不是「引入 Hub」，而是「把既有隐式 Hub 显式化并收口」。** 这个定性差异有两个直接后果：

1. 迁移风险显著低于从零设计——大部分转换算子已存在且经过线上验证，工作性质偏「搬家 + 收口」。
2. IR 选型被现状锁定为 **OpenAI Chat 超集**。约八成转换资产以 Chat 为轴（`helper/convert.go` 全部函数、各渠道 `ConvertOpenAIRequest`、`ResponseClaude2OpenAI` 系列），换成别的 IR 等于全部重写。



### 1.2 隐式 Hub 造成的六个实际缺陷

以下每条都有代码位置，不是推测。

**D1：转换矩阵有洞，Responses 客户端打到三类渠道直接报错**

`channel/gemini/adaptor.go:228`、`channel/vertex/adaptor.go:299`、`channel/aws/adaptor.go:142` 的 `ConvertOpenAIResponsesRequest` 全部是 `// TODO implement me` + `return nil, errors.New("not implemented")`。

底层零件其实齐备（Responses↔Chat 有 `wire/convert`，Chat↔Gemini 有 `helper` 系列），缺的只是把两跳接起来。之所以一直没接，见 D2。

**D2：每渠道手写四路格式分发，新增组合要改所有渠道**

`channel/claude/relay_claude.go:962-1028`（流式）和 `:1196-1230`（非流式）各有一个按 `info.RelayFormat` 的四路 switch（Claude/OpenAI/Responses/Gemini），`channel/openai/helper.go:54-199` 同构。

新增一个协议组合，要在每个渠道的去程 `Convert*Request`、回程流式 switch、回程非流式 switch 三处各加一遍。以 Claude 渠道支持 Responses 为例，除 switch 分支外还额外产生了 `writeClaudeChatChunkAsResponsesEvent`、`ensureClaudeResponsesStreamConverter`、`writeClaudeResponsesFinalEvent` 约 130 行胶水（`relay_claude.go:1087-1147`）。乘以 17 个渠道目录，这就是维护成本的来源。

**D3：内建工具被静默丢弃**

`wire/convert/openai_wire_convert_request_tools.go:338-345`：Responses→Chat 转换时 `web_search`、`web_search_preview`、`image_generation` 三类工具直接 `return nil, nil`。注释写明是「Chat Completions 上游无法识别，直接丢弃，不返回错误，避免阻断请求」。

后果是客户端声明了搜索工具、期待按调用计费，实际上游没执行、也没有任何错误或标记告知客户端能力已降级。

**D4：计费语义标记只在一处设置，复用渠道漏设（存量 bug）**

`FinalRequestRelayFormat` 字段定义在 `relay/common/relay_info.go:160`，注释自己写着「TODO: 当前仅设置了Claude」。全仓库只有 `channel/claude/adaptor.go:121` 一处赋值，消费方是 `domain/billing/usage.go:108` 的 `isClaudeUsageSemantic`。

该标记决定缓存 token 的计费算法：Anthropic 的 `input_tokens` 不含缓存 token，OpenAI 的 `prompt_tokens` 含缓存 token，所以基础 token 要不要减去 `cachedTokens` 取决于它。

而 AWS 渠道复用 Claude 的处理函数（`channel/aws/relay_aws.go:222,235,254,266,279` 调用 `claude.ClaudeResponseInfo`、`claude.HandleClaudeResponseData`、`claude.HandleStreamResponseData`、`claude.HandleStreamFinalResponse`）却从不设置该标记，导致 AWS Bedrock 上的 Claude 模型缓存 token 按 OpenAI 语义计算。

**D5：错误出口漏了 Gemini，底子铺好却没接上**

`domain/shared/error.go:32` 定义了 `ErrorTypeGeminiError = "gemini_error"`，`domain/shared/gemini.go:562` 定义了 Gemini 标准错误结构 `GeminiErrorResponse{Code, Message, Status}`。但：

- `ErrorTypeGeminiError` 全仓库零使用
- `NookMuxError` 只有 `ToOpenAIError()`（`error.go:197`）和 `ToClaudeError()`（`error.go:230`），没有 `ToGeminiError()`
- 出口分发（`httpapi/controller/relay/relay.go:98-110`）只有三路：Realtime、Claude、default

所以 Gemini 客户端报错时走 default 分支，收到 OpenAI 格式错误体。Gemini SDK 期待 `{"error":{"code":429,"message":"...","status":"RESOURCE_EXHAUSTED"}}`，实际拿到 `{"error":{"message":"...","type":"...","code":"..."}}`——`status` 缺失，`code` 从数字变字符串。

附带两个问题：同一套出口分发在 `httpapi/middleware/performance.go:23-31` 又写了一遍；`ToClaudeError()` 把 OpenAI 的 `code` 用 `fmt.Sprintf("%v", ...)` 塞进 Anthropic 的 `type` 字段，而后者是固定枚举（`invalid_request_error`、`rate_limit_error` 等），塞进去的值不在枚举内。

**D6：架构张力已经产生内联补丁**

`channel/openai/adaptor.go:376-379` 的注释：MiniMax 音色解析函数内联在 openai 包里，原因是「避免 relay/channel/minimax 与 relay/channel/openai 之间的循环依赖」。这类补丁会随渠道数量增长。

### 1.3 「渠道魔改协议」的调查结论

调查了 17 个渠道目录，**没有任何渠道发明自己的 Chat/Claude/Responses 语义**。所有偏差是四类线路方言，出站边不需要为它们建协议变体：


| 类别         | 实例                                                                                                                                                                                                                                     | 位置                                                                     |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| URL 路径     | Azure `deployments/{model}?api-version=`（含 `AzureNoRemoveDotTime` 按渠道创建时间的历史兼容）；DeepSeek `/anthropic/v1/messages`；Bytedance `/api/v3`→`/api/compatible`；Moonshot/xiaomi/ollama/zhipu 走 `ChannelSpecialBases` 特殊基址表；zhipu 仅 `/v1`→`/v4` | `channel/openai/adaptor.go:126-195`、`channel/deepseek/adaptor.go:55` 等 |
| 认证头        | OpenRouter 的 Anthropic 端点用 Bearer 而非 `x-api-key`                                                                                                                                                                                       | `channel/openai/adaptor.go:220`                                        |
| Body 扩展字段  | 仅 OpenRouter 的 `provider` 路由对象，非 OpenRouter 渠道必须剥离                                                                                                                                                                                     | `relay/common/openrouter_routing.go`                                   |
| Usage 字段变体 | Moonshot 的 Anthropic 端点用 `cached_tokens` 而非 `cache_read_input_tokens`；Llama 系当前未做兼容                                                                                                                                                    | 参考 axonhub `llm/transformer/anthropic/usage.go` 同样特判                   |


其中 URL 与认证属于渠道适配层，与协议转换正交，本 PRD 不动（见 §3.2）。

### 1.4 与 SDK 迁移 PRD 的关系

`prd-relay-sdk-migration.md` 已延期搁置。两份 PRD 存在方向冲突，需要明确顺序：

SDK 迁移计划把 4 个 `Convert*Request` 方法改为返回官方 SDK params 类型（`openai.ChatCompletionNewParams`、`anthropic.MessageNewParams` 等），而该 PRD 自己也承认「本 PRD 对 adaptor 接口的改造在 IR PRD 中会被推倒重来，这是预期行为」。

**结论：先 IR 后 SDK。** IR 边界定清后，把 Outbound 内部实现换成官方 SDK 是局部替换，不影响上层。反之先做 SDK 迁移，那部分接口改造会白做一遍。

SDK 迁移中唯一值得抢先处理的是 BaseURL 双重版本前缀 bug（用户填 `https://ark.cn-beijing.volces.com/api/v3`，系统拼出 `.../api/v3/v1/chat/completions`）。该问题与 IR 无关，且属真实故障，放入阶段 0.1。

> 注：BaseURL bug 的描述来自 SDK 迁移 PRD，本次未独立复现验证，阶段 0.1 需先复现再修。



## 2. 依据与边界



### 2.1 结论来源分层

避免把参考项目的实现当作本仓库的事实依据：


| 层级          | 来源                                                                                                                                           | 对本 PRD 的作用                                                                 |
| ----------- | -------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| **本仓库代码**   | §1.2 的 D1–D6，全部附文件行号                                                                                                                         | 唯一的问题定性依据。所有缺陷判断以本仓库代码为准                                                   |
| **本仓库既有约定** | `internal/relay/AGENTS.md`（chat↔responses 单点入口规则、流式保护、计费边界）                                                                                  | IR 化后需推广而非推翻，见阶段 10                                                        |
| **参考实现**    | `参考项目/axonhub`：`llm/transformer/interfaces.go` 的 Inbound/Outbound 双侧接口、`llm/model.go` 的统一 IR、`llm/transformer/anthropic/usage.go` 的 usage 归一 | 提供接口形状与命名参考。**不作为需求依据**，其覆盖面（文本为主）小于本项目（含 audio/image/embedding/rerank/ws） |
| **待查官方规范**  | Responses 各工具类型语义、Gemini `thoughtSignature` 生命周期                                                                                             | 阶段执行前须查证，不得凭记忆实现                                                           |




### 2.2 明确不作为动机

以下诉求容易导致目标漂移，显式排除：

1. **不追求「IR 无损」这个提法。** 无损是不可达的：`web_search` 转到 Chat 必然降级（D3）、Gemini 的 `thoughtSignature` 转到 Claude 无对应概念。正确目标是**有损点显式化**——每条边声明能力降级并可观测，而不是静默丢弃。
2. **不追求「全量请求过 IR」。** axonhub 全量过 IR，但本项目已有同协议直通优化（`channel/claude/adaptor.go:33-35` 的 `ConvertClaudeRequest` 直接返回原请求；DeepSeek/Moonshot/OpenRouter 的原生 Anthropic 端点透传）。这些优化必须保留，见 §3.3。
3. **不重写流式转换器。** `wire/stream` 的 chat↔responses 逐帧转换器已处理大量边界情况，IR 化是复用与收口，不是重新实现。



### 2.3 非目标

- 不改渠道 URL 构建与认证逻辑（`GetRequestURL` / `SetupRequestHeader` 原样保留）
- 不改路由对外路径、数据库 schema、模型倍率数值、审计逻辑、前端
- 不改重试策略本身，只适配错误模型
- Realtime/WebSocket 暂不纳入 IR（阶段 10 评估）
- 不引入官方 SDK（见 §1.4）



## 3. 架构设计



### 3.1 IR 选型：OpenAI Chat 超集


| 候选                 | 结论与理由                                                                                                                                                                 |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **OpenAI Chat 超集** | **采纳**。现有转换资产以其为轴；`shared.ClaudeUsageToOpenAIUsage`（`domain/shared/claude.go:609-615`）已按 OpenAI 语义做 `input + cache_read + cache_creation` 归一，与 axonhub `llm.Usage` 同构 |
| OpenAI Responses   | 否。携带会话状态（`previous_response_id`）、服务端内建工具、item id 等产品语义，让 Claude/Gemini 请求先变成 Responses 等于强迫所有协议背负这些概念                                                                 |
| 全新中立 IR            | 否。等价于重写全部转换算子，收益不足以覆盖风险                                                                                                                                               |


**Usage 语义统一为 OpenAI 语义**（`prompt_tokens` 含缓存 token）。依据是 `ClaudeUsageToOpenAIUsage` 已经这么做了，且与 axonhub 一致。落地后 `usage.go:108` 的 `isClaudeUsageSemantic` 分支可以删除——语义由边契约保证，不再依赖「渠道记得打标」，这是 D4 的根治方式而非打补丁。

### 3.2 三层正交分离

```
客户端 HTTP
    │
    ├─ Inbound（每协议一份）：客户端协议 ↔ IR
    │
  IR（OpenAI Chat 超集）
    │
    ├─ Outbound（每协议一份，不是每渠道）：IR ↔ 上游协议
    │
    ├─ 渠道适配层（每渠道）：URL 构建 / 认证头 / 方言补丁 ← 本 PRD 不改
    │
上游 HTTP
```

关键边界：**协议转换只负责请求体与协议语义必需的头**（如 Claude 的 `anthropic-version`、`anthropic-beta`）。密钥、代理、渠道基址、组织头属于渠道适配层，与 IR 无关。这是 §1.3 调查结论的直接推论——方言集中在渠道层，所以出站边可以按标准协议实现，17 个渠道共用 4 条主干边。

### 3.3 同协议直通必须保留

当 Inbound 协议与 Outbound 协议相同、且无渠道级请求改写时，**跳过 IR 往返**，直接透传。

理由：现状已有此优化（见 §2.2 第 2 点），若 IR 化后强制 `Claude→IR→Claude` 往返，会引入两次 marshal 开销和保真风险，纯属倒退。

这也是 IR 化后 `NewEmptyUsageRetryError` 判定的复用点，见 §4.3。

### 3.4 IR 需承载的非 Chat 概念

Chat 作为 IR 底座缺三类概念，必须在超集中补齐，且不能塞进现有文本字段：


| 概念        | 各协议来源                                                                                                                                                                                                                           | 承载要求                                |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| 思考签名      | Claude thinking block 的 signature（`domain/shared/claude.go`）；Gemini `thoughtSignature`（`domain/shared/gemini.go`，含 `GetThoughtSignature()`）；Responses reasoning item 签名（`domain/shared/openai_response.go`、`openai_request.go`） | 与所属内容块绑定，不可展平为全局字段。详见阶段 2/4/6       |
| 服务端工具调用计数 | Responses 内建工具（`relay/common/relay_info.go:382-388` 初始化，`channel/openai/relay_responses.go:81,145,149` 递增）；Claude/Gemini 走 gin 字符串键 `claude_web_search_requests`、`gemini_web_search_requests`                                   | 升为 IR 一等字段，替代 gin context 旁路。详见阶段 7 |
| 工具名代理上下文  | `OpenAIWireToolContext`（挂在 `RelayInfo.OpenAIResponsesToolContext`），负责 namespace 扁平化与还原、custom tool 代理                                                                                                                           | 迁入 IR 会话态，不再挂 RelayInfo。详见阶段 1      |




## 4. 计费专项复核

这一节独立成章，因为它是本次重构风险最高的部分，且现状已有确认的偏差。

### 4.1 问题定性：计费读了转换后的数据

完整链路是：

```
用户请求 → [转换] → 上游请求 → 上游响应 → [转换] → 用户响应
                                    ↑
                          计费必须在此处取数（上游原始语义）
```

现状偏差分两类：

**类型 A：读到转换后的请求体。**

`relay/wire/auto_convert.go` 在协议转换后调用 `setTemporaryRequestBody(c, bodyBytes)` 直接覆写 gin 请求体，靠 `takeRequestBodySnapshot`/`restore` 与 `takeRelayInfoSnapshot`/`restore` 成对恢复。在这个窗口期内，任何通过 `GetRequestBody` 读请求体的逻辑（计费、日志、审计、token 估算）拿到的都是转换后的体，而不是用户原始输入。

**类型 B：语义标记丢失。**

即 D4。`FinalRequestRelayFormat` 漏设导致 usage 按错误语义计算，AWS Bedrock 的 Claude 模型缓存计费偏差。

### 4.2 五条强制原则


| 编号  | 原则                           | 说明                                                     |
| --- | ---------------------------- | ------------------------------------------------------ |
| P1  | usage 归一化在 Outbound 边完成      | 产出物必须是 IR usage 且语义唯一（OpenAI 语义），下游消费者不得再按协议分支判断       |
| P2  | 计费取数只认 Outbound 产出的 IR usage | 禁止从 gin context 请求体、转换后 DTO 反推                         |
| P3  | 原始请求体快照独立于转换链                | 供日志/审计/token 估算使用，不受 `setTemporaryRequestBody` 影响      |
| P4  | 服务端工具计数是 IR 一等公民             | 替代 `c.Set("claude_web_search_requests")` 这类 gin 字符串键旁路 |
| P5  | 工具计价的 provider 维度跟随实际上游      | 不是入站格式。对应 `GetToolBillingPrice` 的 `provider` 参数        |




### 4.3 需重新定义的既有耦合

**空 usage 重试判定会全局失效。** `domain/billing/quota.go:33-38` 的 `NewEmptyUsageRetryError` 以 `len(relayInfo.RequestConversionChain) > 1` 判断「发生过协议转换就不重试」。IR 化后几乎所有请求都有转换链，这个兜底会对所有请求失效。

必须改为显式的「上游原生直通」标记，复用 §3.3 的直通判定结果，而不是数转换链长度。

**其余两处：**

- `BuildContextPricingUsage(usage, isClaudeUsageSemantic)`：语义参数随 P1 落地后移除
- 预扣与 token 估算（`TokenCountMeta`、`SetEstimatePromptTokens`）：必须基于原始入站请求（P3），不得基于转换后请求



### 4.4 现有计费函数清单

`domain/billing/quota.go` 现有：`PreConsumeTokenQuota`、`PostConsumeQuota`、`PostClaudeConsumeQuota`、`PostAudioConsumeQuota`、`PreWssConsumeQuota`、`PostWssConsumeQuota`、`CalcOpenRouterCacheCreateTokens`。`domain/billing/usage.go:74` 的 `CalculateUsage` 是核心计算入口。

本 PRD **不合并**这些函数——它们的差异来自倍率与模态，是合理差异。本 PRD 只保证喂给它们的 usage 已按 P1 归一。函数合并如有必要另开 PRD。

## 5. 分阶段实施

每阶段独立可验证、可回滚，一阶段一 PR。阶段 1–8 按协议对推进，阶段 9 按协议做错误专项，阶段 0 与 10 为前置和收尾。

### 阶段 0：基础准备与存量 bug 修复

本阶段不引入 IR 行为变更，目标是「把地基和安全网准备好，把已知的存量 bug 先修掉」。三个子阶段可并行。

#### 0.1 修存量 bug（与 IR 无关，先修先发）

**bug 1：AWS Bedrock 的 Claude 模型缓存计费偏差（D4）**

现状：`channel/aws/relay_aws.go` 复用 `claude.HandleClaudeResponseData` 等函数处理响应，但不设置 `info.FinalRequestRelayFormat`，导致 `domain/billing/usage.go:108` 的 `isClaudeUsageSemantic` 为 false，Anthropic 语义的 usage 被按 OpenAI 语义计算——缓存 token 被从基础 token 中错误减去。

修法：AWS 渠道的 `DoResponse` 补设 `info.FinalRequestRelayFormat = relayconstant.RelayFormatClaude`，与 `channel/claude/adaptor.go:121` 对齐。同时排查 Vertex 的 Claude 路径是否有同类问题（Vertex 有 `ConvertClaudeRequest` 但未确认是否复用 claude 包处理函数，需先查证）。

这是打补丁而非根治——根治在阶段 8（P1 让语义由边契约保证）。但存量偏差影响真实计费，不能等到阶段 8。

**bug 2：BaseURL 双重版本前缀**

来自 SDK 迁移 PRD 的描述：用户填写带版本前缀的 BaseURL（如 `https://ark.cn-beijing.volces.com/api/v3`）后，拼出的上游 URL 变成 `.../api/v3/v1/chat/completions`。

**本次未独立验证，需先复现再修。** 若复现失败则移出本 PRD。

**验收**：

- AWS Claude 渠道缓存计费用例（含 `cache_read_input_tokens` > 0 场景）结果与直连 Claude 渠道一致
- `go test ./internal/domain/billing/... ./internal/relay/channel/aws/...`
- BaseURL 修复后，带版本前缀与不带版本前缀的渠道配置都能拼出正确 URL



#### 0.2 IR 包骨架

新建 IR 包，只定义结构不接线，本子阶段结束时无任何调用方。

```
internal/relay/ir/
├── request.go          ← IR 请求：Chat 超集 + 签名承载 + 工具声明
├── response.go         ← IR 响应（非流式）
├── stream.go           ← IR 流式事件
├── usage.go            ← IR usage（OpenAI 语义）+ 服务端工具计数
├── error.go            ← IR 错误模型（阶段 9 使用）
├── signature.go        ← 思考签名：与内容块绑定的承载结构
├── toolcontext.go      ← 工具名代理上下文（自 wire/convert/openai_wire_tool_context.go 迁入）
└── capability.go       ← 能力降级声明：记录每条边丢弃了什么
```

字段来源不是凭空设计，而是从现有结构归并：


| IR 结构            | 来源                                                                                                                                                                 |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ir.Request`     | `shared.GeneralOpenAIRequest` 为底，补 §3.4 三类概念                                                                                                                       |
| `ir.Usage`       | `shared.Usage` 为底（已含 `PromptTokensDetails.CachedTokens`/`CachedCreationTokens`/`ClaudeCacheCreation5mTokens`/`ClaudeCacheCreation1hTokens`、`AudioTokens`），补服务端工具计数 |
| `ir.Error`       | `shared.NookMuxError` 的 `RelayError any` 收敛为强类型（阶段 9 展开）                                                                                                           |
| `ir.ToolContext` | `convert.OpenAIWireToolContext` 平移                                                                                                                                 |
| `ir.Capability`  | 新增，D3 的解药                                                                                                                                                          |


`capability.go` 是本次新增的核心概念：每条转换边在发生能力降级时（如 `web_search` 转 Chat）必须写入降级记录，供响应标注、日志和计费决策使用。这是 §2.2 第 1 点「有损点显式化」的落地载体。

**验收**：

- `go build ./internal/relay/ir/`
- IR 包不 import 任何 relay 内部包（对齐 `wire/convert` 现有约束）：`go list -deps ./internal/relay/ir/` 输出不含 `internal/relay/channel`、`internal/relay/handler`、`internal/relay/common`
- IR 结构字段与 `shared.Usage` 字段做覆盖性对照，列出未覆盖字段清单并逐项说明原因



#### 0.3 测试与计费对拍框架

三个框架，后续每个阶段都依赖它们：

**框架 1：roundtrip 测试。** 复用 `wire/convert/openai_wire_convert_request_roundtrip_test.go`、`openai_wire_convert_response_roundtrip_test.go` 的既有模式，扩展为「任意协议 → IR → 同协议」的通用断言，要求 bit 级等价。

**框架 2：流式 golden 测试。** 复用 `wire/stream/openai_wire_stream_conversion_test.go` 模式，为每条边固化 SSE 事件序列快照，断言 chunk 顺序、`finish_reason`、usage 帧位置、`[DONE]`、错误帧。参考项目 axonhub 用 `testdata/*.jsonl` 存流式样本，可借鉴该组织方式。

**框架 3：计费对拍。** 同一请求分别走新旧路径，断言喂给 `CalculateUsage` 的 usage 与最终 quota 完全一致。这是阶段 8 的验收依据，必须在阶段 1 之前就位，否则后续 8 个阶段的计费影响无从验证。

**验收**：

- 三个框架各自能跑通至少一条现有链路（建议用 Chat↔Responses，因其既有测试最完整）
- 对拍框架能检出人为注入的偏差（故意改一个倍率，对拍必须失败）



### 阶段 1：Responses ↔ Chat

**现状。** 这条边是现有实现最完整的一条，已有单点入口约束：请求与非流式响应走 `wire/convert/converter.go` 的 `NewConverter` 四个方法，流式走 `wire/stream` 的 `NewChatToResponsesStreamConverter` / `NewResponsesToChatStreamConverter`。`internal/relay/AGENTS.md` 明文禁止绕过它们直调内部函数。

工具支持情况（已核实）：


| 工具类型                                                     | 现状                                                                                                                                                                                                                                                                                         |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `function`                                               | 支持，含 namespace 扁平化（`convert_request_tools.go:351`）与超长名 sha256 截断（`:485`）                                                                                                                                                                                                                   |
| `custom`                                                 | 支持（`convert_request_tools.go:381`、`tool_context.go:47`）                                                                                                                                                                                                                                    |
| `namespace`                                              | 支持（`convert_request_tools.go:412`）                                                                                                                                                                                                                                                         |
| `tool_search`                                            | 支持（`convert_request_tools.go:331`、`convert_response.go:370`）                                                                                                                                                                                                                               |
| `apply_patch`（此为Codex/Responses协议适配关键）                   | 支持，走 generic custom tool 通道：转为带 `input` 字段的 function call，`buildResponsesCustomToolChatDescription`（`:466`）把原 `format` 约束序列化附到 description，回程 `buildResponsesToolOutputFromChatCall`（`convert_response.go:351`）还原为 `custom_tool_call`，流式双向识别 `response.custom_tool_call_input.delta/.done` |
| `web_search` / `web_search_preview` / `image_generation` | **静默丢弃**（D3，`convert_request_tools.go:338-345`）                                                                                                                                                                                                                                            |


**问题。** 两点：一是转换器虽有单点入口但产出的是 Chat DTO 而非 IR，无法被其他协议复用；二是 D3 的静默丢弃。

**目标。** 把 `wire/convert` 与 `wire/stream` 收口为 Inbound/Outbound 形态，产出 IR 而非 Chat DTO。`OpenAIWireToolContext` 迁入 `ir.ToolContext`。`web_search` / `image_generation` 改为按能力协商处理：目标协议支持则映射，不支持则写入 `ir.Capability` 降级记录（阶段 7 定义完整矩阵，本阶段只要求不再静默丢弃）。

**验收**：

- `Responses→IR→Responses` 与 `Chat→IR→Chat` roundtrip bit 级等价
- 六类工具各有用例，`apply_patch` 重点验证 `format` 约束经 description 传递后的还原保真，以及流式 `custom_tool_call_input` 分片边界
- 流式 golden：chunk 顺序、`finish_reason`、usage 帧、`[DONE]`、错误帧
- `go test ./internal/relay/wire/... ./internal/relay/ir/...`
- 计费对拍通过（此时应无差异，因为上游行为未变）



### 阶段 2：Claude ↔ Chat

**现状。** 转换算子在 `relay/helper/convert.go`：去程 `ClaudeToOpenAIRequest`，回程 `ResponseOpenAI2Claude`、`StreamResponseOpenAI2Claude`，另有 `NormalizeCacheCreationSplit`、`buildClaudeUsageFromOpenAIUsage` 等辅助。反向（Chat→Claude）在 `channel/claude/relay_claude.go` 的 `RequestOpenAI2ClaudeMessage`、`ResponseClaude2OpenAI`、`StreamResponseClaude2OpenAI`。

流式转换是有状态的：`RelayInfo.ClaudeConvertInfo.LastMessagesType`（`relay/common/relay_info.go` 的 `LastMessageTypeNone/Text/Tools/Thinking` 四态）跟踪上一个内容块类型，用于决定 Claude 事件序列（`content_block_start`/`delta`/`stop`）的边界。

**问题。** 三点：

1. 状态机挂在 `RelayInfo` 上而非转换器内部，跨渠道复用时容易漏初始化（`channel/claude/adaptor.go:104-106` 就有一处手动兜底初始化）
2. 思考签名无 IR 承载。Claude thinking block 的 signature 是上游返回的凭据，后续请求需回传，Chat 没有对应字段
3. usage 归一在 `shared.ClaudeUsageToOpenAIUsage`（正确），但语义标记靠 `FinalRequestRelayFormat` 单点赋值（D4）

**目标。** Claude Inbound/Outbound 各自实现，`LastMessagesType` 状态机迁入流式转换器内部。签名按 §3.4 与内容块绑定承载。Claude 三档缓存字段（`ClaudeCacheCreation5mTokens`、`ClaudeCacheCreation1hTokens` 及默认档）在 IR usage 中保持独立，不可合并——三档倍率不同。

**验收**：

- `Claude→IR→Claude` bit 级保真，含 thinking block 与 signature
- 三档缓存 usage 字段无损，倍率计算与现状一致（对拍）
- 状态机迁移后，连续 tool_use 块、thinking 与 text 混排、空 delta 等边界场景流式输出正确
- AWS 复用路径同步验证（`go test ./internal/relay/channel/aws/...`）
- `go test ./internal/relay/helper/... ./internal/relay/channel/claude/...`



### 阶段 3：Responses ↔ Claude

**现状。** 已通，但是手工拼装：去程 `channel/claude/adaptor.go:94-113` 串联两次转换并手动传递 `ToolContext`、初始化 `ClaudeConvertInfo`、追加 `RequestConversionChain`；回程流式在 `relay_claude.go:992-1005` 走 `writeClaudeChatChunkAsResponsesEvent`（`:1094`）+ `ensureClaudeResponsesStreamConverter`（`:1087`）+ `writeClaudeResponsesFinalEvent`（`:1116`）约 130 行；回程非流式在 `:1205-1218`。

**问题。** 这 130 行胶水是 D2 的典型样本——它做的事情本质上是「组合阶段 1 和阶段 2 的边」，但因为没有 IR 层，只能在渠道内部手写串联。每个想支持 Responses 的渠道都要抄一遍，这就是 Gemini/Vertex/AWS 至今 `not implemented` 的原因。

**目标。** 不新增任何点对点转换代码，纯粹由 `Responses↔IR`（阶段 1）与 `IR↔Claude`（阶段 2）组合得到。删除上述胶水。

**验收**：

- 现有 Claude + Responses 行为不回退（用现网请求样本对拍）
- 内建工具按能力协商处理而非静默丢弃
- 删除的胶水代码无残留引用：`rg "writeClaudeChatChunkAsResponsesEvent|ensureClaudeResponsesStreamConverter|writeClaudeResponsesFinalEvent" internal/` 无输出
- `go test ./internal/relay/...`



### 阶段 4：Gemini ↔ Chat

**现状。** 去程 `helper.GeminiToOpenAIRequest`，回程 `helper.ResponseOpenAI2Gemini`、`helper.StreamResponseOpenAI2Gemini`。流式状态在 `RelayInfo.GeminiConvertInfo`，辅助函数 `ensureGeminiConvertInfo`、`ensureGeminiChoiceToolCallState`。

思考签名现状最脏：`channel/gemini/relay_gemini.go` 有一个绕行常量（值为 `context_engineering_is_the_way_to_go`）和 `parseFunctionCallSignature`，在渠道类型为 Gemini 或请求显式要求时附加签名。这是既有 workaround，不是设计。

Gemini 还有一个特殊性：上游会把 429/5xx 转成 HTTP 200 + `{"error":{...}}` 下发，`domain/shared/gemini.go:558` 的 `GeminiChatResponse.Error` 字段专门保留这个载荷供 handler 识别真实错误。

**问题。** 除 §3.4 的签名承载缺失外，还有 D5 的一半：Gemini 客户端收到 OpenAI 格式错误（本阶段只做转换层，错误出口在阶段 9.3 处理，但本阶段需确保 IR 能承载 Gemini 错误载荷）。

**目标。** Gemini Inbound/Outbound 实现，`GeminiConvertInfo` 状态机迁入转换器内部。`thoughtSignature` 按 §3.4 承载，清理绕行常量与 workaround——签名应由 IR 正常传递，不需要魔法值。HTTP 200 携带错误载荷的识别逻辑保留，纳入 Outbound 的响应解析。

**验收**：

- `Gemini→IR→Gemini` bit 级保真，含 `thoughtSignature`
- 绕行常量删除后签名正常传递：`rg "context_engineering_is_the_way_to_go" internal/` 无输出
- HTTP 200 + error 载荷仍被正确识别为上游错误（不能被当成正常响应计费）
- Gemini audio token 计费不变（`GetGeminiInputAudioPricePerMillionTokens` 路径对拍）
- `go test ./internal/relay/channel/gemini/... ./internal/relay/helper/...`



### 阶段 5：Responses ↔ Gemini

**现状。** 不通。`channel/gemini/adaptor.go:228` 与 `channel/vertex/adaptor.go:299` 均为 `not implemented`（D1）。

**问题。** 零件齐备但无人组合，同阶段 3 的成因。

**目标。** 由阶段 1 与阶段 4 的边组合得到，消除两处 `not implemented`。同时处理 Gemini 原生 Google Search grounding 与 Responses `web_search` 的映射：Responses 客户端声明 `web_search` 时，映射到 Gemini 的 grounding 能力；计费按实际执行方（Gemini）取价，对应 P5。

**验收**：

- Responses 客户端可正常使用 Gemini 渠道（流式与非流式）
- Vertex 同步打通
- grounding 计费正确：`gemini_web_search_requests` 计数经 IR 传递，`GetToolBillingPrice(provider="gemini")` 取价（对拍）
- `rg "not implemented" internal/relay/channel/{gemini,vertex}/adaptor.go` 无 Responses 相关残留
- `go test ./internal/relay/channel/gemini/... ./internal/relay/channel/vertex/...`



### 阶段 6：Gemini ↔ Claude

**现状。** 部分通且实现分散：Gemini 客户端 → Claude 上游在 `channel/claude/adaptor.go:26` 经 `helper.GeminiToOpenAIRequest`；Claude 上游 → Gemini 客户端在 `channel/claude/relay_claude.go:1006-1027`（流式）与 `:1220-1229`（非流式）。反向（Claude 客户端 → Gemini 上游）在 `channel/gemini/adaptor.go:46` 经 `helper.ClaudeToOpenAIRequest`。

**问题。** 这是唯一一条两端都非 Chat 的边，签名跨协议问题在此暴露：Gemini 的 `thoughtSignature` 与 Claude thinking block 的 signature 是两套互不兼容的上游凭据，不能互相转换。现状没有明确策略。

**目标。** 由阶段 2 与阶段 4 的边组合得到。签名跨协议按「丢弃并记入 `ir.Capability`」处理，**严禁伪造**——把 Gemini 签名塞进 Claude 的 signature 字段会导致上游拒绝请求或推理链断裂。删除 `relay_claude.go` 的 Gemini 分支胶水。

**验收**：

- 双向可用（流式与非流式）
- 签名不伪造：`Gemini→IR→Claude` 与 `Claude→IR→Gemini` 用例断言目标协议签名字段为空且 `ir.Capability` 有降级记录
- 删除的胶水无残留引用
- `go test ./internal/relay/...`



### 阶段 7：内建工具全矩阵复核

**现状。** 服务端工具的计数与计价分三套互不相通的机制：


| 机制                                    | 位置                                                                                                                | 覆盖范围             |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------------------- | ---------------- |
| `ResponsesUsageInfo.BuiltInTools` map | `relay/common/relay_info.go:382-388` 按请求声明初始化（CallCount=0），`channel/openai/relay_responses.go:81,145,149` 按上游事件递增 | 仅原生 Responses 上游 |
| gin 字符串键 `claude_web_search_requests` | `channel/claude/relay_claude.go:960,1233` 从 Claude usage 的 `ServerToolUse.WebSearchRequests` 写入                   | 仅 Claude 上游      |
| gin 字符串键 `gemini_web_search_requests` | 由 Gemini 侧写入                                                                                                      | 仅 Gemini 上游      |


计价在 `domain/billing/usage.go:140-211`，三套分支各自读上述来源，`GetToolBillingPrice` 按 `provider` 参数（openai/claude/gemini）区分单价。

**问题。** 三点：

1. **跨协议组合下计数必然丢失。** 例如 Responses 客户端 + Claude 上游：`BuiltInTools` 被请求声明初始化，但 Claude 上游不会返回 `response.output_item.done` 事件，CallCount 永远为 0。当前之所以没造成错账，是因为 D3 把工具丢弃了，上游根本没执行——两个 bug 互相掩盖。阶段 1 修掉 D3 之后，如果计数机制不统一，就会变成真实漏计费。
2. **gin 字符串键无类型约束**，拼错静默失效。
3. **能力矩阵无文档**，哪些组合支持搜索、哪些降级，只能靠读代码推断。

**目标。** 落地 P4 与 P5：服务端工具调用记录升为 IR 一等字段，由各协议 Outbound 负责把自己的上游事件/usage 字段映射进来；计费层只读 IR，按记录中的实际执行方取价。

产出完整能力矩阵文档，每格明确三种结果之一：原生映射、降级为函数工具、拒绝请求。


| 客户端声明 → 上游协议       | Chat                    | Claude                  | Gemini                     | Responses |
| ------------------ | ----------------------- | ----------------------- | -------------------------- | --------- |
| `web_search`       | 降级为函数工具（客户端执行）或拒绝，按渠道配置 | 映射 Claude 原生 web search | 映射 Google Search grounding | 原生        |
| `file_search`      | 降级或拒绝                   | 无对应能力，降级或拒绝             | 无对应能力，降级或拒绝                | 原生        |
| `image_generation` | 降级或拒绝                   | 无对应能力                   | 无对应能力                      | 原生        |


矩阵每格的最终结论以阶段执行时查证的上游官方文档为准，上表是待验证草案。

**验收**：

- 矩阵每格有测试用例
- 跨协议组合的工具计数不丢：Responses 客户端 + Claude 上游声明 `web_search`，若映射成功则按 Claude 单价计费（对拍），若降级则有 `ir.Capability` 记录且不计工具费
- gin 字符串键旁路清除：`rg "claude_web_search_requests|gemini_web_search_requests" internal/` 仅在兼容层出现或无输出
- `go test ./internal/domain/billing/... ./internal/relay/...`



### 阶段 8：计费专项复核

**现状。** 见 §4.1 的类型 A 与类型 B。

**问题。** 这是本次重构最容易埋雷的地方：前七个阶段每改一条边，都可能改变喂给 `CalculateUsage` 的数据。阶段 0.3 的对拍框架就是为此准备的，但对拍只能发现偏差，不能保证语义正确——需要本阶段做系统性收口。

**目标。** 逐条落地 §4.2 的 P1–P5，并处理 §4.3 的三处耦合：

1. **删除** `isClaudeUsageSemantic` **分支**（`usage.go:108` 及其在 `BuildContextPricingUsage` 的传参）。前置条件是所有 Outbound 已按 P1 归一 usage 语义，届时该分支恒为一个值，属死代码。这同时根治 D4——不再依赖渠道记得打标。
2. **重新定义空 usage 重试判定**（`quota.go:33-38`）。从「转换链长度 > 1」改为「非上游原生直通」，复用 §3.3 的直通判定。
3. **原始请求体快照独立化**（P3）。确保 token 估算与日志读的是用户原始输入，不受 `setTemporaryRequestBody` 窗口影响。

**验收**：

- 全渠道 × 全协议组合计费对拍全绿（这是本阶段核心验收项）
- Claude 三档缓存、Gemini audio token、Responses 内建工具、OpenRouter 缓存创建（`CalcOpenRouterCacheCreateTokens`）逐项回归
- 空 usage 重试语义符合预期：原生直通请求空 usage 应重试，跨协议转换请求不重试
- 死代码清除：`rg "isClaudeUsageSemantic" internal/` 无输出
- `go test ./internal/domain/billing/...` 含 `usage_test.go` 全部用例



### 阶段 9：三家协议错误处理专项

错误路径与转换路径同层但独立收口，按协议拆三个子阶段。共同前提：`ir.Error`（阶段 0.2 已定义骨架）作为唯一内部错误模型，各协议 Inbound 负责 IR→客户端格式，Outbound 负责上游格式→IR。

三家共有的两个问题先说清楚：

**共有问题 1：出口分发重复两处。** `httpapi/controller/relay/relay.go:98-110` 与 `httpapi/middleware/performance.go:23-31` 各有一套按格式分发的 switch。改一处漏另一处。

**共有问题 2：脱敏分散三处。** `domain/shared/error.go` 的 `ToOpenAIError`、`ToClaudeError` 各调一次 `MaskSensitiveInfoWithExemptions`，`relay/helper/error.go:47` 的 `ClaudeErrorWrapper` 另有一套手写替换逻辑（把含 post/dial/http 的错误文本替换为「请求上游地址失败」）。

#### 9.1 OpenAI 系错误（Chat + Responses）

**现状。** 入站解析在 `relay/helper/error.go:71` 的 `RelayErrorHandler`：读上游错误体，尝试解析为 `shared.GeneralErrorResponse`，再用 `TryToOpenAIError()`（`domain/shared/error_response.go:40`）提取。解析失败则退化为 body 文本拼接。出站是 `NookMuxError.ToOpenAIError()`（`error.go:197`）。

上游状态码还原有专门逻辑：`helper/error.go` 的 `UpstreamErrorStatusCode`、`ResetStatusCode` 与 `upstreamErrorStatusCodeByCode` 映射表，`NookMuxError.OriginalStatusCode` 保留渠道级状态码映射前的原始值（用于重试与渠道禁用判定）。

**问题。** OpenAI 系是当前的「归一中心」，本身问题最少。主要问题是它承担了所有非 Claude 格式的兜底出口——包括 Gemini（见 9.3）。另外 `GeneralErrorResponse.Error` 是 `json.RawMessage`，`TryToOpenAIError` 只处理 object 类型，string/number 类型的 `error` 字段走 `ToMessage()` 降级为纯文本，丢失结构。

**目标。** `RelayErrorHandler` 拆为 OpenAI Outbound 的 `TransformError`（上游 OpenAI 格式 → `ir.Error`），出口拆为 OpenAI Inbound 的 `TransformError`（`ir.Error` → OpenAI 格式）。`ir.Error` 保留 `OriginalStatusCode` 与错误码，确保重试与渠道禁用判定行为不变。Responses 格式的错误出口与 Chat 共用（同属 OpenAI 系，错误体结构一致）。

**验收**：

- 上游各类错误体（标准 object、error 为 string、error 为 number、非 JSON body、空 body）解析结果与现状一致
- `StatusCode` / `OriginalStatusCode` 映射不变，重试与渠道自动禁用判定回归
- `go test ./internal/relay/helper/... ./internal/domain/shared/...`



#### 9.2 Claude 错误

**现状。** 上游错误识别在 `channel/claude/relay_claude.go:1184-1187`：从响应体取 `GetClaudeError()`，用 `shared.ClaudeErrorStatusCode(claudeError.Type)` 按 Anthropic 官方文档把错误类型还原为真实 HTTP 状态码（注释明确说明这是为了保持重试与渠道禁用判断准确）。流式错误拦截另有 `stream_error_interception_test.go` 覆盖。

出站是 `ToClaudeError()`（`error.go:230`）。本地错误包装是 `helper/error.go:47` 的 `ClaudeErrorWrapper`。

**问题。** 跨格式映射错误：`ToClaudeError()` 在源错误类型为 OpenAI 时，执行 `Type: fmt.Sprintf("%v", openAIError.Code)`——把 OpenAI 的 `code`（如 `invalid_model`、数字 400）塞进 Anthropic 的 `type` 字段。而 Anthropic 的 `type` 是固定枚举（`invalid_request_error`、`authentication_error`、`rate_limit_error`、`api_error`、`overloaded_error` 等），塞入非枚举值会导致严格按 Anthropic 规范解析的客户端行为异常。

反向也有问题：`ToOpenAIError()` 在源类型为 Claude 时执行 `Type: claudeError.Type`，把 `overloaded_error` 这种 Anthropic 专有类型原样丢给 OpenAI 客户端。

**目标。** 建立显式的错误类型映射表（HTTP 状态码 + 语义 → 各协议合法枚举值），替代当前的字段直塞。映射表要双向且只产出目标协议的合法枚举值。`ClaudeErrorStatusCode` 的状态码还原逻辑保留并纳入 Claude Outbound。`ClaudeErrorWrapper` 的文本替换逻辑并入统一脱敏单点。

**验收**：

- Claude 上游各类错误类型的状态码还原与现状一致
- 跨格式映射产出合法枚举：`OpenAI→ir.Error→Claude` 的 `type` 字段断言在 Anthropic 枚举集合内；`Claude→ir.Error→OpenAI` 同理
- 流式错误拦截行为不变（`stream_error_interception_test.go` 通过）
- `go test ./internal/relay/channel/claude/...`



#### 9.3 Gemini 错误

**现状。** 这是三家里问题最大的一家，D5 的主体。

已经铺好的底子：`domain/shared/error.go:32` 定义了 `ErrorTypeGeminiError = "gemini_error"`；`domain/shared/gemini.go:562` 定义了 `GeminiErrorResponse{Code int, Message string, Status string}`，即 Gemini API 标准错误载荷；`GeminiChatResponse.Error`（`gemini.go:558`）专门保留 HTTP 200 中携带的错误载荷。

没接上的部分：`ErrorTypeGeminiError` 全仓库零使用；`NookMuxError` 没有 `ToGeminiError()` 方法；出口分发（`relay.go:98-110`）只有 Realtime/Claude/default 三路。

**问题。** Gemini 客户端报错时走 default 分支，收到 OpenAI 格式错误体：

- 期待 `{"error":{"code":429,"message":"...","status":"RESOURCE_EXHAUSTED"}}`
- 实际 `{"error":{"message":"...","type":"...","code":"..."}}`

`status` 字段完全缺失，`code` 类型从数字变字符串。按 Gemini 规范解析的客户端会拿到空 `status` 或解析失败。这不是「不够优雅」，是协议不兼容。

**目标。** 补齐 Gemini 错误出口：新增 `ToGeminiError()`（或 IR 化后的 Gemini Inbound `TransformError`），启用 `ErrorTypeGeminiError`，出口分发增加 Gemini 分支。`status` 字段需按 HTTP 状态码映射到 Google API 标准 status 枚举（`INVALID_ARGUMENT`、`RESOURCE_EXHAUSTED`、`UNAVAILABLE`、`INTERNAL` 等），映射表需查证 Google API 官方文档后确定。HTTP 200 携带错误载荷的识别逻辑纳入 Gemini Outbound。

**验收**：

- Gemini 客户端收到符合 Gemini 规范的错误体：`code` 为数字、`status` 为合法枚举、字段结构正确
- `ErrorTypeGeminiError` 已被使用：`rg "ErrorTypeGeminiError" internal/` 有实际引用
- HTTP 200 + error 载荷仍被识别为错误且不计费
- 出口分发合并为单点，`relay.go` 与 `performance.go` 不再各写一份
- `go test ./internal/relay/channel/gemini/... ./internal/httpapi/...`



### 阶段 10：实机测试与收尾

**现状。** 前九个阶段的验收都基于单元测试与对拍，未经真实上游验证。

**问题。** 协议转换的边界情况（上游返回非文档化字段、流式分片位置异常、超长内容截断行为、真实网络中断）无法靠单测覆盖。

**目标。** 三件事：

1. **实机矩阵验证。** 每个 Inbound × 每个 Outbound 组合，流式与非流式各跑一遍真实请求。矩阵规模为 4 × 4 × 2 = 32 个组合（Chat/Responses/Claude/Gemini 四种客户端格式 × 四种上游协议 × 流式/非流式），逐格记录结果。
2. **旧代码物理删除。** 前置是 deprecated 观察期通过。删除清单：`relay/helper/convert.go` 的协议转换函数、各渠道 `Convert*Request` 的冗余实现、各渠道 `DoResponse` 的四路 switch、`wire/auto_convert.go` 的快照恢复机制（若 IR 化后不再需要覆写请求体）。
3. **文档同步。** `internal/relay/AGENTS.md` 的单点入口规则从「chat↔responses 必须走 `convert.NewConverter`」推广为「所有协议转换必须走 IR 边」；依赖方向段落按新包结构更新；`docs/` 下涉及协议转换的开发文档同步。

**Realtime 评估。** `relay/core/websocket.go` 的 `WssHelper` 目前是独立入口，不走 adaptor 的 Convert 方法，计费走 `PreWssConsumeQuota`/`PostWssConsumeQuota`。WebSocket 的双向流模型与 IR 的单向流模型差异较大，本阶段只做评估并给出结论，不强制纳入。倾向暂缓，保留旁路。

**验收**：

- 32 格实机矩阵全绿，逐格留记录（含请求样本与响应快照）
- `go build ./...` 与 `go test ./...` 全绿
- 旧代码删除后无残留引用，`gofmt -l` 与 `go vet ./...` 干净
- `internal/relay/AGENTS.md` 与代码一致
- Realtime 处置结论已写入本文档



## 6. 风险与缓解


| 风险                      | 影响             | 缓解                                                                                   |
| ----------------------- | -------------- | ------------------------------------------------------------------------------------ |
| 流式状态机迁移导致 chunk 顺序或事件错乱 | 客户端流式输出异常      | 每阶段流式 golden 测试（阶段 0.3 框架 2）；复用既有 `wire/stream` 实现而非重写（§2.2 第 3 点）                   |
| 思考签名跨协议保真失败             | 上游拒绝后续请求，推理链断裂 | 阶段 2/4 要求同协议 bit 级 roundtrip；阶段 6 要求异构组合明确丢弃并记录，严禁伪造                                 |
| 计费对拍覆盖不全导致偏差            | 直接资金影响         | 阶段 0.3 先建对拍框架再动转换代码；阶段 8 全组合对拍；上线分渠道灰度                                               |
| 同协议直通被 IR 往返破坏          | 延迟上升，保真风险      | §3.3 直通判定，同协议跳过 IR                                                                   |
| 阶段 1 修掉 D3 后暴露漏计费       | 服务端工具漏计费       | D3 与工具计数缺失当前互相掩盖（阶段 7 问题 1），阶段 1 与阶段 7 之间不得上线中间态                                     |
| 空 usage 重试判定全局失效        | 上游异常时不再重试      | 阶段 8 明确改为直通判定，并补用例覆盖两种分支                                                             |
| 存量线上用户回归                | 服务中断           | deprecated 过渡 + feature flag 分渠道灰度，不硬删（对齐参考 PRD 的删除策略）                               |
| IR 表达力不足                | 上游能力丢失         | Chat 超集 + §3.4 三类概念显式承载；`ir.Capability` 记录降级。**禁止**用通用 `extra map` 承载已知协议概念，避免退化为垃圾桶 |




## 7. 完成标准

- 全部协议互转组合可用，无 `not implemented` 残留
- 每个 Inbound × Outbound 组合有 roundtrip 与流式 golden 测试
- 内建工具能力矩阵文档化，每格结论明确，无静默丢弃
- 思考签名同协议往返 bit 级保真，跨协议行为明确且不伪造
- 三家协议错误出口符合各自规范，Gemini 客户端收到 Gemini 格式错误
- 计费对拍全绿；计费取数不依赖转换后请求体；`isClaudeUsageSemantic` 类语义分支已删除
- 旧协议转换与胶水代码物理删除
- `go build ./...`、`go test ./...`、`gofmt -l`、`go vet ./...` 全部干净
- `internal/relay/AGENTS.md` 已更新



## 8. 待决问题

1. **Gemini status 枚举映射表**：HTTP 状态码 → Google API status（`INVALID_ARGUMENT`/`RESOURCE_EXHAUSTED`/…）的映射需查 Google API 官方文档确定，阶段 9.3 执行前完成。
2. **能力降级的对外可观测方式**：`ir.Capability` 的降级记录如何告知客户端——响应扩展字段、响应头、仅记日志，或组合。需产品决策，影响阶段 1 与阶段 7。
3. **签名跨协议丢弃是否需提示客户端**：避免客户端静默丢失推理链而不自知。同上属产品决策。
4. `file_search` **/** `image_generation` **在非 Responses 上游的处置**：降级为函数工具（客户端无法真正执行文件检索）还是直接拒绝请求。倾向拒绝，但需确认是否有存量用户依赖当前的静默丢弃行为。
5. **Realtime 是否纳入 IR**：阶段 10 评估，倾向暂缓。
6. **BaseURL 双重前缀 bug 是否真实存在**：阶段 0.1 需先复现，未复现则移出本 PRD。

