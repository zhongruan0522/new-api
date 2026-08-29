package billing

import (
	"fmt"

	"strings"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/infra/log"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"

	"github.com/gin-gonic/gin"
)

// 本文件实现计费 PRD 阶段 1：三规范（Claude Messages、OpenAI Chat/Responses、
// Gemini）Usage 归一化为内部 BillingUsage，并序列化成 Log.billing_details 的
// canonical JSON（schema 见 PRD 第 4 章）。
//
// 阶段边界：本文件只负责"转换、校验、落库 JSON 与新旧读取兼容"，
// 不切换 quota 公式；现有计费快照与 Other 继续按原逻辑写入。

// BillingDetailsSchemaVersion 当前 billing_details JSON 的 schema 版本（PRD 4.2）。
const BillingDetailsSchemaVersion = 1

// BillingUsage 是归一化后的语义化 Token 用量（PRD 3.1 语义映射表）。
//
// 口径约定：
//   - PromptAggregateTokens 是官方 raw 输入总量（OpenAI/Gemini 已含缓存读取，
//     Claude 按 input+cache_read+cache_creation 三项相加）。
//   - InputTokens() 是扣除缓存后的普通输入，模态明细不默认从中二次扣除。
//   - 输出模态与 reasoning/accepted/rejected 是输出总量子集，不得加回。
//   - 指针字段为 nil 表示上游未返回该拆分；只有官方明确返回才写入，
//     不能用 0 伪装成"官方返回了零"。
type BillingUsage struct {
	Source relayconstant.UsageSource

	// 官方总量（计费基准与 TPM 口径）
	PromptAggregateTokens int
	OutputTokens          int

	CacheReadTokens  int
	CacheWriteTokens int
	// CacheWrite5mTokens / CacheWrite1hTokens 仅在官方明确返回分档时非 nil；
	// 无分档总量按 PRD 转换规则令 5m 分档等于总量（见 finalizeBillingUsage）。
	CacheWrite5mTokens *int
	CacheWrite1hTokens *int

	// 输入模态明细（官方字段；与普通输入是子集关系，不与缓存互计）
	TextInputTokens     *int
	ImageInputTokens    *int
	AudioInputTokens    *int
	VideoInputTokens    *int
	DocumentInputTokens *int

	// 输出拆分（输出总量子集/审计拆分）
	TextOutputTokens         *int
	AudioOutputTokens        *int
	ImageOutputTokens        *int
	ReasoningTokens          *int
	AcceptedPredictionTokens *int
	RejectedPredictionTokens *int

	// Gemini toolUsePromptTokenCount：官方总量公式未把它加入输入总量、
	// TPM 或处理总量，只作独立审计字段（PRD 3.1）。
	ToolUsePromptTokens *int

	// Warnings 保留可诊断异常（如明细大于官方总量），不阻断落库。
	Warnings []string
}

// InputTokens 普通输入 = raw 输入总量 - 缓存读取 - 缓存写入。
// Claude 的 raw 总量已按官方三项相加、OpenAI/Gemini 的 raw 总量本身含缓存，
// 因此三个规范族都用同一减法得到"不含缓存的普通输入"。
func (bu *BillingUsage) InputTokens() int {
	return bu.PromptAggregateTokens - bu.CacheReadTokens - bu.CacheWriteTokens
}

// UntieredCacheWriteTokens 未分档缓存写入 = 写入总量 - 官方 5m/1h 分档。
// 归一化校验保证非负；无分档总量经转换规则后该值为 0（已并入 5m 分档）。
func (bu *BillingUsage) UntieredCacheWriteTokens() int {
	untiered := bu.CacheWriteTokens - intValue(bu.CacheWrite5mTokens) - intValue(bu.CacheWrite1hTokens)
	if untiered < 0 {
		return 0
	}
	return untiered
}

// TotalProcessedTokens 处理总量，只用于 TPM；按官方总量公式去重，
// 不加入 Gemini toolUsePromptTokenCount（PRD 3.1）。
func (bu *BillingUsage) TotalProcessedTokens() int {
	return bu.PromptAggregateTokens + bu.OutputTokens
}

// BuildBillingUsage 把上游 usage 归一化为语义化 BillingUsage。
// source 必须由响应解析点显式写入（RelayInfo.UsageSource），禁止用
// FinalRequestRelayFormat 反推语义。Gemini 来源必须携带原始 usageMetadata
// （RelayInfo.UsageGeminiMetadata），因为转换后的 shared.Usage 已把
// toolUsePromptTokenCount 并入 prompt 总量与模态明细，无法还原官方拆分。
//
// 返回的 warnings 是不阻断落库的可诊断异常；error 表示必须放弃写入
// billing_details（负数、分档大于总量等显式失败）。
func BuildBillingUsage(source relayconstant.UsageSource, usage *shared.Usage, geminiMetadata *shared.GeminiUsageMetadata) (*BillingUsage, []string, error) {
	switch source {
	case relayconstant.UsageSourceClaude:
		return buildClaudeBillingUsage(usage)
	case relayconstant.UsageSourceOpenAIChat:
		return buildOpenAIChatBillingUsage(usage)
	case relayconstant.UsageSourceOpenAIResponses:
		return buildOpenAIResponsesBillingUsage(usage)
	case relayconstant.UsageSourceGemini:
		return buildGeminiBillingUsage(geminiMetadata)
	default:
		return nil, nil, fmt.Errorf("unknown usage source: %q", source)
	}
}

// BuildRealtimeBillingUsage 归一化 OpenAI Realtime（wss）用量。Realtime 的
// usage schema（input_tokens/input_token_details/output_tokens）与 Responses
// 同族，因此来源按 UsageSourceOpenAIResponses 归一化。
func BuildRealtimeBillingUsage(source relayconstant.UsageSource, usage *shared.RealtimeUsage) (*BillingUsage, []string, error) {
	if usage == nil {
		return nil, nil, fmt.Errorf("realtime usage is nil")
	}
	if source != relayconstant.UsageSourceOpenAIResponses {
		return nil, nil, fmt.Errorf("unexpected realtime usage source: %q", source)
	}
	bu := &BillingUsage{
		Source:                source,
		PromptAggregateTokens: usage.InputTokens,
		OutputTokens:          usage.OutputTokens,
		CacheReadTokens:       usage.InputTokenDetails.CachedTokens,
	}
	// 明细非零即透传（含负数），负数由 finalizeBillingUsage 显式拒绝，
	// 避免静默裁剪上游矛盾数据。
	if usage.InputTokenDetails.TextTokens != 0 {
		bu.TextInputTokens = &usage.InputTokenDetails.TextTokens
	}
	if usage.InputTokenDetails.AudioTokens != 0 {
		bu.AudioInputTokens = &usage.InputTokenDetails.AudioTokens
	}
	if usage.OutputTokenDetails.TextTokens != 0 {
		bu.TextOutputTokens = &usage.OutputTokenDetails.TextTokens
	}
	if usage.OutputTokenDetails.AudioTokens != 0 {
		bu.AudioOutputTokens = &usage.OutputTokenDetails.AudioTokens
	}
	if usage.OutputTokenDetails.ReasoningTokens != 0 {
		bu.ReasoningTokens = &usage.OutputTokenDetails.ReasoningTokens
	}
	return finalizeBillingUsage(bu)
}

// buildClaudeBillingUsage 归一化 Claude Messages 语义。Claude 原生、AWS
// Bedrock 复用路径、Vertex Claude 模式与 OpenRouter 原生 /api/v1/messages
// 都经 claude 包收敛到 shared.Usage 的同一批字段（PromptTokensDetails 缓存
// 读写、ClaudeCacheCreation5m/1hTokens 分档），因此共用这一条 Claude 规则。
// Claude 无标准输入/输出模态与预测拆分，对应字段保持 nil，不虚构。
func buildClaudeBillingUsage(usage *shared.Usage) (*BillingUsage, []string, error) {
	if usage == nil {
		return nil, nil, fmt.Errorf("usage is nil")
	}
	bu := &BillingUsage{
		Source:                relayconstant.UsageSourceClaude,
		PromptAggregateTokens: usage.PromptTokens,
		OutputTokens:          usage.CompletionTokens,
		CacheReadTokens: firstNonZero(usage.PromptTokensDetails.CachedTokens,
			inputDetailsCachedTokens(usage.InputTokensDetails), usage.PromptCacheHitTokens),
		CacheWriteTokens: firstNonZero(usage.PromptTokensDetails.CachedCreationTokens,
			inputDetailsCachedCreationTokens(usage.InputTokensDetails)),
	}
	// ClaudeCacheCreation5m/1hTokens 是 ClaudeUsageToOpenAIUsage 从官方
	// cache_creation 分档程序化填入的；0 视为官方未返回。
	// 分档/推理明细非零即透传（含负数），负数由 finalizeBillingUsage 显式拒绝。
	if usage.ClaudeCacheCreation5mTokens != 0 {
		bu.CacheWrite5mTokens = &usage.ClaudeCacheCreation5mTokens
	}
	if usage.ClaudeCacheCreation1hTokens != 0 {
		bu.CacheWrite1hTokens = &usage.ClaudeCacheCreation1hTokens
	}
	if tokens := firstNonZero(usage.CompletionTokenDetails.ReasoningTokens, outputDetailsReasoningTokens(usage.OutputTokensDetails)); tokens != 0 {
		bu.ReasoningTokens = &tokens
	}
	return finalizeBillingUsage(bu)
}

// buildOpenAIChatBillingUsage 归一化 Chat Completions 官方语义及 OpenAI 兼容
// 渠道来源。prompt_tokens 已含缓存读取/写入，模态明细从 prompt_tokens_details
// 透传；5m/1h 写入分档无标准字段，不虚构。
func buildOpenAIChatBillingUsage(usage *shared.Usage) (*BillingUsage, []string, error) {
	if usage == nil {
		return nil, nil, fmt.Errorf("usage is nil")
	}
	bu := &BillingUsage{
		Source:                relayconstant.UsageSourceOpenAIChat,
		PromptAggregateTokens: usage.PromptTokens,
		OutputTokens:          usage.CompletionTokens,
		CacheReadTokens: firstNonZero(usage.PromptTokensDetails.CachedTokens,
			inputDetailsCachedTokens(usage.InputTokensDetails), usage.PromptCacheHitTokens),
		CacheWriteTokens: firstNonZero(usage.PromptTokensDetails.CachedCreationTokens,
			inputDetailsCachedCreationTokens(usage.InputTokensDetails)),
	}
	fillInputModalities(bu, usage.PromptTokensDetails, usage.InputTokensDetails)
	fillOutputSplits(bu, usage.CompletionTokenDetails, usage.OutputTokensDetails)
	return finalizeBillingUsage(bu)
}

// buildOpenAIResponsesBillingUsage 归一化 Responses 官方语义。Responses 侧
// 没有标准输入图片/音频、输出图像/音频字段，只有上游明确返回的明细才透传，
// 不从其他入口补造（PRD 阶段 1）。
func buildOpenAIResponsesBillingUsage(usage *shared.Usage) (*BillingUsage, []string, error) {
	if usage == nil {
		return nil, nil, fmt.Errorf("usage is nil")
	}
	bu := &BillingUsage{
		Source:                relayconstant.UsageSourceOpenAIResponses,
		PromptAggregateTokens: firstNonZero(usage.InputTokens, usage.PromptTokens),
		OutputTokens:          firstNonZero(usage.OutputTokens, usage.CompletionTokens),
		CacheReadTokens: firstNonZero(inputDetailsCachedTokens(usage.InputTokensDetails),
			usage.PromptTokensDetails.CachedTokens, usage.PromptCacheHitTokens),
		CacheWriteTokens: firstNonZero(inputDetailsCachedCreationTokens(usage.InputTokensDetails),
			usage.PromptTokensDetails.CachedCreationTokens),
	}
	fillInputModalities(bu, usage.PromptTokensDetails, usage.InputTokensDetails)
	fillOutputSplits(bu, usage.CompletionTokenDetails, usage.OutputTokensDetails)
	return finalizeBillingUsage(bu)
}

// buildGeminiBillingUsage 从 Gemini 原始 usageMetadata 归一化。必须使用原始
// metadata：GeminiUsageMetadataToOpenAIUsage 会把 toolUsePromptTokenCount
// 并入 prompt 总量并把 toolUsePromptTokensDetails 并入输入模态明细，而
// tool-use 只允许审计、不进入计价输入（PRD 3.1）。Gemini 无标准缓存写入
// 字段，不虚构。
func buildGeminiBillingUsage(metadata *shared.GeminiUsageMetadata) (*BillingUsage, []string, error) {
	if metadata == nil {
		return nil, nil, fmt.Errorf("gemini usage metadata is nil")
	}
	bu := &BillingUsage{
		Source:                relayconstant.UsageSourceGemini,
		PromptAggregateTokens: metadata.PromptTokenCount,
		OutputTokens:          metadata.CandidatesTokenCount + metadata.ThoughtsTokenCount,
		CacheReadTokens:       metadata.CachedContentTokenCount,
	}
	for _, detail := range metadata.PromptTokensDetails {
		count := detail.TokenCount
		if count == 0 {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(detail.Modality)) {
		case "TEXT":
			bu.TextInputTokens = addModality(bu.TextInputTokens, count)
		case "AUDIO":
			bu.AudioInputTokens = addModality(bu.AudioInputTokens, count)
		case "IMAGE":
			bu.ImageInputTokens = addModality(bu.ImageInputTokens, count)
		case "VIDEO":
			bu.VideoInputTokens = addModality(bu.VideoInputTokens, count)
		case "DOCUMENT":
			bu.DocumentInputTokens = addModality(bu.DocumentInputTokens, count)
		}
	}
	for _, detail := range metadata.CandidatesTokensDetails {
		count := detail.TokenCount
		if count == 0 {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(detail.Modality)) {
		case "TEXT":
			bu.TextOutputTokens = addModality(bu.TextOutputTokens, count)
		case "AUDIO":
			bu.AudioOutputTokens = addModality(bu.AudioOutputTokens, count)
		case "IMAGE":
			bu.ImageOutputTokens = addModality(bu.ImageOutputTokens, count)
		}
	}
	if metadata.ThoughtsTokenCount != 0 {
		bu.ReasoningTokens = &metadata.ThoughtsTokenCount
	}
	if metadata.ToolUsePromptTokenCount != 0 {
		bu.ToolUsePromptTokens = &metadata.ToolUsePromptTokenCount
	}
	return finalizeBillingUsage(bu)
}

// finalizeBillingUsage 应用缓存写入转换规则、负数/分档校验与可诊断异常收集。
func finalizeBillingUsage(bu *BillingUsage) (*BillingUsage, []string, error) {
	nonNegative := map[string]int{
		"prompt_aggregate_tokens": bu.PromptAggregateTokens,
		"output_tokens":           bu.OutputTokens,
		"read_cache":              bu.CacheReadTokens,
		"write_cache":             bu.CacheWriteTokens,
	}
	for name, value := range nonNegative {
		if value < 0 {
			return nil, nil, fmt.Errorf("negative token count %s=%d", name, value)
		}
	}
	for name, value := range map[string]*int{
		"write_cache_5m":      bu.CacheWrite5mTokens,
		"write_cache_1h":      bu.CacheWrite1hTokens,
		"text_input":          bu.TextInputTokens,
		"image_input":         bu.ImageInputTokens,
		"audio_input":         bu.AudioInputTokens,
		"video_input":         bu.VideoInputTokens,
		"document_input":      bu.DocumentInputTokens,
		"text_output":         bu.TextOutputTokens,
		"audio_output":        bu.AudioOutputTokens,
		"image_output":        bu.ImageOutputTokens,
		"reasoning_output":    bu.ReasoningTokens,
		"accepted_prediction": bu.AcceptedPredictionTokens,
		"rejected_prediction": bu.RejectedPredictionTokens,
		"tool_use_prompt":     bu.ToolUsePromptTokens,
	} {
		if value != nil && *value < 0 {
			return nil, nil, fmt.Errorf("negative token count %s=%d", name, *value)
		}
	}

	// 官方分档存在时，校验其和不大于写入总量；否则是上游数据矛盾，显式失败。
	if bu.CacheWrite5mTokens != nil || bu.CacheWrite1hTokens != nil {
		officialTiered := intValue(bu.CacheWrite5mTokens) + intValue(bu.CacheWrite1hTokens)
		if officialTiered > bu.CacheWriteTokens {
			return nil, nil, fmt.Errorf("cache write tiers (%d) exceed write_cache total (%d)",
				officialTiered, bu.CacheWriteTokens)
		}
	}

	// 缓存写入转换规则：存在 write_cache 且官方未返回任何分档时，
	// 令 write_cache_5m = write_cache（未分档写入按 5 分钟档计价）。
	// 注意拷贝一份，避免 5m 指针与总量字段自别名。
	if bu.CacheWriteTokens > 0 && bu.CacheWrite5mTokens == nil && bu.CacheWrite1hTokens == nil {
		converted5m := bu.CacheWriteTokens
		bu.CacheWrite5mTokens = &converted5m
	}

	// 明细大于官方总量是可诊断异常：记录告警但不静默裁剪、不伪造空明细。
	var warnings []string
	if detail := intValue(bu.TextInputTokens) + intValue(bu.ImageInputTokens) + intValue(bu.AudioInputTokens) +
		intValue(bu.VideoInputTokens) + intValue(bu.DocumentInputTokens); detail > bu.PromptAggregateTokens && bu.PromptAggregateTokens > 0 {
		warnings = append(warnings, fmt.Sprintf("input modality details (%d) exceed prompt aggregate (%d)", detail, bu.PromptAggregateTokens))
	}
	if detail := intValue(bu.TextOutputTokens) + intValue(bu.AudioOutputTokens) + intValue(bu.ImageOutputTokens) +
		intValue(bu.ReasoningTokens) + intValue(bu.AcceptedPredictionTokens) + intValue(bu.RejectedPredictionTokens); detail > bu.OutputTokens && bu.OutputTokens > 0 {
		warnings = append(warnings, fmt.Sprintf("output splits (%d) exceed output total (%d)", detail, bu.OutputTokens))
	}
	if bu.CacheReadTokens > bu.PromptAggregateTokens && bu.PromptAggregateTokens > 0 {
		warnings = append(warnings, fmt.Sprintf("read_cache (%d) exceeds prompt aggregate (%d)", bu.CacheReadTokens, bu.PromptAggregateTokens))
	}
	if bu.CacheWriteTokens > bu.PromptAggregateTokens && bu.PromptAggregateTokens > 0 {
		warnings = append(warnings, fmt.Sprintf("write_cache (%d) exceeds prompt aggregate (%d)", bu.CacheWriteTokens, bu.PromptAggregateTokens))
	}
	return bu, warnings, nil
}

func fillInputModalities(bu *BillingUsage, promptDetails shared.InputTokenDetails, inputDetails *shared.InputTokenDetails) {
	// 明细非零即透传（含负数），负数由 finalizeBillingUsage 显式拒绝。
	if tokens := firstNonZero(promptDetails.TextTokens, inputDetailsTextTokens(inputDetails)); tokens != 0 {
		bu.TextInputTokens = &tokens
	}
	if tokens := firstNonZero(promptDetails.ImageTokens, inputDetailsImageTokens(inputDetails)); tokens != 0 {
		bu.ImageInputTokens = &tokens
	}
	if tokens := firstNonZero(promptDetails.AudioTokens, inputDetailsAudioTokens(inputDetails)); tokens != 0 {
		bu.AudioInputTokens = &tokens
	}
}

func fillOutputSplits(bu *BillingUsage, completionDetails shared.OutputTokenDetails, outputDetails *shared.OutputTokenDetails) {
	if tokens := firstNonZero(completionDetails.TextTokens, outputDetailsTextTokens(outputDetails)); tokens != 0 {
		bu.TextOutputTokens = &tokens
	}
	if tokens := firstNonZero(completionDetails.AudioTokens, outputDetailsAudioTokens(outputDetails)); tokens != 0 {
		bu.AudioOutputTokens = &tokens
	}
	if tokens := firstNonZero(completionDetails.ReasoningTokens, outputDetailsReasoningTokens(outputDetails)); tokens != 0 {
		bu.ReasoningTokens = &tokens
	}
	if tokens := firstNonZero(completionDetails.AcceptedPredictionTokens, outputDetailsAcceptedPredictionTokens(outputDetails)); tokens != 0 {
		bu.AcceptedPredictionTokens = &tokens
	}
	if tokens := firstNonZero(completionDetails.RejectedPredictionTokens, outputDetailsRejectedPredictionTokens(outputDetails)); tokens != 0 {
		bu.RejectedPredictionTokens = &tokens
	}
}

func addModality(existing *int, count int) *int {
	if existing == nil {
		return &count
	}
	merged := *existing + count
	return &merged
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func inputDetailsCachedTokens(details *shared.InputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.CachedTokens
}

func inputDetailsCachedCreationTokens(details *shared.InputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.CachedCreationTokens
}

func inputDetailsTextTokens(details *shared.InputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.TextTokens
}

func inputDetailsImageTokens(details *shared.InputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.ImageTokens
}

func inputDetailsAudioTokens(details *shared.InputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.AudioTokens
}

func outputDetailsTextTokens(details *shared.OutputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.TextTokens
}

func outputDetailsAudioTokens(details *shared.OutputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.AudioTokens
}

func outputDetailsReasoningTokens(details *shared.OutputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.ReasoningTokens
}

func outputDetailsAcceptedPredictionTokens(details *shared.OutputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.AcceptedPredictionTokens
}

func outputDetailsRejectedPredictionTokens(details *shared.OutputTokenDetails) int {
	if details == nil {
		return 0
	}
	return details.RejectedPredictionTokens
}

// BuildBillingDetailsForLog 把有 Token 用量的消费入口归一化并序列化为
// billing_details JSON（RecordConsumeLogParams.BillingDetails）。
// 返回空串表示不写该列：
//   - 本地估算/非 token 口径的伪 usage（ContextKeyLocalCountTokens）；
//   - 无 token 用量（总量与明细全零，如纯工具费、违规费入口）；
//   - 来源未标识、归一化或序列化失败——走 LogWarn/LogError 可诊断路径，
//     不伪造空拆分。
func BuildBillingDetailsForLog(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *shared.Usage) string {
	if usage == nil {
		return ""
	}
	if httpapi.GetContextKeyBool(ctx, common.ContextKeyLocalCountTokens) {
		return ""
	}
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 {
		return ""
	}
	if relayInfo == nil || relayInfo.UsageSource == relayconstant.UsageSourceNone {
		log.LogError(ctx, fmt.Sprintf("billing_details skipped: usage source not identified, model=%s", usageSourceModelName(relayInfo)))
		return ""
	}
	bu, warnings, err := BuildBillingUsage(relayInfo.UsageSource, usage, relayInfo.UsageGeminiMetadata)
	if err != nil {
		log.LogError(ctx, "billing_details normalization failed: "+err.Error())
		return ""
	}
	logBillingWarnings(ctx, warnings)
	return serializeForLog(ctx, bu)
}

// BuildRealtimeBillingDetailsForLog 同 BuildBillingDetailsForLog，用于
// Realtime/WSS 消费入口（usage 为 shared.RealtimeUsage）。
func BuildRealtimeBillingDetailsForLog(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *shared.RealtimeUsage) string {
	if usage == nil {
		return ""
	}
	if httpapi.GetContextKeyBool(ctx, common.ContextKeyLocalCountTokens) {
		return ""
	}
	if usage.InputTokens == 0 && usage.OutputTokens == 0 {
		return ""
	}
	if relayInfo == nil || relayInfo.UsageSource == relayconstant.UsageSourceNone {
		log.LogError(ctx, fmt.Sprintf("billing_details skipped: usage source not identified, model=%s", usageSourceModelName(relayInfo)))
		return ""
	}
	bu, warnings, err := BuildRealtimeBillingUsage(relayInfo.UsageSource, usage)
	if err != nil {
		log.LogError(ctx, "billing_details normalization failed: "+err.Error())
		return ""
	}
	logBillingWarnings(ctx, warnings)
	return serializeForLog(ctx, bu)
}

func serializeForLog(ctx *gin.Context, bu *BillingUsage) string {
	payload, err := SerializeBillingUsage(bu)
	if err != nil {
		log.LogError(ctx, "billing_details serialization failed: "+err.Error())
		return ""
	}
	return payload
}

func logBillingWarnings(ctx *gin.Context, warnings []string) {
	for _, warning := range warnings {
		log.LogWarn(ctx, "billing_details anomaly: "+warning)
	}
}

func usageSourceModelName(relayInfo *relaycommon.RelayInfo) string {
	if relayInfo == nil {
		return ""
	}
	return relayInfo.OriginModelName
}
