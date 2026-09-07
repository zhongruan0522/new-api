package billing

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/config/ratio"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"

	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/gin-gonic/gin"
)

const serviceContextPricingConfig = `{
  "service-tier-model": {
    "enabled": true,
    "tiers": [
      {
        "name": "<200K",
        "min_tokens": 0,
        "max_tokens": 200000,
        "model_ratio": 1,
        "completion_ratio": 2,
        "cache_ratio": 0.5,
        "create_cache_ratio": 1.25,
        "audio_ratio": 3,
        "audio_completion_ratio": 4
      },
      {
        "name": ">=200K",
        "min_tokens": 200000,
        "model_ratio": 10,
        "completion_ratio": 20,
        "cache_ratio": 5,
        "create_cache_ratio": 12.5,
        "audio_ratio": 30,
        "audio_completion_ratio": 40
      }
    ]
  }
}`

func installServiceContextPricing(t *testing.T) {
	t.Helper()
	if err := ratio.UpdateContextPricingByJSONString(serviceContextPricingConfig); err != nil {
		t.Fatalf("UpdateContextPricingByJSONString returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = ratio.UpdateContextPricingByJSONString("{}")
	})
}

// buildContextPricingTestUsage 构造 OpenAI Chat 语义的归一化用量
// （prompt_tokens 含缓存读取/写入）。
func buildContextPricingTestUsage(promptTokens, completionTokens, cachedTokens, cachedCreationTokens int) *BillingUsage {
	usage := &shared.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
	usage.PromptTokensDetails.CachedTokens = cachedTokens
	usage.PromptTokensDetails.CachedCreationTokens = cachedCreationTokens
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceOpenAIChat, usage, nil)
	if err != nil {
		panic(err)
	}
	return bu
}

func TestApplyContextPricingDisabledLeavesPriceDataUnchanged(t *testing.T) {
	if err := ratio.UpdateContextPricingByJSONString("{}"); err != nil {
		t.Fatalf("failed to reset context pricing: %v", err)
	}

	priceData := contract.PriceData{
		ModelRatio:           1,
		CompletionRatio:      2,
		CacheRatio:           3,
		CacheCreationRatio:   4,
		AudioRatio:           5,
		AudioCompletionRatio: 6,
	}

	result, enabled, err := ApplyContextPricingForBillingUsage("missing-model", buildContextPricingTestUsage(300000, 0, 0, 0), &priceData)
	if err != nil {
		t.Fatalf("ApplyContextPricingForBillingUsage returned error: %v", err)
	}
	if enabled || result != nil {
		t.Fatalf("expected disabled context pricing, got enabled=%v result=%+v", enabled, result)
	}
	if priceData.ModelRatio != 1 || priceData.CompletionRatio != 2 || priceData.CacheRatio != 3 ||
		priceData.CacheCreationRatio != 4 || priceData.AudioRatio != 5 || priceData.AudioCompletionRatio != 6 {
		t.Fatalf("priceData mutated when context pricing disabled: %+v", priceData)
	}
}

// 阶段 2：档位 tokens = 普通输入 + 输出 + 缓存读取 + 缓存写入（含输出维度）。
// 输入 199000 不足 200K，但叠加输出 80000 后总处理量进入高档位。
func TestApplyContextPricingTierIncludesOutputDimension(t *testing.T) {
	installServiceContextPricing(t)

	priceData := contract.PriceData{}
	result, enabled, err := ApplyContextPricingForBillingUsage("service-tier-model", buildContextPricingTestUsage(199000, 80000, 0, 0), &priceData)
	if err != nil {
		t.Fatalf("ApplyContextPricingForBillingUsage returned error: %v", err)
	}
	if !enabled || result == nil {
		t.Fatalf("expected enabled context pricing")
	}
	if result.TierName != ">=200K" || result.ContextTokensForTier != 279000 {
		t.Fatalf("result = %+v, want high tier with input+output 279000", result)
	}
	if priceData.ModelRatio != 10 {
		t.Fatalf("model ratio = %v, want high tier ratio 10", priceData.ModelRatio)
	}
}

// 输入与输出之和未达档位阈值时命中低档位。
func TestApplyContextPricingLowTierBelowThreshold(t *testing.T) {
	installServiceContextPricing(t)

	priceData := contract.PriceData{}
	result, enabled, err := ApplyContextPricingForBillingUsage("service-tier-model", buildContextPricingTestUsage(199000, 800, 0, 0), &priceData)
	if err != nil {
		t.Fatalf("ApplyContextPricingForBillingUsage returned error: %v", err)
	}
	if !enabled || result == nil {
		t.Fatalf("expected enabled context pricing")
	}
	if result.TierName != "<200K" || result.ContextTokensForTier != 199800 {
		t.Fatalf("result = %+v, want low tier with 199800 tokens", result)
	}
	if priceData.ModelRatio != 1 || priceData.CompletionRatio != 2 {
		t.Fatalf("low tier did not apply prices to priceData: %+v", priceData)
	}
}

// 缓存读取/写入参与档位 tokens 且不重复计数：Claude 聚合 180 =
// 普通输入 100 + 读取 30 + 写入 50，四维相加仍为 180。
func TestContextTokensForTierAvoidsDoubleCountingClaudeCache(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens: 180,
		PromptTokensDetails: shared.InputTokenDetails{
			CachedTokens:         30,
			CachedCreationTokens: 50,
		},
	}
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceClaude, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	if bu.InputTokens() != 100 {
		t.Fatalf("normal input tokens = %d, want 100", bu.InputTokens())
	}
	if got := ContextTokensForTier(bu); got != 180 {
		t.Fatalf("ContextTokensForTier = %d, want 180", got)
	}
}

// Claude 缓存命中把档位推入高档：聚合 210000 + 输出 1 = 210001 ≥ 200K，
// 档位价格全量写回 PriceData（价格快照仍写入现有位置）。
func TestApplyContextPricingCacheCanPushClaudeUsageToHighTier(t *testing.T) {
	installServiceContextPricing(t)

	usage := &shared.Usage{
		PromptTokens:     210000,
		CompletionTokens: 1,
		PromptTokensDetails: shared.InputTokenDetails{
			CachedTokens:         30000,
			CachedCreationTokens: 30000,
		},
	}
	bu, _, err := BuildBillingUsage(relayconstant.UsageSourceClaude, usage, nil)
	if err != nil {
		t.Fatalf("BuildBillingUsage returned error: %v", err)
	}
	priceData := contract.PriceData{}
	result, enabled, err := ApplyContextPricingForBillingUsage("service-tier-model", bu, &priceData)
	if err != nil {
		t.Fatalf("ApplyContextPricingForBillingUsage returned error: %v", err)
	}
	if !enabled || result == nil {
		t.Fatalf("expected enabled context pricing")
	}
	if result.TierName != ">=200K" || result.ContextTokensForTier != 210001 {
		t.Fatalf("result = %+v, want high tier with 210001 context tokens", result)
	}
	if priceData.ModelRatio != 10 || priceData.CompletionRatio != 20 || priceData.CacheRatio != 5 ||
		priceData.CacheCreationRatio != 12.5 || priceData.AudioRatio != 30 || priceData.AudioCompletionRatio != 40 {
		t.Fatalf("high tier did not apply all prices to priceData: %+v", priceData)
	}
}

func TestGenerateTextOtherInfoIncludesContextPricingAuditFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("use_channel", []string{"1"})

	maxTokens := 1000000
	relayInfo := &relaycommon.RelayInfo{
		StartTime:         time.Unix(100, 0),
		FirstResponseTime: time.Unix(100, int64(50*time.Millisecond)),
		RequestURLPath:    "/v1/chat/completions",
		ChannelMeta:       &relaycommon.ChannelMeta{},
		PriceData: contract.PriceData{
			ContextPricing: &contract.ContextPricingResult{
				Enabled:              true,
				ContextTokensForTier: 250000,
				TierIndex:            1,
				TierName:             "200K~1000K",
				MinTokens:            200000,
				MaxTokens:            &maxTokens,
				Prices: contract.ContextPricingTierPrices{
					ModelRatio:           10,
					CompletionRatio:      20,
					CacheRatio:           5,
					CacheCreationRatio:   12.5,
					CacheCreation5mRatio: 12.5,
					CacheCreation1hRatio: 20,
					AudioRatio:           30,
					AudioCompletionRatio: 40,
				},
			},
		},
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, 10, 1.5, 20, 0.5, -1, -1, 1.2)
	if enabled, ok := other["context_pricing_enabled"].(bool); !ok || !enabled {
		t.Fatalf("context_pricing_enabled = %#v, want true", other["context_pricing_enabled"])
	}
	if got := other["context_tokens_for_tier"]; got != 250000 {
		t.Fatalf("context_tokens_for_tier = %#v, want 250000", got)
	}
	if got := other["context_pricing_tier_name"]; got != "200K~1000K" {
		t.Fatalf("context_pricing_tier_name = %#v", got)
	}
	if got := other["dynamic_ratio"]; got != 1.2 {
		t.Fatalf("dynamic_ratio = %#v, want 1.2", got)
	}
	prices, ok := other["context_pricing_prices"].(contract.ContextPricingTierPrices)
	if !ok {
		t.Fatalf("context_pricing_prices type = %T", other["context_pricing_prices"])
	}
	if prices.ModelRatio != 10 || prices.AudioCompletionRatio != 40 {
		t.Fatalf("context_pricing_prices = %+v", prices)
	}
}
