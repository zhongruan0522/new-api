package billing

import (
	"testing"

	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/shopspring/decimal"
)

// CalculateNormalizedQuota 单元测试：四个规范族在 PRD 3.4 公式下的 quota
// 逐行拆解。价格表沿用阶段 2 倍率制（ModelRatio/CompletionRatio/CacheRatio/
// CacheCreationRatio 族 + AudioRatio 族），分组倍率固定为 1。

func normalizedQuotaTestPriceData() contract.PriceData {
	return contract.PriceData{
		ModelRatio:           2,
		CompletionRatio:      3,
		CacheRatio:           0.5,
		CacheCreationRatio:   1.25,
		CacheCreation5mRatio: 1.25,
		CacheCreation1hRatio: 2.0, // 生产口径：1h = 5m × 1.6
		AudioRatio:           8,
		AudioCompletionRatio: 2,
		GroupRatioInfo:       contract.GroupRatioInfo{GroupRatio: 1},
	}
}

func mustQuota(t *testing.T, bu *BillingUsage, priceData contract.PriceData, mode AudioPricingMode, modelName string) BillingQuotaResult {
	t.Helper()
	result, err := calculateNormalizedQuota(bu, priceData, mode, modelName)
	if err != nil {
		t.Fatalf("CalculateNormalizedQuota returned error: %v", err)
	}
	return result
}

func assertQuotaTotal(t *testing.T, result BillingQuotaResult, want int64) {
	t.Helper()
	if result.TokenTotal.Cmp(decimal.NewFromInt(want)) != 0 {
		t.Fatalf("TokenTotal = %s, want %d", result.TokenTotal.String(), want)
	}
}

// OpenAI Chat（PRD 3.4）：普通输入/输出已扣除缓存；图片/音频输入明细在
// 倍率价格表无差异化价格维度时随普通输入计价，不重复扣费。
func TestCalculateNormalizedQuotaOpenAIChatModalitiesWithoutPriceDimensions(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens:     1000, // raw 总量，含缓存与模态明细
		CompletionTokens: 500,
		TotalTokens:      1500,
	}
	usage.PromptTokensDetails.CachedTokens = 200
	usage.PromptTokensDetails.CachedCreationTokens = 100
	usage.PromptTokensDetails.ImageTokens = 50
	usage.PromptTokensDetails.AudioTokens = 100
	usage.CompletionTokenDetails.AudioTokens = 100

	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceOpenAIChat, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	// 普通输入 = 1000-200-100 = 700（image/audio 明细无差异化单价，不扣）；
	// 输出 = 500（输出音频无差异化单价，不扣）。
	// quota = (700 + 200×0.5 + 100×1.25 + 500×3) × 2 = 2425×2 = 4850
	result := mustQuota(t, bu, normalizedQuotaTestPriceData(), AudioPricingAbsolute, "gpt-4o")
	assertQuotaTotal(t, result, 4850)
}

// OpenAI Chat 缓存写入分档：5m/1h 官方分档分别按 5m/1h 单价计。
func TestCalculateNormalizedQuotaCacheWriteTiers(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}
	usage.PromptTokensDetails.CachedTokens = 200
	usage.PromptTokensDetails.CachedCreationTokens = 100
	usage.ClaudeCacheCreation5mTokens = 60
	usage.ClaudeCacheCreation1hTokens = 40

	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceClaude, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	// quota = (700×1 + 200×0.5 + 60×1.25 + 40×2.0 + 500×3) × 2
	//       = (700 + 100 + 75 + 80 + 1500) × 2 = 2455×2 = 4910
	result := mustQuota(t, bu, normalizedQuotaTestPriceData(), AudioPricingAbsolute, "claude-sonnet-4-5")
	assertQuotaTotal(t, result, 4910)
}

// Gemini（PRD 3.4）：音频输入按每百万美元单价差异化计价（不乘模型倍率），
// 图片/视频/文档明细无差异化价格时随普通输入计价；thoughts 计入输出。
func TestCalculateNormalizedQuotaGeminiAudioAbsolutePrice(t *testing.T) {
	metadata := &shared.GeminiUsageMetadata{
		PromptTokenCount:        1000,
		CandidatesTokenCount:    500,
		ThoughtsTokenCount:      50,
		CachedContentTokenCount: 200,
		PromptTokensDetails: []shared.GeminiPromptTokensDetails{
			{Modality: "AUDIO", TokenCount: 100},
			{Modality: "IMAGE", TokenCount: 50},
		},
	}
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceGemini, nil, metadata)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	if bu.InputTokens() != 800 {
		t.Fatalf("InputTokens = %d, want 800", bu.InputTokens())
	}
	// gemini-2.5-flash 每百万音频输入单价 1.00 美元：
	// audioQuota = 1.0/1e6 × 100 × 1 × 500000 = 50
	// token 部分 = (800-100)×1 + 200×0.5 + 550×3 = 700+100+1650 = 2450 → ×2 = 4900
	result := mustQuota(t, bu, normalizedQuotaTestPriceData(), AudioPricingAbsolute, "gemini-2.5-flash")
	assertQuotaTotal(t, result, 4900)
	if result.AudioInputQuota.Cmp(decimal.NewFromInt(50)) != 0 {
		t.Fatalf("AudioInputQuota = %s, want 50", result.AudioInputQuota.String())
	}
	if result.AudioInputPrice != 1.0 {
		t.Fatalf("AudioInputPrice = %v, want 1.0", result.AudioInputPrice)
	}
}

// 无每百万音频单价的模型（不在 operation 音频价格前缀表内）：
// 音频输入不差异化，随普通输入计价（与旧口径一致）。
func TestCalculateNormalizedQuotaGeminiWithoutAudioPrice(t *testing.T) {
	metadata := &shared.GeminiUsageMetadata{
		PromptTokenCount:     1000,
		CandidatesTokenCount: 500,
		ThoughtsTokenCount:   50,
		PromptTokensDetails: []shared.GeminiPromptTokensDetails{
			{Modality: "AUDIO", TokenCount: 100},
		},
	}
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceGemini, nil, metadata)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	// quota = 1000×1 + 550×3 = 2650 → ×2 = 5300；音频输入保持普通输入计价
	result := mustQuota(t, bu, normalizedQuotaTestPriceData(), AudioPricingAbsolute, "gemini-1.5-flash")
	assertQuotaTotal(t, result, 5300)
	if !result.AudioInputQuota.IsZero() {
		t.Fatalf("AudioInputQuota = %s, want 0", result.AudioInputQuota.String())
	}
}

// audio/realtime（RatioModel 模式）：音频输入按 AudioRatio、音频输出按
// AudioRatio×AudioCompletionRatio 差异化计价；缓存读取按 CacheRatio 计费。
func TestCalculateNormalizedQuotaRealtimeRelativeAudio(t *testing.T) {
	usage := &shared.RealtimeUsage{
		TotalTokens:  1500,
		InputTokens:  1000,
		OutputTokens: 500,
		InputTokenDetails: shared.InputTokenDetails{
			TextTokens:   700,
			AudioTokens:  200,
			CachedTokens: 100,
		},
		OutputTokenDetails: shared.OutputTokenDetails{
			TextTokens:  400,
			AudioTokens: 100,
		},
	}
	bu, _, err := BuildRealtimeBillingUsage(relayconstant.UsageSourceOpenAIResponses, usage)
	if err != nil {
		t.Fatalf("BuildRealtimeBillingUsage returned error: %v", err)
	}
	// 普通输入 = 1000-100-200 = 700；非音频输出 = 500-100 = 400
	// quota = (700×1 + 200×8 + 100×0.5 + 400×3 + 100×8×2) × 2
	//       = (700 + 1600 + 50 + 1200 + 1600) × 2 = 5150×2 = 10300
	result := mustQuota(t, bu, normalizedQuotaTestPriceData(), AudioPricingRatioModel, "gpt-4o-realtime-preview")
	assertQuotaTotal(t, result, 10300)
}

// 按次计费：ModelPrice × QuotaPerUnit × GroupRatio，token 明细不参与。
func TestCalculateNormalizedQuotaUsePrice(t *testing.T) {
	priceData := normalizedQuotaTestPriceData()
	priceData.UsePrice = true
	priceData.ModelPrice = 0.02
	bu := &BillingUsage{PromptAggregateTokens: 1000000, OutputTokens: 1000000}
	result := mustQuota(t, bu, priceData, AudioPricingAbsolute, "any")
	if !result.UsePrice {
		t.Fatal("expected UsePrice result")
	}
	assertQuotaTotal(t, result, 10000)
}

// buildAggregateBillingUsage：本地估算/伪 usage 按聚合口径构造，
// 与旧通用公式口径一致（缓存读写从总量扣除）。
func TestBuildAggregateBillingUsageMirrorsLegacyAggregateSemantics(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}
	usage.PromptTokensDetails.CachedTokens = 200
	usage.PromptTokensDetails.CachedCreationTokens = 100

	bu, err := buildAggregateBillingUsage(usage)
	if err != nil {
		t.Fatalf("buildAggregateBillingUsage returned error: %v", err)
	}
	if bu.InputTokens() != 700 {
		t.Fatalf("InputTokens = %d, want 700", bu.InputTokens())
	}
	// quota = (700 + 200×0.5 + 100×1.25 + 500×3) × 2 = 4850（与 OpenAI Chat 聚合口径一致）
	result := mustQuota(t, bu, normalizedQuotaTestPriceData(), AudioPricingAbsolute, "gpt-4o")
	assertQuotaTotal(t, result, 4850)
}

// 无缓存无模态：四个规范族归一化结果一致（同一 quota）。
func TestCalculateNormalizedQuotaNoCacheParityAcrossSources(t *testing.T) {
	plainUsage := &shared.Usage{
		PromptTokens:     700,
		CompletionTokens: 500,
		TotalTokens:      1200,
	}

	claudeBU, _, err := BuildBillingUsage(relayconstant.UsageSourceClaude, plainUsage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	chatBU, _, err := BuildBillingUsage(relayconstant.UsageSourceOpenAIChat, plainUsage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	responsesBU, _, err := BuildBillingUsage(relayconstant.UsageSourceOpenAIResponses, plainUsage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}

	priceData := normalizedQuotaTestPriceData()
	// quota = (700 + 500×3) × 2 = 4400
	for name, bu := range map[string]*BillingUsage{
		"claude": claudeBU, "openai_chat": chatBU, "openai_responses": responsesBU,
	} {
		result := mustQuota(t, bu, priceData, AudioPricingAbsolute, "any-model")
		assertQuotaTotal(t, result, 4400)
		_ = name
	}
}

// Gemini 纯文本（无缓存无模态）与其它规范族 parity。
func TestCalculateNormalizedQuotaGeminiPlainTextParity(t *testing.T) {
	metadata := &shared.GeminiUsageMetadata{
		PromptTokenCount:     700,
		CandidatesTokenCount: 500,
	}
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceGemini, nil, metadata)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	// quota = (700 + 500×3) × 2 = 4400（与 Claude/OpenAI no-cache 一致）
	result := mustQuota(t, bu, normalizedQuotaTestPriceData(), AudioPricingAbsolute, "gemini-1.5-flash")
	assertQuotaTotal(t, result, 4400)
}

// Gemini 官方口径重叠：promptTokensDetails 的 TEXT 明细包含缓存读取部分。
// schema v1 的 text_input 必须去除缓存，避免同一 token 同时进入输入模态和
// read_cache；计费公式继续使用 PromptAggregate - read 作为普通输入。
func TestCalculateNormalizedQuotaGeminiTextCacheOverlapNoDoubleBilling(t *testing.T) {
	metadata := &shared.GeminiUsageMetadata{
		PromptTokenCount:        1000, // 含 200 缓存读取
		CandidatesTokenCount:    500,
		CachedContentTokenCount: 200,
		PromptTokensDetails: []shared.GeminiPromptTokensDetails{
			{Modality: "TEXT", TokenCount: 1000}, // 官方按总量拆分，含缓存部分
		},
	}
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceGemini, nil, metadata)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	if bu.TextInputTokens == nil || *bu.TextInputTokens != 800 {
		t.Fatalf("text_input = %v, want 800 (cache removed)", bu.TextInputTokens)
	}
	// quota = (1000-200)×1 + 200×0.5 + 500×3 = 800+100+1500 = 2400 → ×2 = 4800
	// TEXT 明细（800）不含缓存，与 read_cache（200）不重叠也不重复计费
	result := mustQuota(t, bu, normalizedQuotaTestPriceData(), AudioPricingAbsolute, "gemini-1.5-flash")
	assertQuotaTotal(t, result, 4800)
}
