package billing

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// 影子对拍测试（计费 PRD 阶段 2 迁移期）：断言旧公式与新公式在无缓存/
// 无模态场景下 quota 相同（迁移期不变量），并对已知语义差异类别给出精确
// 差值——差异必须能定位为旧公式 bug 或新语义 bug，不允许吸收。

func shadowTestPriceData() contract.PriceData {
	return contract.PriceData{
		ModelRatio:           2,
		CompletionRatio:      3,
		CacheRatio:           0.5,
		CacheCreationRatio:   1.25,
		CacheCreation5mRatio: 1.25,
		CacheCreation1hRatio: 2.0,
		AudioRatio:           8,
		AudioCompletionRatio: 2,
		GroupRatioInfo:       contract.GroupRatioInfo{GroupRatio: 1},
	}
}

// 通用文本路径：无缓存无模态时新旧公式 quota 相同。
func TestShadowGenericParityWithoutCache(t *testing.T) {
	usage := &shared.Usage{PromptTokens: 700, CompletionTokens: 500, TotalTokens: 1200}
	priceData := shadowTestPriceData()

	legacy := legacyGenericFinalQuota(usage, false, priceData, "gpt-4o", decimal.Zero, nil)
	bu, err := buildAggregateBillingUsage(usage)
	if err != nil {
		t.Fatalf("buildAggregateBillingUsage: %v", err)
	}
	result, err := CalculateNormalizedQuota(bu, priceData, AudioPricingAbsolute, "gpt-4o")
	if err != nil {
		t.Fatalf("CalculateNormalizedQuota: %v", err)
	}
	normalized := int(result.TokenTotal.Round(0).IntPart())
	// (700 + 500×3) × 2 = 4400
	if legacy != 4400 || normalized != 4400 {
		t.Fatalf("legacy=%d normalized=%d, want both 4400", legacy, normalized)
	}
}

// 已知差异（旧公式 bug）：带 prompt_cache_hit_tokens 的 DeepSeek 风格 usage
// 在旧公式下缓存命中 tokens 被按全价计费；新公式识别并按缓存单价计。
func TestShadowGenericDeepSeekCacheHitTokensDifference(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens:         1000,
		CompletionTokens:     500,
		TotalTokens:          1500,
		PromptCacheHitTokens: 200,
	}
	priceData := shadowTestPriceData()

	legacy := legacyGenericFinalQuota(usage, false, priceData, "deepseek-chat", decimal.Zero, nil)
	// 旧公式只看 PromptTokensDetails.CachedTokens（0）：(1000 + 1500) × 2 = 5000
	if legacy != 5000 {
		t.Fatalf("legacy = %d, want 5000 (cache-hit tokens billed at full price)", legacy)
	}

	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceOpenAIChat, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage: %v", err)
	}
	result, err := CalculateNormalizedQuota(bu, priceData, AudioPricingAbsolute, "deepseek-chat")
	if err != nil {
		t.Fatalf("CalculateNormalizedQuota: %v", err)
	}
	normalized := int(result.TokenTotal.Round(0).IntPart())
	// 新公式：(800 + 200×0.5 + 1500) × 2 = 4800
	if normalized != 4800 {
		t.Fatalf("normalized = %d, want 4800", normalized)
	}
	if legacy-normalized != 200*0.5*2 {
		t.Fatalf("diff = %d, want legacy overcharge of cache-hit×(1-cacheRatio)×modelRatio", legacy-normalized)
	}
}

// 已知差异（旧公式 bug）：原生 Claude 缓存读写按基础价重复计费。
// legacy - normalized = (read+write) × 1 × modelRatio × groupRatio。
func TestShadowClaudeCacheDoubleBillingDifference(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens:     1000, // Claude 聚合 = input(700) + read(200) + write(100)
		CompletionTokens: 500,
		TotalTokens:      1500,
	}
	usage.PromptTokensDetails.CachedTokens = 200
	usage.PromptTokensDetails.CachedCreationTokens = 100
	priceData := shadowTestPriceData()

	legacy := legacyClaudeQuota(usage, false, priceData, "claude-sonnet-4-5")
	// 旧公式：aggregate×1 + 200×0.5 + 100×1.25（未分档按 5m 价） + 500×3
	//        = 1000+100+125+1500 = 2725 → ×2 = 5450
	if legacy != 5450 {
		t.Fatalf("legacy = %d, want 5450", legacy)
	}

	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceClaude, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage: %v", err)
	}
	result, err := CalculateNormalizedQuota(bu, priceData, AudioPricingAbsolute, "claude-sonnet-4-5")
	if err != nil {
		t.Fatalf("CalculateNormalizedQuota: %v", err)
	}
	normalized := int(result.TokenTotal.IntPart())
	// 新公式（PRD 3.4）：700×1 + 200×0.5 + 100×1.25 + 500×3 = 2455 → ×2 = 4850
	if normalized != 4850 {
		t.Fatalf("normalized = %d, want 4850", normalized)
	}
	if legacy-normalized != 600 {
		t.Fatalf("diff = %d, want 600 = (read+write)×modelRatio", legacy-normalized)
	}
}

// 无缓存的原生 Claude：新旧公式一致。
func TestShadowClaudeParityWithoutCache(t *testing.T) {
	usage := &shared.Usage{PromptTokens: 700, CompletionTokens: 500, TotalTokens: 1200}
	priceData := shadowTestPriceData()

	legacy := legacyClaudeQuota(usage, false, priceData, "claude-sonnet-4-5")
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceClaude, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage: %v", err)
	}
	result, err := CalculateNormalizedQuota(bu, priceData, AudioPricingAbsolute, "claude-sonnet-4-5")
	if err != nil {
		t.Fatalf("CalculateNormalizedQuota: %v", err)
	}
	normalized := int(result.TokenTotal.IntPart())
	if legacy != 4400 || normalized != 4400 {
		t.Fatalf("legacy=%d normalized=%d, want both 4400", legacy, normalized)
	}
}

// audio 路径：官方 text/audio 明细与聚合一致时新旧公式相同。
func TestShadowAudioParityWithConsistentDetails(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens:     1000, // text(700) + audio(200) + cached(100)
		CompletionTokens: 500,
		TotalTokens:      1500,
	}
	usage.PromptTokensDetails.TextTokens = 700
	usage.PromptTokensDetails.AudioTokens = 200
	usage.PromptTokensDetails.CachedTokens = 100
	usage.CompletionTokenDetails.TextTokens = 400
	usage.CompletionTokenDetails.AudioTokens = 100
	priceData := shadowTestPriceData()

	legacy := legacyAudioQuota(QuotaInfo{
		InputDetails:         TokenDetails{TextTokens: 700, AudioTokens: 200},
		OutputDetails:        TokenDetails{TextTokens: 400, AudioTokens: 100},
		ModelName:            "gpt-4o-audio-preview",
		ModelRatio:           priceData.ModelRatio,
		GroupRatio:           priceData.GroupRatioInfo.GroupRatio,
		CompletionRatio:      priceData.CompletionRatio,
		AudioRatio:           priceData.AudioRatio,
		AudioCompletionRatio: priceData.AudioCompletionRatio,
	})
	// 旧公式漏计缓存读取：(700 + 400×3 + 200×8 + 100×8×2) × 2 = 10200
	if legacy != 10200 {
		t.Fatalf("legacy = %d, want 10200", legacy)
	}

	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceOpenAIChat, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage: %v", err)
	}
	result, err := CalculateNormalizedQuota(bu, priceData, AudioPricingRatioModel, "gpt-4o-audio-preview")
	if err != nil {
		t.Fatalf("CalculateNormalizedQuota: %v", err)
	}
	normalized := int(result.TokenTotal.Round(0).IntPart())
	// 新公式计缓存读取：(700 + 200×8 + 100×0.5 + 400×3 + 100×8×2) × 2 = 10300
	if normalized != 10300 {
		t.Fatalf("normalized = %d, want 10300", normalized)
	}
}

// 本地字符数计费（MiniMax TTS 风格伪 usage）：新旧公式一致。
func TestShadowAudioParityForLocalCharacterUsage(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens:     100,
		CompletionTokens: 100,
		TotalTokens:      200,
	}
	usage.PromptTokensDetails.TextTokens = 100
	usage.CompletionTokenDetails.AudioTokens = 100
	priceData := shadowTestPriceData()

	legacy := legacyAudioQuota(QuotaInfo{
		InputDetails:         TokenDetails{TextTokens: 100, AudioTokens: 0},
		OutputDetails:        TokenDetails{TextTokens: 0, AudioTokens: 100},
		ModelName:            "minimax-tts",
		ModelRatio:           priceData.ModelRatio,
		GroupRatio:           priceData.GroupRatioInfo.GroupRatio,
		CompletionRatio:      priceData.CompletionRatio,
		AudioRatio:           priceData.AudioRatio,
		AudioCompletionRatio: priceData.AudioCompletionRatio,
	})
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceOpenAIChat, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage: %v", err)
	}
	result, err := CalculateNormalizedQuota(bu, priceData, AudioPricingRatioModel, "minimax-tts")
	if err != nil {
		t.Fatalf("CalculateNormalizedQuota: %v", err)
	}
	normalized := int(result.TokenTotal.Round(0).IntPart())
	// (100×1 + 100×8×2) × 2 = 3400
	if legacy != 3400 || normalized != 3400 {
		t.Fatalf("legacy=%d normalized=%d, want both 3400", legacy, normalized)
	}
}

// 已知差异分类：Claude 缓存命中在 classifyShadowDiff 中标注预期类别。
func TestClassifyShadowDiffKnownClasses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	claudeUsage := &shared.Usage{PromptTokens: 1000, CompletionTokens: 500}
	claudeUsage.PromptTokensDetails.CachedTokens = 200
	claudeBU, _, err := BuildBillingUsage(relayconstant.UsageSourceClaude, claudeUsage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage: %v", err)
	}
	hint := classifyShadowDiff("claude", nil, claudeBU, false)
	if !strings.Contains(hint, "PRD 3.4") {
		t.Fatalf("claude cache hint = %q, want PRD 3.4 reference", hint)
	}

	plainBU, _, err := BuildBillingUsage(relayconstant.UsageSourceOpenAIChat, &shared.Usage{PromptTokens: 700, CompletionTokens: 500}, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage: %v", err)
	}
	hint = classifyShadowDiff("generic", nil, plainBU, false)
	if !strings.Contains(hint, "unclassified") {
		t.Fatalf("plain generic hint = %q, want unclassified", hint)
	}
}
