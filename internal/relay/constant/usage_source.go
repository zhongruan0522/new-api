package constant

// UsageSource 标识上游 usage 的解析协议族（PRD 计费 3.1 语义映射）。
// 由各渠道响应解析点显式写入 RelayInfo.UsageSource，计费归一化
// （domain/billing.BuildBillingUsage）按它选择语义规则；
// 禁止用 FinalRequestRelayFormat 等请求侧格式反推 usage 语义。
type UsageSource string

const (
	// UsageSourceNone 表示尚无解析点写入来源；携带真实 token 用量的日志
	// 不允许停留在该状态，计费归一化遇到它必须走可诊断错误路径。
	UsageSourceNone UsageSource = ""
	// UsageSourceClaude 覆盖 Claude Messages 官方语义：原生 /v1/messages、
	// AWS Bedrock 复用路径、Vertex Claude 模式与 OpenRouter 原生
	// /api/v1/messages 等所有经 claude 包解析的返回。
	UsageSourceClaude UsageSource = "claude"
	// UsageSourceOpenAIChat 覆盖 Chat Completions 官方语义及各 OpenAI 兼容
	// 渠道（含 embeddings/rerank/audio TTS·STT/images 等按 prompt/completion
	// 口径上报的上游）。
	UsageSourceOpenAIChat UsageSource = "openai_chat"
	// UsageSourceOpenAIResponses 覆盖 Responses API 官方语义，以及 OpenAI
	// Realtime（usage 为 input_tokens/input_token_details 的同族 schema）。
	UsageSourceOpenAIResponses UsageSource = "openai_responses"
	// UsageSourceGemini 覆盖 Gemini generateContent / streamGenerateContent
	// 官方 usageMetadata 语义。
	UsageSourceGemini UsageSource = "gemini"
)
