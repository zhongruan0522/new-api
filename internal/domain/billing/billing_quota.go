package billing

import (
	"fmt"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/operation"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/shopspring/decimal"
)

// 计费 PRD 阶段 2：quota 计算切换到归一化 BillingUsage（PRD 第 3 章公式）。
// 本文件是四个计费入口（通用文本、Claude、audio、realtime/WSS）共享的
// 单点计价核心：计费输入一律来自同一次请求已归一化的 BillingUsage，
// 不再从聚合字段反推普通输入，也不再区分 isClaudeUsageSemantic 语义。
//
// 与旧公式的已知口径差异（由 billing_shadow.go 影子对拍暴露并分类，
// 不允许吸收差异）：
//   - 原生 Claude 的缓存读写不再随 raw 输入总量按基础价重复计费
//     （PRD 3.4：InputTokens 已不含缓存，缓存读写按缓存单价单列）；
//   - audio/realtime 路径的缓存读取按缓存单价计费（旧公式静默漏计）；
//   - OpenRouter 专属的 cost 反推缓存写入减法删除（PRD 阶段 2 指令）。

// AudioPricingMode 决定音频模态接入计费公式的价格维度。阶段 2 价格表仍为
// 倍率制，音频存在两种既有计价机制（阶段 3 组件价格表上线后收敛）。
type AudioPricingMode int

const (
	// AudioPricingRatioModel：audio / realtime(WSS) 入口。音频输入按
	// ModelRatio×AudioRatio、音频输出按 ModelRatio×AudioRatio×AudioCompletionRatio
	// 计价（与旧 calculateAudioQuota 口径一致；未配置时 AudioRatio 默认 1，
	// 即音频随普通输入/输出文本价计）。
	AudioPricingRatioModel AudioPricingMode = iota
	// AudioPricingAbsolute：通用文本入口。音频输入仅在配置了每百万 token
	// 美元单价时差异化计价（绝对价，不乘 ModelRatio，与旧通用公式一致）；
	// 音频输出没有差异化价格维度，随输出文本计价。
	AudioPricingAbsolute
)

// BillingQuotaLine 是单个计价行项，用于消费日志解释与影子对拍差异定位。
type BillingQuotaLine struct {
	Label  string
	Tokens int
	Quota  decimal.Decimal
}

// BillingQuotaResult 是归一化公式的未取整结果。
// TokenTotal 是参与最低消费规则的 token 部分费用；Absolute 模式下音频输入
// 独立费用单列在 AudioInputQuota（不参与 TokenTotal，由调用方按旧口径在
// 最低消费规则之后追加）。Lines 汇总全部行项（含 AudioInputQuota 对应行），
// 仅作解释用途。
type BillingQuotaResult struct {
	UsePrice   bool
	Lines      []BillingQuotaLine
	TokenTotal decimal.Decimal
	// AudioInputQuota 仅 Absolute 模式下非零：Gemini 每百万单价折算的音频
	// 输入独立费用（已含分组倍率与 QuotaPerUnit 折算，不乘 ModelRatio）。
	AudioInputQuota decimal.Decimal
	// AudioInputPrice 记录本次结算实际使用的每百万音频输入单价（0 表示未
	// 差异化计价），供计费快照 other["audio_input_price"] 写入。
	AudioInputPrice float64
}

// CalculateNormalizedQuota 按 PRD 3.4 节公式从归一化 BillingUsage 计算费用。
//
// 口径约定：
//   - 普通输入 = InputTokens()（raw 总量扣除缓存读取/写入），模态明细只在
//     存在差异化价格时移出（audio、image、video、document 只有价格表明确
//     差异化计价时才参与费用，否则不因明细存在而重复扣费）；
//   - 输出总量 = OutputTokens，reasoning 与 accepted/rejected prediction 是
//     输出审计拆分，包含在输出内按输出文本价计，不额外累加；
//   - 缓存写入 5m 档承担"官方 5m 档 + 未分档写入"（未分档按 5m 档计价）；
//   - Gemini toolUsePromptTokens 是审计字段，不进入任何计价行项。
func CalculateNormalizedQuota(bu *BillingUsage, priceData contract.PriceData, mode AudioPricingMode, modelName string) (BillingQuotaResult, error) {
	if bu == nil {
		return BillingQuotaResult{}, fmt.Errorf("billing usage is nil")
	}
	groupRatio := decimal.NewFromFloat(priceData.GroupRatioInfo.GroupRatio)
	quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	if priceData.UsePrice {
		return BillingQuotaResult{
			UsePrice:   true,
			TokenTotal: decimal.NewFromFloat(priceData.ModelPrice).Mul(quotaPerUnit).Mul(groupRatio),
		}, nil
	}

	modelRatio := decimal.NewFromFloat(priceData.ModelRatio)
	completionRatio := decimal.NewFromFloat(priceData.CompletionRatio)
	cacheRatio := decimal.NewFromFloat(priceData.CacheRatio)
	create5mRatio := decimal.NewFromFloat(priceData.CacheCreation5mRatio)
	create1hRatio := decimal.NewFromFloat(priceData.CacheCreation1hRatio)
	ratio := modelRatio.Mul(groupRatio)

	inputTokens := bu.InputTokens()
	outputTokens := bu.OutputTokens
	audioInputTokens := intValue(bu.AudioInputTokens)
	audioOutputTokens := intValue(bu.AudioOutputTokens)

	result := BillingQuotaResult{TokenTotal: decimal.Zero}
	addTokenLine := func(label string, tokens int, quota decimal.Decimal) {
		result.Lines = append(result.Lines, BillingQuotaLine{Label: label, Tokens: tokens, Quota: quota})
		result.TokenTotal = result.TokenTotal.Add(quota)
	}

	switch mode {
	case AudioPricingAbsolute:
		// 每百万美元单价是旧通用公式唯一识别的音频输入差异化价格；
		// 音频输出在该模式下随输出文本计价（旧口径）。
		perMillionPrice := operation.GetGeminiInputAudioPricePerMillionTokens(modelName)
		if perMillionPrice > 0 && audioInputTokens != 0 {
			inputTokens -= audioInputTokens
			quota := decimal.NewFromFloat(perMillionPrice).
				Div(decimal.NewFromInt(1000000)).
				Mul(decimal.NewFromInt(int64(audioInputTokens))).
				Mul(groupRatio).Mul(quotaPerUnit)
			result.AudioInputQuota = quota
			result.AudioInputPrice = perMillionPrice
			result.Lines = append(result.Lines, BillingQuotaLine{Label: "音频输入（独立单价）", Tokens: audioInputTokens, Quota: quota})
		}
	case AudioPricingRatioModel:
		if audioInputTokens != 0 {
			inputTokens -= audioInputTokens
			addTokenLine("音频输入", audioInputTokens,
				decimal.NewFromInt(int64(audioInputTokens)).Mul(decimal.NewFromFloat(priceData.AudioRatio)).Mul(ratio))
		}
		if audioOutputTokens != 0 {
			outputTokens -= audioOutputTokens
			addTokenLine("音频输出", audioOutputTokens,
				decimal.NewFromInt(int64(audioOutputTokens)).Mul(decimal.NewFromFloat(priceData.AudioRatio)).Mul(decimal.NewFromFloat(priceData.AudioCompletionRatio)).Mul(ratio))
		}
	default:
		return BillingQuotaResult{}, fmt.Errorf("unknown audio pricing mode: %d", mode)
	}

	addTokenLine("普通输入", inputTokens, decimal.NewFromInt(int64(inputTokens)).Mul(ratio))
	addTokenLine("输出文本", outputTokens, decimal.NewFromInt(int64(outputTokens)).Mul(completionRatio).Mul(ratio))
	if bu.CacheReadTokens != 0 {
		addTokenLine("缓存读取", bu.CacheReadTokens,
			decimal.NewFromInt(int64(bu.CacheReadTokens)).Mul(cacheRatio).Mul(ratio))
	}
	// 未分档写入经归一化转换规则并入 5m 档：write5m = 5m 档 + (write - 5m - 1h)。
	write5mTokens := bu.CacheWriteTokens - intValue(bu.CacheWrite1hTokens)
	if write5mTokens != 0 {
		addTokenLine("缓存写入(5m/未分档)", write5mTokens,
			decimal.NewFromInt(int64(write5mTokens)).Mul(create5mRatio).Mul(ratio))
	}
	if write1hTokens := intValue(bu.CacheWrite1hTokens); write1hTokens != 0 {
		addTokenLine("缓存写入(1h)", write1hTokens,
			decimal.NewFromInt(int64(write1hTokens)).Mul(create1hRatio).Mul(ratio))
	}

	return result, nil
}

// buildAggregateBillingUsage 把本地估算/本地计数的聚合 usage 构造为
// BillingUsage，让上游无 usage 兜底与本地伪 token 用量（按张数充数、
// 字符数计费等）复用同一条归一化计费公式。这类 usage 没有官方缓存/模态
// 语义可归一化，按聚合口径构造（缓存读写从 raw 总量扣除），与旧通用公式
// 口径一致；billing_details 仍由调用方按估算/本地计数标记跳过。
func buildAggregateBillingUsage(usage *shared.Usage) (*BillingUsage, error) {
	if usage == nil {
		return nil, fmt.Errorf("aggregate usage is nil")
	}
	bu := &BillingUsage{
		PromptAggregateTokens: usage.PromptTokens,
		OutputTokens:          usage.CompletionTokens,
		CacheReadTokens:       usage.PromptTokensDetails.CachedTokens,
		CacheWriteTokens:      usage.PromptTokensDetails.CachedCreationTokens,
	}
	if usage.PromptTokensDetails.AudioTokens != 0 {
		audio := usage.PromptTokensDetails.AudioTokens
		bu.AudioInputTokens = &audio
	}
	finalized, _, err := finalizeBillingUsage(bu)
	if err != nil {
		return nil, fmt.Errorf("aggregate billing usage invalid: %w", err)
	}
	return finalized, nil
}
