package billing

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/operation"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	pricingstore "github.com/NookMux/NookMux/internal/store/pricing"
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

// AudioPricingMode 决定音频模态接入计费公式的价格维度。阶段 4 起显式价格表
// 可直接给出音频组件价；两种既有机制保留为旧 ratio 投影的回退口径。
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

// ErrBillingPriceConfigMissing separates malformed/missing settlement pricing
// from invalid upstream usage normalization. Callers must preserve this cause
// so operators can distinguish configuration failures from provider data.
var ErrBillingPriceConfigMissing = errors.New("billing price configuration missing")

// BillingQuotaLine 是单个计价行项，用于消费日志解释与影子对拍差异定位。
type BillingQuotaLine struct {
	Label  string
	Tokens int
	Quota  decimal.Decimal
}

type BillingPriceComponentSnapshot struct {
	Component       contract.PriceComponent  `json:"component"`
	Unit            contract.PriceUnit       `json:"unit"`
	Quantity        int                      `json:"quantity"`
	UnitPrice       string                   `json:"unit_price"`
	Currency        string                   `json:"currency"`
	ExchangeRate    string                   `json:"exchange_rate"`
	GroupMultiplier string                   `json:"group_multiplier"`
	PlanSource      contract.PricePlanSource `json:"plan_source"`
	PlanID          int64                    `json:"plan_id,omitempty"`
}

type BillingPriceSnapshot struct {
	Source                contract.PricePlanSource        `json:"source"`
	BillingMode           contract.BillingMode            `json:"billing_mode"`
	Endpoint              string                          `json:"endpoint"`
	ServiceTier           string                          `json:"service_tier"`
	ContextTokens         int                             `json:"context_tokens"`
	ContextMinTokens      int                             `json:"context_min_tokens"`
	ContextMaxTokens      *int                            `json:"context_max_tokens,omitempty"`
	GroupMultiplierSource contract.GroupMultiplierSource  `json:"group_multiplier_source"`
	GroupMultiplier       string                          `json:"group_multiplier"`
	RoundingMode          contract.PriceRoundingMode      `json:"rounding_mode"`
	PricePrecision        int                             `json:"price_precision"`
	Components            []BillingPriceComponentSnapshot `json:"components"`
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
	// RoundingMode 来自生效计划；旧 ratio 投影由各入口 roundEntryQuota 保持
	// 既有取整口径（通用/audio/realtime half-up，Claude 截断）。
	RoundingMode contract.PriceRoundingMode
	// PriceSnapshot 记录实际价格依据，写入 Log.Other 而不是 billing_details。
	PriceSnapshot *BillingPriceSnapshot
}

func CalculateNormalizedQuotaForRelay(
	bu *BillingUsage,
	priceData contract.PriceData,
	mode AudioPricingMode,
	modelName string,
	relayInfo *relaycommon.RelayInfo,
) (BillingQuotaResult, error) {
	if bu == nil {
		return BillingQuotaResult{}, fmt.Errorf("billing usage is nil")
	}
	if relayInfo == nil {
		return BillingQuotaResult{}, fmt.Errorf("relay info is nil")
	}
	explicitPlans, err := pricingstore.GetModelPricePlans()
	if err != nil {
		// 价格表加载失败意味着结算无法看到任何计价配置，归因为
		// billing_config_missing 而不是 normalization_failed（PRD 三类
		// 可观测原因），调用方的失败日志与对账排查按配置问题分组。
		return BillingQuotaResult{}, fmt.Errorf("%w: load explicit model price plans: %v", ErrBillingPriceConfigMissing, err)
	}
	plans := append(explicitPlans, legacyPricePlansForRelay(modelName, priceData)...)
	query := contract.ModelPricePlanQuery{
		ModelName:      modelName,
		Endpoint:       relayInfo.RequestURLPath,
		EffectiveGroup: relayInfo.UsingGroup,
		ServiceTier:    relayInfo.ServiceTierEffective,
		ContextTokens:  ContextTokensForTier(bu),
		EffectiveAt:    common.GetTimestamp(),
	}
	return calculateNormalizedQuotaWithPlans(bu, priceData, mode, plans, query)
}

func legacyPricePlansForRelay(modelName string, priceData contract.PriceData) []contract.ModelPricePlan {
	input := contract.LegacyPriceInput{
		ModelName:            modelName,
		HasModelRatio:        !priceData.UsePrice,
		ModelRatio:           priceData.ModelRatio,
		CompletionRatio:      priceData.CompletionRatio,
		CacheRatio:           priceData.CacheRatio,
		CacheCreationRatio:   priceData.CacheCreation5mRatio,
		CacheCreation1hRatio: priceData.CacheCreation1hRatio,
		AudioRatio:           priceData.AudioRatio,
		AudioCompletionRatio: priceData.AudioCompletionRatio,
		QuotaPerUnit:         common.QuotaPerUnit,
	}
	if priceData.UsePrice {
		modelPrice := priceData.ModelPrice
		input.ModelPrice = &modelPrice
	}
	if result := priceData.ContextPricing; result != nil && result.Enabled {
		input.ContextPricing = &contract.ContextPricingConfig{
			Enabled: true,
			Tiers: []contract.ContextPricingTier{{
				MinTokens:            result.MinTokens,
				MaxTokens:            result.MaxTokens,
				ModelRatio:           priceData.ModelRatio,
				CompletionRatio:      priceData.CompletionRatio,
				CacheRatio:           priceData.CacheRatio,
				CreateCacheRatio:     priceData.CacheCreation5mRatio,
				AudioRatio:           priceData.AudioRatio,
				AudioCompletionRatio: priceData.AudioCompletionRatio,
			}},
		}
	}
	return contract.LegacyPricePlans(input)
}

// calculateNormalizedQuota 仅用于测试的旧配置直算包装；生产结算必须走
// CalculateNormalizedQuotaForRelay（价格表与旧配置的优先级、分组倍率与
// service tier 匹配都在单点内固定）。不导出以防止调用方绕过价格表。
func calculateNormalizedQuota(bu *BillingUsage, priceData contract.PriceData, mode AudioPricingMode, modelName string) (BillingQuotaResult, error) {
	return calculateNormalizedQuotaWithPlans(
		bu, priceData, mode,
		legacyPricePlansForRelay(modelName, priceData),
		contract.ModelPricePlanQuery{ModelName: modelName},
	)
}

func calculateNormalizedQuotaWithPlans(
	bu *BillingUsage,
	priceData contract.PriceData,
	mode AudioPricingMode,
	plans []contract.ModelPricePlan,
	query contract.ModelPricePlanQuery,
) (BillingQuotaResult, error) {
	if bu == nil {
		return BillingQuotaResult{}, fmt.Errorf("billing usage is nil")
	}
	if mode != AudioPricingAbsolute && mode != AudioPricingRatioModel {
		return BillingQuotaResult{}, fmt.Errorf("unknown audio pricing mode: %d", mode)
	}
	effectivePlan, hasPlan := ResolveModelPricePlan(plans, query)
	if !hasPlan {
		return BillingQuotaResult{}, fmt.Errorf("%w: model price plan is missing", ErrBillingPriceConfigMissing)
	}
	plan := *effectivePlan
	if normalizedPricePlanSource(plan.Source) != contract.PricePlanSourceExplicit {
		switch plan.BillingMode {
		case contract.BillingModePerRequest:
			return perRequestQuotaWithSnapshot(plans, query, plan, priceData)
		case contract.BillingModeFree:
			return freeQuotaWithSnapshot(plan, query, priceData), nil
		case contract.BillingModeToken:
			return legacyQuotaWithSnapshot(bu, priceData, mode, plan, query)
		default:
			return BillingQuotaResult{}, fmt.Errorf("%w: billing mode is invalid", ErrBillingPriceConfigMissing)
		}
	}

	switch effectivePlan.BillingMode {
	case contract.BillingModePerRequest:
		return perRequestQuotaWithSnapshot(plans, query, plan, priceData)
	case contract.BillingModeFree:
		return freeQuotaWithSnapshot(plan, query, priceData), nil
	case contract.BillingModeToken:
		return explicitTokenQuotaWithSnapshot(bu, plans, query, plan, priceData, mode)
	default:
		return BillingQuotaResult{}, fmt.Errorf("%w: billing mode is invalid", ErrBillingPriceConfigMissing)
	}
}

func legacyQuotaWithSnapshot(
	bu *BillingUsage,
	priceData contract.PriceData,
	mode AudioPricingMode,
	plan contract.ModelPricePlan,
	query contract.ModelPricePlanQuery,
) (BillingQuotaResult, error) {
	// Absolute legacy audio pricing is the one pre-component fee whose unit
	// price lives outside the ratio projection. Keep its exact old path.
	if mode == AudioPricingAbsolute && intValue(bu.AudioInputTokens) != 0 &&
		operation.GetGeminiInputAudioPricePerMillionTokens(query.ModelName) > 0 {
		result, err := legacyNormalizedQuota(bu, priceData, mode, query.ModelName)
		if err != nil {
			return BillingQuotaResult{}, err
		}
		groupMultiplier := decimal.NewFromFloat(priceData.GroupRatioInfo.GroupRatio)
		result.RoundingMode = contract.PriceRoundingHalfUp
		result.PriceSnapshot = newBillingPriceSnapshot(plan, query, plan.GroupMultiplierSource, groupMultiplier, result.RoundingMode, plan.PricePrecision)
		appendLegacyComponentSnapshots(result.PriceSnapshot, plan, groupMultiplier, bu, result.AudioInputPrice)
		perMillionPrice := decimal.NewFromFloat(result.AudioInputPrice)
		result.PriceSnapshot.Components = append(result.PriceSnapshot.Components, BillingPriceComponentSnapshot{
			Component:       contract.PriceComponentAudioInput,
			Unit:            contract.PriceUnitPerMillionTokens,
			Quantity:        intValue(bu.AudioInputTokens),
			UnitPrice:       perMillionPrice.String(),
			Currency:        plan.Currency,
			ExchangeRate:    plan.ExchangeRate,
			GroupMultiplier: groupMultiplier.String(),
			PlanSource:      contract.PricePlanSourceLegacy,
		})
		return result, nil
	}
	return explicitTokenQuotaWithSnapshot(bu, []contract.ModelPricePlan{plan}, query, plan, priceData, mode)
}

func appendLegacyComponentSnapshots(
	snapshot *BillingPriceSnapshot,
	plan contract.ModelPricePlan,
	groupMultiplier decimal.Decimal,
	bu *BillingUsage,
	audioInputPrice float64,
) {
	absoluteAudioInput := audioInputPrice > 0
	inputTokens := bu.InputTokens()
	outputTokens := bu.OutputTokens
	if absoluteAudioInput {
		inputTokens -= intValue(bu.AudioInputTokens)
	}
	tokenCounts := map[contract.PriceComponent]int{
		contract.PriceComponentInput:         inputTokens,
		contract.PriceComponentOutput:        outputTokens,
		contract.PriceComponentCacheRead:     bu.CacheReadTokens,
		contract.PriceComponentCacheWrite5m:  bu.CacheWriteTokens - intValue(bu.CacheWrite1hTokens),
		contract.PriceComponentCacheWrite1h:  intValue(bu.CacheWrite1hTokens),
		contract.PriceComponentAudioInput:    intValue(bu.AudioInputTokens),
		contract.PriceComponentAudioOutput:   intValue(bu.AudioOutputTokens),
		contract.PriceComponentTextInput:     intValue(bu.TextInputTokens),
		contract.PriceComponentTextOutput:    intValue(bu.TextOutputTokens),
		contract.PriceComponentImageInput:    intValue(bu.ImageInputTokens),
		contract.PriceComponentImageOutput:   intValue(bu.ImageOutputTokens),
		contract.PriceComponentVideoInput:    intValue(bu.VideoInputTokens),
		contract.PriceComponentDocumentInput: intValue(bu.DocumentInputTokens),
	}
	for _, component := range plan.Components {
		tokens := tokenCounts[component.Component]
		switch component.Component {
		case contract.PriceComponentAudioInput:
			if absoluteAudioInput {
				// Absolute legacy audio input is recorded separately below with
				// its operation-level unit price.
				continue
			}
		case contract.PriceComponentTextInput:
			if absoluteAudioInput {
				tokenCounts[component.Component] = inputTokens
				tokens = inputTokens
			}
		case contract.PriceComponentTextOutput, contract.PriceComponentAudioOutput:
			// The special Absolute path intentionally charges all output at the
			// generic output price. Projected children cannot explain that
			// settlement, so the generic output snapshot above remains
			// authoritative even when audio completion ratios caused a split.
			if component.Component == contract.PriceComponentTextOutput && absoluteAudioInput {
				tokenCounts[component.Component] = outputTokens
				tokens = outputTokens
			} else {
				continue
			}
		}
		if tokens == 0 {
			continue
		}
		snapshot.Components = append(snapshot.Components, BillingPriceComponentSnapshot{
			Component:       component.Component,
			Unit:            component.Unit,
			Quantity:        tokens,
			UnitPrice:       component.UnitPrice,
			Currency:        plan.Currency,
			ExchangeRate:    plan.ExchangeRate,
			GroupMultiplier: groupMultiplier.String(),
			PlanSource:      contract.PricePlanSourceLegacy,
		})
	}
}

func perRequestQuotaWithSnapshot(
	plans []contract.ModelPricePlan,
	query contract.ModelPricePlanQuery,
	plan contract.ModelPricePlan,
	priceData contract.PriceData,
) (BillingQuotaResult, error) {
	resolved := &contract.ResolvedModelPriceComponent{Plan: plan, PlanSource: plan.Source, BillingMode: plan.BillingMode}
	found := false
	for _, component := range plan.Components {
		if component.Component == contract.PriceComponentRequest {
			resolved.Component = component
			found = true
			break
		}
	}
	if !found {
		return BillingQuotaResult{}, fmt.Errorf("%w: request price component is missing", ErrBillingPriceConfigMissing)
	}
	unitPrice, exchangeRate, err := parseComponentDecimals(resolved.Component, resolved.Plan)
	if err != nil {
		return BillingQuotaResult{}, err
	}
	groupMultiplier, err := effectiveGroupMultiplier(plan, priceData)
	if err != nil {
		return BillingQuotaResult{}, err
	}
	quota := unitPrice.Mul(exchangeRate).Mul(groupMultiplier).Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	result := BillingQuotaResult{
		UsePrice:     true,
		TokenTotal:   quota,
		RoundingMode: plan.RoundingMode,
	}
	result.PriceSnapshot = newBillingPriceSnapshot(plan, query, plan.GroupMultiplierSource, groupMultiplier, result.RoundingMode, plan.PricePrecision)
	result.PriceSnapshot.Components = append(result.PriceSnapshot.Components, newPriceComponentSnapshot(resolved, groupMultiplier, 1))
	return result, nil
}

func freeQuotaWithSnapshot(plan contract.ModelPricePlan, query contract.ModelPricePlanQuery, priceData contract.PriceData) BillingQuotaResult {
	groupMultiplier := decimal.NewFromFloat(priceData.GroupRatioInfo.GroupRatio)
	if plan.GroupMultiplierSource == contract.GroupMultiplierSourceFixed {
		groupMultiplier = decimal.RequireFromString(plan.GroupMultiplier)
	}
	result := BillingQuotaResult{RoundingMode: plan.RoundingMode}
	result.PriceSnapshot = newBillingPriceSnapshot(plan, query, plan.GroupMultiplierSource, groupMultiplier, plan.RoundingMode, plan.PricePrecision)
	return result
}

func explicitTokenQuotaWithSnapshot(
	bu *BillingUsage,
	plans []contract.ModelPricePlan,
	query contract.ModelPricePlanQuery,
	plan contract.ModelPricePlan,
	priceData contract.PriceData,
	mode AudioPricingMode,
) (BillingQuotaResult, error) {
	groupMultiplier, err := effectiveGroupMultiplier(plan, priceData)
	if err != nil {
		return BillingQuotaResult{}, err
	}
	inheritedGroupMultiplier := decimal.NewFromFloat(priceData.GroupRatioInfo.GroupRatio)
	result := BillingQuotaResult{
		Lines:        []BillingQuotaLine{},
		TokenTotal:   decimal.Zero,
		RoundingMode: plan.RoundingMode,
	}
	snapshot := newBillingPriceSnapshot(plan, query, plan.GroupMultiplierSource, groupMultiplier, result.RoundingMode, plan.PricePrecision)
	addLine := func(label string, tokens int, resolved *contract.ResolvedModelPriceComponent) error {
		if tokens == 0 {
			return nil
		}
		quota, err := resolvedTokenQuota(resolved, tokens, groupMultiplier)
		if err != nil {
			return err
		}
		componentMultiplier, multiplierErr := componentGroupMultiplier(resolved, inheritedGroupMultiplier)
		if multiplierErr != nil {
			return multiplierErr
		}
		result.Lines = append(result.Lines, BillingQuotaLine{Label: label, Tokens: tokens, Quota: quota})
		result.TokenTotal = result.TokenTotal.Add(quota)
		snapshot.Components = append(snapshot.Components, newPriceComponentSnapshot(resolved, componentMultiplier, tokens))
		return nil
	}

	inputTokens := bu.InputTokens()
	outputTokens := bu.OutputTokens
	audioInputTokens := intValue(bu.AudioInputTokens)
	audioOutputTokens := intValue(bu.AudioOutputTokens)
	inputParent := contract.PriceComponentInput
	outputParent := contract.PriceComponentOutput
	if anyMatchingPlanHasChild(plans, query, contract.InputChildPriceComponents) {
		inputParent = contract.PriceComponentTextInput
	}
	if anyMatchingPlanHasChild(plans, query, contract.OutputChildPriceComponents) {
		outputParent = contract.PriceComponentTextOutput
	}

	if audioInputTokens != 0 {
		resolved, ok := directMatchingComponent(plans, query, contract.PriceComponentAudioInput)
		// Legacy audio-ratio children only apply on the relative audio path;
		// Absolute mode either used its dedicated legacy price above or treats
		// audio as ordinary input.
		if ok && (mode == AudioPricingRatioModel ||
			normalizedPricePlanSource(resolved.Plan.Source) == contract.PricePlanSourceExplicit) {
			quota, quotaErr := resolvedTokenQuota(resolved, audioInputTokens, groupMultiplier)
			if quotaErr != nil {
				return BillingQuotaResult{}, quotaErr
			}
			audioMultiplier, multiplierErr := componentGroupMultiplier(resolved, inheritedGroupMultiplier)
			if multiplierErr != nil {
				return BillingQuotaResult{}, multiplierErr
			}
			if mode == AudioPricingRatioModel {
				result.Lines = append(result.Lines, BillingQuotaLine{Label: "音频输入", Tokens: audioInputTokens, Quota: quota})
				result.TokenTotal = result.TokenTotal.Add(quota)
			} else {
				result.AudioInputQuota = quota
				result.AudioInputPrice = unitPriceFloat(resolved.Component.UnitPrice)
			}
			snapshot.Components = append(snapshot.Components, newPriceComponentSnapshot(resolved, audioMultiplier, audioInputTokens))
			inputTokens -= audioInputTokens
		}
	}
	if audioOutputTokens != 0 {
		if resolved, ok := directMatchingComponent(plans, query, contract.PriceComponentAudioOutput); ok {
			// Legacy ratio children are relative-mode dimensions. In Absolute
			// mode their price is the ordinary output price, so subtracting and
			// re-adding them separately is unnecessary and can shift rounding.
			if mode == AudioPricingRatioModel ||
				normalizedPricePlanSource(resolved.Plan.Source) == contract.PricePlanSourceExplicit {
				if err := addLine("音频输出", audioOutputTokens, resolved); err != nil {
					return BillingQuotaResult{}, err
				}
				outputTokens -= audioOutputTokens
			}
		}
	}
	for _, component := range []contract.PriceComponent{
		contract.PriceComponentImageInput,
		contract.PriceComponentVideoInput,
		contract.PriceComponentDocumentInput,
	} {
		if resolved, ok := directMatchingComponent(plans, query, component); ok {
			tokens := inputChildTokens(bu, component)
			if err := addLine(componentLabel(component), tokens, resolved); err != nil {
				return BillingQuotaResult{}, err
			}
			inputTokens -= tokens
		}
	}
	if resolved, ok := directMatchingComponent(plans, query, contract.PriceComponentImageOutput); ok {
		tokens := intValue(bu.ImageOutputTokens)
		if err := addLine("图像输出", tokens, resolved); err != nil {
			return BillingQuotaResult{}, err
		}
		outputTokens -= tokens
	}

	inputResolved, ok := ResolveModelPriceComponent(plans, query, inputParent)
	if !ok {
		return BillingQuotaResult{}, fmt.Errorf("%w: %s price component is missing", ErrBillingPriceConfigMissing, inputParent)
	}
	if err := addLine("普通输入", inputTokens, inputResolved); err != nil {
		return BillingQuotaResult{}, err
	}
	outputResolved, ok := ResolveModelPriceComponent(plans, query, outputParent)
	if !ok {
		return BillingQuotaResult{}, fmt.Errorf("%w: %s price component is missing", ErrBillingPriceConfigMissing, outputParent)
	}
	if err := addLine("输出文本", outputTokens, outputResolved); err != nil {
		return BillingQuotaResult{}, err
	}
	for _, item := range []struct {
		component contract.PriceComponent
		tokens    int
		label     string
	}{
		{contract.PriceComponentCacheRead, bu.CacheReadTokens, "缓存读取"},
		{contract.PriceComponentCacheWrite5m, bu.CacheWriteTokens - intValue(bu.CacheWrite1hTokens), "缓存写入(5m/未分档)"},
		{contract.PriceComponentCacheWrite1h, intValue(bu.CacheWrite1hTokens), "缓存写入(1h)"},
	} {
		resolved, ok := ResolveModelPriceComponent(plans, query, item.component)
		if !ok {
			return BillingQuotaResult{}, fmt.Errorf("%w: %s price component is missing", ErrBillingPriceConfigMissing, item.component)
		}
		if err := addLine(item.label, item.tokens, resolved); err != nil {
			return BillingQuotaResult{}, err
		}
	}
	result.PriceSnapshot = snapshot
	return result, nil
}

func effectiveGroupMultiplier(plan contract.ModelPricePlan, priceData contract.PriceData) (decimal.Decimal, error) {
	if plan.GroupMultiplierSource != contract.GroupMultiplierSourceFixed {
		return decimal.NewFromFloat(priceData.GroupRatioInfo.GroupRatio), nil
	}
	value, err := decimal.NewFromString(plan.GroupMultiplier)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse effective group multiplier: %w", err)
	}
	return value, nil
}

func resolvedTokenQuota(resolved *contract.ResolvedModelPriceComponent, tokens int, groupMultiplier decimal.Decimal) (decimal.Decimal, error) {
	unitPrice, exchangeRate, err := parseComponentDecimals(resolved.Component, resolved.Plan)
	if err != nil {
		return decimal.Zero, err
	}
	componentMultiplier, err := componentGroupMultiplier(resolved, groupMultiplier)
	if err != nil {
		return decimal.Zero, err
	}
	return tokenComponentQuota(tokens, unitPrice, exchangeRate, componentMultiplier, decimal.NewFromFloat(common.QuotaPerUnit)), nil
}

func componentGroupMultiplier(resolved *contract.ResolvedModelPriceComponent, inheritedGroupMultiplier decimal.Decimal) (decimal.Decimal, error) {
	if resolved.Plan.GroupMultiplierSource != contract.GroupMultiplierSourceFixed {
		return inheritedGroupMultiplier, nil
	}
	value, err := decimal.NewFromString(resolved.Plan.GroupMultiplier)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse component group multiplier: %w", err)
	}
	return value, nil
}

func directMatchingComponent(plans []contract.ModelPricePlan, query contract.ModelPricePlanQuery, component contract.PriceComponent) (*contract.ResolvedModelPriceComponent, bool) {
	// 子组件与计划解析遵循同一固定优先级（modelPricePlanPrecedes）：多个
	// 匹配计划都配置该组件时按特异性取价，不随持久化顺序漂移。
	candidates := matchingModelPricePlans(plans, query, true)
	sort.SliceStable(candidates, func(i, j int) bool {
		return modelPricePlanPrecedes(candidates[i], candidates[j])
	})
	for _, plan := range candidates {
		for _, candidate := range plan.Components {
			if candidate.Component != component {
				continue
			}
			return &contract.ResolvedModelPriceComponent{
				Component:   candidate,
				PlanSource:  plan.Source,
				BillingMode: plan.BillingMode,
				Plan:        plan,
			}, true
		}
	}
	return nil, false
}

func anyMatchingPlanHasChild(plans []contract.ModelPricePlan, query contract.ModelPricePlanQuery, children []contract.PriceComponent) bool {
	for _, child := range children {
		if _, ok := directMatchingComponent(plans, query, child); ok {
			return true
		}
	}
	return false
}

func inputChildTokens(bu *BillingUsage, component contract.PriceComponent) int {
	switch component {
	case contract.PriceComponentImageInput:
		return intValue(bu.ImageInputTokens)
	case contract.PriceComponentVideoInput:
		return intValue(bu.VideoInputTokens)
	case contract.PriceComponentDocumentInput:
		return intValue(bu.DocumentInputTokens)
	default:
		return 0
	}
}

func componentLabel(component contract.PriceComponent) string {
	switch component {
	case contract.PriceComponentImageInput:
		return "图像输入"
	case contract.PriceComponentVideoInput:
		return "视频输入"
	case contract.PriceComponentDocumentInput:
		return "文档输入"
	default:
		return string(component)
	}
}

func tokenComponentQuota(tokens int, unitPrice, exchangeRate, groupMultiplier, quotaPerUnit decimal.Decimal) decimal.Decimal {
	return decimal.NewFromInt(int64(tokens)).
		Div(decimal.NewFromInt(1_000_000)).
		Mul(unitPrice).
		Mul(exchangeRate).
		Mul(groupMultiplier).
		Mul(quotaPerUnit)
}

func parseComponentDecimals(component contract.ModelPriceComponent, plan contract.ModelPricePlan) (decimal.Decimal, decimal.Decimal, error) {
	unitPrice, err := decimal.NewFromString(component.UnitPrice)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("parse %s unit price: %w", component.Component, err)
	}
	exchangeRate, err := decimal.NewFromString(plan.ExchangeRate)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("parse exchange rate: %w", err)
	}
	return unitPrice, exchangeRate, nil
}

func unitPriceFloat(value string) float64 {
	parsed, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed.InexactFloat64()
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func RoundBillingQuota(value decimal.Decimal, mode contract.PriceRoundingMode) int {
	switch mode {
	case contract.PriceRoundingHalfEven:
		return int(value.RoundBank(0).IntPart())
	case contract.PriceRoundingFloor:
		return int(value.Floor().IntPart())
	case contract.PriceRoundingCeil:
		return int(value.Ceil().IntPart())
	default:
		return int(value.Round(0).IntPart())
	}
}

// AppendBillingPriceSnapshot adds settlement pricing to the existing Other
// snapshot. It is additive: every legacy pricing key remains authoritative for
// old frontend paths and historical-log compatibility.
func AppendBillingPriceSnapshot(other map[string]interface{}, result *BillingQuotaResult) {
	if other == nil || result == nil || result.PriceSnapshot == nil {
		return
	}
	other["billing_price_snapshot"] = result.PriceSnapshot
}

// IsLegacyPriceSettlement reports whether the result was settled from the
// legacy ratio projection. The legacy minimum-quota rule is a safety net for
// ratio misconfiguration; explicit price tables own their pricing, including
// legitimately zero-cost plans, so every entry must gate that rule on this
// check instead of re-deriving it from PriceData.
func IsLegacyPriceSettlement(result BillingQuotaResult) bool {
	return result.PriceSnapshot != nil &&
		normalizedPricePlanSource(result.PriceSnapshot.Source) == contract.PricePlanSourceLegacy
}

func newPriceComponentSnapshot(
	resolved *contract.ResolvedModelPriceComponent,
	groupMultiplier decimal.Decimal,
	quantity int,
) BillingPriceComponentSnapshot {
	return BillingPriceComponentSnapshot{
		Component:       resolved.Component.Component,
		Unit:            resolved.Component.Unit,
		Quantity:        quantity,
		UnitPrice:       resolved.Component.UnitPrice,
		Currency:        resolved.Plan.Currency,
		ExchangeRate:    resolved.Plan.ExchangeRate,
		GroupMultiplier: groupMultiplier.String(),
		PlanSource:      normalizedPricePlanSource(resolved.Plan.Source),
		PlanID:          resolved.Plan.ID,
	}
}

func newBillingPriceSnapshot(
	plan contract.ModelPricePlan,
	query contract.ModelPricePlanQuery,
	groupMultiplierSource contract.GroupMultiplierSource,
	groupMultiplier decimal.Decimal,
	roundingMode contract.PriceRoundingMode,
	precision int,
) *BillingPriceSnapshot {
	return &BillingPriceSnapshot{
		Source:                normalizedPricePlanSource(plan.Source),
		BillingMode:           plan.BillingMode,
		Endpoint:              query.Endpoint,
		ServiceTier:           query.ServiceTier,
		ContextTokens:         query.ContextTokens,
		ContextMinTokens:      plan.ContextMinTokens,
		ContextMaxTokens:      cloneIntPointer(plan.ContextMaxTokens),
		GroupMultiplierSource: groupMultiplierSource,
		GroupMultiplier:       groupMultiplier.String(),
		RoundingMode:          roundingMode,
		PricePrecision:        precision,
		Components:            []BillingPriceComponentSnapshot{},
	}
}

// legacyNormalizedQuota 按 PRD 3.4 节公式从归一化 BillingUsage 计算费用。
//
// 口径约定：
//   - 普通输入 = InputTokens()（raw 总量扣除缓存读取/写入），模态明细只在
//     存在差异化价格时移出（audio、image、video、document 只有价格表明确
//     差异化计价时才参与费用，否则不因明细存在而重复扣费）；
//   - 输出总量 = OutputTokens，reasoning 与 accepted/rejected prediction 是
//     输出审计拆分，包含在输出内按输出文本价计，不额外累加；
//   - 缓存写入 5m 档承担"官方 5m 档 + 未分档写入"（未分档按 5m 档计价）；
//   - Gemini toolUsePromptTokens 是审计字段，不进入任何计价行项。
func legacyNormalizedQuota(bu *BillingUsage, priceData contract.PriceData, mode AudioPricingMode, modelName string) (BillingQuotaResult, error) {
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
