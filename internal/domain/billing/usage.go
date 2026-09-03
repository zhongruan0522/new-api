package billing

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/operation"
	billingcontract "github.com/NookMux/NookMux/internal/domain/billing/contract"
	domainchannel "github.com/NookMux/NookMux/internal/domain/channel"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/infra/log"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/internal/store/channel"
	"github.com/NookMux/NookMux/internal/store/log"
	"github.com/NookMux/NookMux/internal/store/user"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// UsageSettlement 承载 CalculateUsage 的计算结果，交由 ApplyQuota 落账。
// 字段不导出：调用方把它当作不透明句柄传回即可。
type UsageSettlement struct {
	modelName    string
	tokenName    string
	useTimeMs    int64
	quota        int
	extraContent []string

	promptTokens         int
	completionTokens     int
	cacheTokens          int
	cachedCreationTokens int
	cachedCreationRatio  float64
	audioTokens          int

	modelRatio      float64
	completionRatio float64
	cacheRatio      float64
	modelPrice      float64
	// groupRatio 是计价前快照；ApplyQuota 计算 originalGroupRatio 以它为基数，
	// 不得改为落账时从 PriceData 重读（计价/落账副作用之间不存在改写者，
	// 但快照语义与计价阶段使用的分组倍率保持同源）。
	groupRatio float64

	adminRejectReason string

	webSearchPrice           float64
	claudeWebSearchPrice     float64
	claudeWebSearchCallCount int
	geminiWebSearchPrice     float64
	geminiWebSearchCallCount int
	fileSearchPrice          float64
	imageGenerationCallPrice float64
	audioInputPrice          float64

	dWebSearchQuota           decimal.Decimal
	dClaudeWebSearchQuota     decimal.Decimal
	dGeminiWebSearchQuota     decimal.Decimal
	dFileSearchQuota          decimal.Decimal
	dImageGenerationCallQuota decimal.Decimal
	audioInputQuota           decimal.Decimal

	totalTokensZero bool
	hasToolFees     bool
	// estimatedUsage 表示 usage 来自本地估算（上游无计费信息）而非上游返回。
	estimatedUsage bool
	// bu 是本次请求的归一化 Token 用量（计费 PRD 阶段 2）：quota 与
	// billing_details JSON 都来自同一次转换，不再从聚合字段反推。
	bu *BillingUsage
	// billingDetailsJSON 由 CalculateUsage 内的同一个 bu 序列化得到；空串
	// 表示不写该列（本地估算、本地计数伪 usage 或无 token 用量）。
	billingDetailsJSON string
	// quotaLines 是归一化计价行项，供影子对拍差异定位与日志解释。
	quotaLines []BillingQuotaLine
	// priceSnapshot 保留实际结算价格依据；quotaLines 只有金额，不足以解释
	// 价格配置后续变化时的汇率、分组倍率与上下文档位。
	priceSnapshot *BillingPriceSnapshot
}

// normalizeUsageForBilling 是通用入口的归一化单点：真实上游 usage 必须携带
// 显式来源标识（relay 各解析点写入 RelayInfo.UsageSource），上游无 usage 的
// 本地估算与未打标的本地伪 usage 按聚合口径构造并复用同一计费公式。
// 归一化失败属于显式错误路径（PRD：计费配置缺失、归一化失败、上游无 usage
// 三类原因可观测），直接失败暴露问题，不回退旧公式掩盖。
func normalizeUsageForBilling(relayInfo *relaycommon.RelayInfo, usage *shared.Usage, localCountTokens bool) (*BillingUsage, []string, error) {
	// 上游无 token 用量（全零）：无可归一化语义，按聚合口径构造，
	// 后续按 totalTokens==0 的既有路径处理（原生请求触发重试），
	// 不判为归一化失败。
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 {
		bu, err := buildAggregateBillingUsage(usage)
		return bu, nil, err
	}
	// 本地估算/伪 usage（本地计数标记）：无官方语义可归一化（且可能缺少
	// Gemini 原始 metadata），按聚合口径构造，与旧通用公式口径一致；
	// billing_details 由调用方按本地计数标记跳过。audio/realtime 入口的
	// 伪 usage 携带 text/audio 明细，不走本分支（各入口直接归一化）。
	if localCountTokens {
		bu, err := buildAggregateBillingUsage(usage)
		return bu, nil, err
	}
	return BuildBillingUsage(relayInfo.UsageSource, usage, relayInfo.UsageGeminiMetadata)
}

// CalculateUsage 根据原始 usage 与计费上下文（RelayInfo.PriceData 等）计算应扣额度，
// 是通用文本路径 usage 计费的单点入口。rawUsage 为 nil 时按预估 prompt tokens 兜底。
// 返回错误时（归一化失败或上游缺失计费信息且不可重试豁免）调用方应直接中止，
// 不要调用 ApplyQuota。
func CalculateUsage(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, rawUsage *shared.Usage, extraContent ...string) (*UsageSettlement, *shared.NookMuxError) {
	usage := rawUsage
	settlementEstimatedUsage := false
	if usage == nil {
		usage = &shared.Usage{
			PromptTokens:     relayInfo.GetEstimatePromptTokens(),
			CompletionTokens: 0,
			TotalTokens:      relayInfo.GetEstimatePromptTokens(),
		}
		extraContent = append(extraContent, "上游无计费信息")
		settlementEstimatedUsage = true
	}

	if rawUsage != nil {
		domainchannel.ObserveChannelAffinityUsageCacheFromContext(ctx, usage)
	}

	adminRejectReason := httpapi.GetContextKeyString(ctx, common.ContextKeyAdminRejectReason)

	useTimeMs := time.Since(relayInfo.StartTime).Milliseconds()
	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens

	modelName := relayInfo.OriginModelName

	tokenName := ctx.GetString("token_name")
	groupRatio := relayInfo.PriceData.GroupRatioInfo.GroupRatio
	modelPrice := relayInfo.PriceData.ModelPrice
	localCountTokens := httpapi.GetContextKeyBool(ctx, common.ContextKeyLocalCountTokens)

	// 归一化（PRD 阶段 2）：同一次转换同时供计费与 billing_details 使用。
	bu, warnings, normErr := normalizeUsageForBilling(relayInfo, usage, localCountTokens)
	if normErr != nil {
		log.LogError(ctx, "billing normalization failed (cause=normalization_failed): "+normErr.Error())
		return nil, shared.NewOpenAIError(
			fmt.Errorf("%s", i18n.T(ctx, i18n.MsgQuotaBillingNormalizationFailed)),
			shared.ErrorCodeBadResponse, http.StatusBadGateway, shared.ErrOptionWithSkipRetry())
	}
	for _, warning := range warnings {
		log.LogWarn(ctx, "billing_details anomaly: "+warning)
	}

	// 上下文分段计费：档位匹配改用普通输入/输出/缓存读取/缓存写入四维
	// （PRD 阶段 2）；命中档位后的价格快照仍经 appendBillingInfo 写入现有位置。
	if _, enabled, err := ApplyContextPricingForBillingUsage(modelName, bu, &relayInfo.PriceData); enabled {
		if err != nil {
			log.LogError(ctx, "context pricing failed (cause=billing_config_missing): "+err.Error())
			extraContent = append(extraContent, "分段计费匹配失败: "+err.Error())
		}
	}

	quotaResult, quotaErr := CalculateNormalizedQuotaForRelay(bu, relayInfo.PriceData, AudioPricingAbsolute, modelName, relayInfo)
	if quotaErr != nil {
		log.LogError(ctx, "billing normalized quota failed (cause="+billingQuotaFailureCause(quotaErr)+"): "+quotaErr.Error())
		return nil, billingQuotaFailedError(ctx, quotaErr)
	}

	// 序列化失败不能伪装成“无明细的成功消费”：先显式失败，调用方保留预扣
	// 退款路径，避免 quota 已落账而 billing_details 缺失。
	var billingDetailsJSON string
	if !settlementEstimatedUsage && !localCountTokens &&
		!(promptTokens == 0 && completionTokens == 0) {
		payload, serializeErr := SerializeBillingUsage(bu)
		if serializeErr != nil {
			log.LogError(ctx, "billing_details serialization failed (cause=billing_details_serialization_failed): "+serializeErr.Error())
			return nil, normalizationFailedError(ctx)
		}
		billingDetailsJSON = payload
	}

	if !quotaResult.AudioInputQuota.IsZero() {
		extraContent = append(extraContent, fmt.Sprintf("Audio Input 花费 %s", quotaResult.AudioInputQuota.String()))
	}

	// Convert values to decimal for precise calculation
	dGroupRatio := decimal.NewFromFloat(groupRatio)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	dRatio := decimal.NewFromFloat(relayInfo.PriceData.ModelRatio).Mul(dGroupRatio)

	// openai web search 工具计费
	var dWebSearchQuota decimal.Decimal
	var webSearchPrice float64
	// response api 格式工具计费
	if relayInfo.ResponsesUsageInfo != nil {
		if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[shared.BuildInToolWebSearchPreview]; exists && webSearchTool.CallCount > 0 {
			// 优先使用可配置价格，回退到硬编码常量
			if pricePerCall, ok := operation.GetToolBillingPrice("web_search", map[string]string{"model": modelName, "provider": "openai"}); ok {
				webSearchPrice = pricePerCall
				dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
					Mul(decimal.NewFromInt(int64(webSearchTool.CallCount))).
					Mul(dGroupRatio).Mul(dQuotaPerUnit)
			} else {
				webSearchPrice = operation.GetWebSearchPricePerThousand(modelName, webSearchTool.SearchContextSize)
				dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
					Mul(decimal.NewFromInt(int64(webSearchTool.CallCount))).
					Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
			}
			extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 %d 次，上下文大小 %s，调用花费 %s",
				webSearchTool.CallCount, webSearchTool.SearchContextSize, dWebSearchQuota.String()))
		}
	} else if strings.HasSuffix(modelName, "search-preview") {
		// search-preview 模型不支持 response api
		searchContextSize := ctx.GetString("chat_completion_web_search_context_size")
		if searchContextSize == "" {
			searchContextSize = "medium"
		}
		if pricePerCall, ok := operation.GetToolBillingPrice("web_search", map[string]string{"model": modelName, "provider": "openai"}); ok {
			webSearchPrice = pricePerCall
			dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
				Mul(dGroupRatio).Mul(dQuotaPerUnit)
		} else {
			webSearchPrice = operation.GetWebSearchPricePerThousand(modelName, searchContextSize)
			dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
		}
		extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 1 次，上下文大小 %s，调用花费 %s",
			searchContextSize, dWebSearchQuota.String()))
	}
	// claude web search tool 计费
	var dClaudeWebSearchQuota decimal.Decimal
	var claudeWebSearchPrice float64
	claudeWebSearchCallCount := ctx.GetInt("claude_web_search_requests")
	if claudeWebSearchCallCount > 0 {
		if pricePerCall, ok := operation.GetToolBillingPrice("web_search", map[string]string{"model": modelName, "provider": "claude"}); ok {
			claudeWebSearchPrice = pricePerCall
			dClaudeWebSearchQuota = decimal.NewFromFloat(claudeWebSearchPrice).
				Mul(decimal.NewFromInt(int64(claudeWebSearchCallCount))).
				Mul(dGroupRatio).Mul(dQuotaPerUnit)
		} else {
			claudeWebSearchPrice = operation.GetClaudeWebSearchPricePerThousand()
			dClaudeWebSearchQuota = decimal.NewFromFloat(claudeWebSearchPrice).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit).Mul(decimal.NewFromInt(int64(claudeWebSearchCallCount)))
		}
		extraContent = append(extraContent, fmt.Sprintf("Claude Web Search 调用 %d 次，调用花费 %s",
			claudeWebSearchCallCount, dClaudeWebSearchQuota.String()))
	}
	// gemini web search tool 计费
	var dGeminiWebSearchQuota decimal.Decimal
	var geminiWebSearchPrice float64
	geminiWebSearchCallCount := ctx.GetInt("gemini_web_search_requests")
	if geminiWebSearchCallCount > 0 {
		if pricePerCall, ok := operation.GetToolBillingPrice("web_search", map[string]string{"model": modelName, "provider": "gemini"}); ok {
			geminiWebSearchPrice = pricePerCall
			dGeminiWebSearchQuota = decimal.NewFromFloat(geminiWebSearchPrice).
				Mul(decimal.NewFromInt(int64(geminiWebSearchCallCount))).
				Mul(dGroupRatio).Mul(dQuotaPerUnit)
		}
		if !dGeminiWebSearchQuota.IsZero() {
			extraContent = append(extraContent, fmt.Sprintf("Gemini Web Search 调用 %d 次，调用花费 %s",
				geminiWebSearchCallCount, dGeminiWebSearchQuota.String()))
		}
	}
	// file search tool 计费
	var dFileSearchQuota decimal.Decimal
	var fileSearchPrice float64
	if relayInfo.ResponsesUsageInfo != nil {
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[shared.BuildInToolFileSearch]; exists && fileSearchTool.CallCount > 0 {
			fileSearchPrice = operation.GetFileSearchPricePerThousand()
			dFileSearchQuota = decimal.NewFromFloat(fileSearchPrice).
				Mul(decimal.NewFromInt(int64(fileSearchTool.CallCount))).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
			extraContent = append(extraContent, fmt.Sprintf("File Search 调用 %d 次，调用花费 %s",
				fileSearchTool.CallCount, dFileSearchQuota.String()))
		}
	}
	var dImageGenerationCallQuota decimal.Decimal
	var imageGenerationCallPrice float64
	if ctx.GetBool("image_generation_call") {
		if pricePerCall, ok := operation.GetToolBillingPrice("image_generation", map[string]string{
			"model":    modelName,
			"provider": "openai",
			"quality":  ctx.GetString("image_generation_call_quality"),
			"size":     ctx.GetString("image_generation_call_size"),
		}); ok {
			imageGenerationCallPrice = pricePerCall
		} else {
			imageGenerationCallPrice = operation.GetGPTImage1PriceOnceCall(ctx.GetString("image_generation_call_quality"), ctx.GetString("image_generation_call_size"))
		}
		dImageGenerationCallQuota = decimal.NewFromFloat(imageGenerationCallPrice).Mul(dGroupRatio).Mul(dQuotaPerUnit)
		extraContent = append(extraContent, fmt.Sprintf("Image Generation Call 花费 %s", dImageGenerationCallQuota.String()))
	}

	// token 部分费用只来自归一化结果；工具费/图片调用费/音频独立费继续走独立路径。
	quotaCalculateDecimal := quotaResult.TokenTotal
	if !quotaResult.UsePrice && quotaResult.PriceSnapshot != nil &&
		quotaResult.PriceSnapshot.Source == billingcontract.PricePlanSourceLegacy &&
		!dRatio.IsZero() && quotaCalculateDecimal.LessThanOrEqual(decimal.Zero) {
		quotaCalculateDecimal = decimal.NewFromInt(1)
	}
	// 添加 responses tools call 调用的配额
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dWebSearchQuota)
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dClaudeWebSearchQuota)
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dGeminiWebSearchQuota)
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dFileSearchQuota)
	// 添加 audio input 独立计费
	quotaCalculateDecimal = quotaCalculateDecimal.Add(quotaResult.AudioInputQuota)
	// 添加 image generation call 计费
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dImageGenerationCallQuota)

	if len(relayInfo.PriceData.OtherRatios) > 0 {
		for key, otherRatio := range relayInfo.PriceData.OtherRatios {
			dOtherRatio := decimal.NewFromFloat(otherRatio)
			quotaCalculateDecimal = quotaCalculateDecimal.Mul(dOtherRatio)
			extraContent = append(extraContent, fmt.Sprintf("其他倍率 %s: %f", key, otherRatio))
		}
	}

	settlement := &UsageSettlement{
		modelName:        modelName,
		tokenName:        tokenName,
		useTimeMs:        useTimeMs,
		promptTokens:     promptTokens,
		completionTokens: completionTokens,
		// 计费快照字段从归一化结果取值（阶段 2 起缓存/模态口径以归一化为准）
		cacheTokens:          bu.CacheReadTokens,
		cachedCreationTokens: bu.CacheWriteTokens,
		cachedCreationRatio:  relayInfo.PriceData.CacheCreationRatio,
		audioTokens:          intValue(bu.AudioInputTokens),
		modelRatio:           relayInfo.PriceData.ModelRatio,
		completionRatio:      relayInfo.PriceData.CompletionRatio,
		cacheRatio:           relayInfo.PriceData.CacheRatio,
		modelPrice:           modelPrice,
		groupRatio:           groupRatio,
		adminRejectReason:    adminRejectReason,
		estimatedUsage:       settlementEstimatedUsage,
		bu:                   bu,
		billingDetailsJSON:   billingDetailsJSON,
		quotaLines:           quotaResult.Lines,
		priceSnapshot:        quotaResult.PriceSnapshot,

		webSearchPrice:           webSearchPrice,
		claudeWebSearchPrice:     claudeWebSearchPrice,
		claudeWebSearchCallCount: claudeWebSearchCallCount,
		geminiWebSearchPrice:     geminiWebSearchPrice,
		geminiWebSearchCallCount: geminiWebSearchCallCount,
		fileSearchPrice:          fileSearchPrice,
		imageGenerationCallPrice: imageGenerationCallPrice,
		audioInputPrice:          quotaResult.AudioInputPrice,

		dWebSearchQuota:           dWebSearchQuota,
		dClaudeWebSearchQuota:     dClaudeWebSearchQuota,
		dGeminiWebSearchQuota:     dGeminiWebSearchQuota,
		dFileSearchQuota:          dFileSearchQuota,
		dImageGenerationCallQuota: dImageGenerationCallQuota,
		audioInputQuota:           quotaResult.AudioInputQuota,

		extraContent: extraContent,
	}

	quota := RoundBillingQuota(quotaCalculateDecimal, quotaResult.RoundingMode)
	totalTokens := promptTokens + completionTokens

	// record all the consume log even if quota is 0
	// 未发生规范转换时，totalTokens == 0 代表上游缺失计费信息，需要交给外层重试。
	// 发生规范转换时，可能是转换导致的 token 统计异常，继续记录消费日志但不触发重试。
	if totalTokens == 0 {
		if apiErr := NewEmptyUsageRetryError(ctx, relayInfo); apiErr != nil {
			return nil, apiErr
		}
		settlement.totalTokensZero = true
		// 上游没有返回 token 信息（可能是超时或错误），但如果有工具调用费用，仍需扣费
		toolQuota := dWebSearchQuota.Add(dClaudeWebSearchQuota).Add(dGeminiWebSearchQuota).
			Add(dFileSearchQuota).Add(dImageGenerationCallQuota).Add(quotaResult.AudioInputQuota)
		if toolQuota.GreaterThan(decimal.Zero) {
			settlement.hasToolFees = true
			settlement.quota = int(toolQuota.Round(0).IntPart())
			settlement.extraContent = append(settlement.extraContent, "上游没有返回计费信息，但工具调用费用仍需扣除")
		} else {
			settlement.quota = 0
			settlement.extraContent = append(settlement.extraContent, "上游没有返回计费信息，无法扣费（可能是上游超时）")
		}
	} else {
		if !dRatio.IsZero() && quota == 0 && settlement.priceSnapshot != nil &&
			settlement.priceSnapshot.Source == billingcontract.PricePlanSourceLegacy {
			quota = 1
		}
		settlement.quota = quota
	}

	// 影子对拍（计费 PRD 阶段 2 迁移期）：旧公式并行计算，quota 不一致输出告警，
	// 差异必须定位为旧公式 bug 或新语义 bug，不允许吸收差异。
	if totalTokens > 0 {
		claudeSemantic := relayInfo.FinalRequestRelayFormat == relayconstant.RelayFormatClaude
		toolQuota := dWebSearchQuota.Add(dClaudeWebSearchQuota).Add(dGeminiWebSearchQuota).
			Add(dFileSearchQuota).Add(dImageGenerationCallQuota)
		legacyQuota := legacyGenericFinalQuota(usage, claudeSemantic, relayInfo.PriceData, modelName, toolQuota, relayInfo.PriceData.OtherRatios)
		if legacyQuota != settlement.quota {
			reportBillingShadowMismatch(ctx, "generic", relayInfo, legacyQuota, settlement.quota, bu, quotaResult.Lines, false)
		}
	}

	return settlement, nil
}

// ApplyQuota 将 CalculateUsage 的结果落到配额与账务：用户/渠道已用计数、计费会话结算、消费日志。
// 与 CalculateUsage 成对使用；返回值当前恒为 nil，保持与各 PostXxxConsumeQuota 相同的签名形状。
func ApplyQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, settlement *UsageSettlement) *shared.NookMuxError {
	// record all the consume log even if quota is 0
	logType := 0 // 0 表示使用默认的 LogTypeConsume
	if settlement.totalTokensZero {
		if settlement.hasToolFees {
			userstore.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, settlement.quota)
			channelstore.UpdateChannelUsedQuota(relayInfo.ChannelId, settlement.quota)
		}
		log.LogError(ctx, fmt.Sprintf("total tokens is 0, cannot consume quota, userId %d, channelId %d, "+
			"tokenId %d, model %s， pre-consumed quota %d", relayInfo.UserId, relayInfo.ChannelId, relayInfo.TokenId, settlement.modelName, relayInfo.FinalPreConsumedQuota))
	} else {
		userstore.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, settlement.quota)
		channelstore.UpdateChannelUsedQuota(relayInfo.ChannelId, settlement.quota)
	}

	if err := SettleBilling(ctx, relayInfo, settlement.quota); err != nil {
		log.LogError(ctx, "error settling billing: "+err.Error())
	}

	logModel := settlement.modelName
	if strings.HasPrefix(logModel, "gpt-4-gizmo") {
		logModel = "gpt-4-gizmo-*"
		settlement.extraContent = append(settlement.extraContent, fmt.Sprintf("模型 %s", settlement.modelName))
	}
	if strings.HasPrefix(logModel, "gpt-4o-gizmo") {
		logModel = "gpt-4o-gizmo-*"
		settlement.extraContent = append(settlement.extraContent, fmt.Sprintf("模型 %s", settlement.modelName))
	}
	logContent := strings.Join(settlement.extraContent, ", ")
	// 计算原始分组倍率（不含动态倍率），用于日志记录
	groupRatio := settlement.groupRatio
	originalGroupRatio := groupRatio
	dynamicRatio := relayInfo.PriceData.GroupRatioInfo.DynamicRatio
	if dynamicRatio > 0 {
		originalGroupRatio = groupRatio / dynamicRatio
	}
	other := GenerateTextOtherInfo(ctx, relayInfo, settlement.modelRatio, originalGroupRatio, settlement.completionRatio, settlement.cacheTokens, settlement.cacheRatio, settlement.modelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio, dynamicRatio)
	AppendBillingPriceSnapshot(other, &BillingQuotaResult{PriceSnapshot: settlement.priceSnapshot})
	if settlement.adminRejectReason != "" {
		other["reject_reason"] = settlement.adminRejectReason
	}
	if settlement.cachedCreationTokens != 0 {
		other["cache_creation_tokens"] = settlement.cachedCreationTokens
		other["cache_creation_ratio"] = settlement.cachedCreationRatio
	}
	if !settlement.dWebSearchQuota.IsZero() {
		if relayInfo.ResponsesUsageInfo != nil {
			if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[shared.BuildInToolWebSearchPreview]; exists {
				other["web_search"] = true
				other["web_search_call_count"] = webSearchTool.CallCount
				other["web_search_price"] = settlement.webSearchPrice
			}
		} else if strings.HasSuffix(settlement.modelName, "search-preview") {
			other["web_search"] = true
			other["web_search_call_count"] = 1
			other["web_search_price"] = settlement.webSearchPrice
		}
	} else if !settlement.dClaudeWebSearchQuota.IsZero() {
		other["web_search"] = true
		other["web_search_call_count"] = settlement.claudeWebSearchCallCount
		other["web_search_price"] = settlement.claudeWebSearchPrice
	} else if !settlement.dGeminiWebSearchQuota.IsZero() {
		other["web_search"] = true
		other["web_search_call_count"] = settlement.geminiWebSearchCallCount
		other["web_search_price"] = settlement.geminiWebSearchPrice
	}
	if !settlement.dFileSearchQuota.IsZero() && relayInfo.ResponsesUsageInfo != nil {
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[shared.BuildInToolFileSearch]; exists {
			other["file_search"] = true
			other["file_search_call_count"] = fileSearchTool.CallCount
			other["file_search_price"] = settlement.fileSearchPrice
		}
	}
	if !settlement.audioInputQuota.IsZero() {
		other["audio_input_seperate_price"] = true
		other["audio_input_token_count"] = settlement.audioTokens
		other["audio_input_price"] = settlement.audioInputPrice
	}
	if !settlement.dImageGenerationCallQuota.IsZero() {
		other["image_generation_call"] = true
		other["image_generation_call_price"] = settlement.imageGenerationCallPrice
	}
	// 共享流式日志指标，确保 OpenAI 兼容与 Claude 消费日志展示一致。
	AppendStreamMetrics(other, relayInfo, settlement.useTimeMs, settlement.completionTokens)
	logstore.RecordConsumeLog(ctx, relayInfo.UserId, logstore.RecordConsumeLogParams{
		ChannelId:        relayInfo.ChannelId,
		PromptTokens:     settlement.promptTokens,
		CompletionTokens: settlement.completionTokens,
		ModelName:        logModel,
		TokenName:        settlement.tokenName,
		Quota:            settlement.quota,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeMs:        int(settlement.useTimeMs),
		IsStream:         relayInfo.IsStream,
		Group:            relayInfo.UsingGroup,
		Other:            other,
		LogType:          logType,
		// 阶段 2：billing_details 由 CalculateUsage 内的同一归一化结果序列化。
		BillingDetails: settlement.billingDetailsJSON,
	})
	return nil
}
