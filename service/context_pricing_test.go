package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
	"github.com/zhongruan0522/new-api/types"

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
	if err := ratio_setting.UpdateContextPricingByJSONString(serviceContextPricingConfig); err != nil {
		t.Fatalf("UpdateContextPricingByJSONString returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = ratio_setting.UpdateContextPricingByJSONString("{}")
	})
}

func TestApplyContextPricingDisabledLeavesPriceDataUnchanged(t *testing.T) {
	if err := ratio_setting.UpdateContextPricingByJSONString("{}"); err != nil {
		t.Fatalf("failed to reset context pricing: %v", err)
	}

	priceData := types.PriceData{
		ModelRatio:           1,
		CompletionRatio:      2,
		CacheRatio:           3,
		CacheCreationRatio:   4,
		AudioRatio:           5,
		AudioCompletionRatio: 6,
	}

	result, enabled, err := ApplyContextPricingForUsage("missing-model", ContextPricingUsage{PromptTokens: 300000}, &priceData)
	if err != nil {
		t.Fatalf("ApplyContextPricingForUsage returned error: %v", err)
	}
	if enabled || result != nil {
		t.Fatalf("expected disabled context pricing, got enabled=%v result=%+v", enabled, result)
	}
	if priceData.ModelRatio != 1 || priceData.CompletionRatio != 2 || priceData.CacheRatio != 3 ||
		priceData.CacheCreationRatio != 4 || priceData.AudioRatio != 5 || priceData.AudioCompletionRatio != 6 {
		t.Fatalf("priceData mutated when context pricing disabled: %+v", priceData)
	}
}

func TestApplyContextPricingMatchesInputContextAndIgnoresOutput(t *testing.T) {
	installServiceContextPricing(t)

	usage := &dto.Usage{
		PromptTokens:     199000,
		CompletionTokens: 800000,
	}
	priceData := types.PriceData{}
	result, enabled, err := ApplyContextPricingForUsage("service-tier-model", BuildContextPricingUsage(usage, false), &priceData)
	if err != nil {
		t.Fatalf("ApplyContextPricingForUsage returned error: %v", err)
	}
	if !enabled || result == nil {
		t.Fatalf("expected enabled context pricing")
	}
	if result.TierName != "<200K" || result.ContextTokensForTier != 199000 {
		t.Fatalf("result = %+v, want low tier matched by input only", result)
	}
	if priceData.ModelRatio != 1 {
		t.Fatalf("model ratio = %v, want low tier ratio 1", priceData.ModelRatio)
	}
}

func TestApplyContextPricingCacheCanPushClaudeUsageToHighTier(t *testing.T) {
	installServiceContextPricing(t)

	usage := &dto.Usage{
		PromptTokens:     210000,
		CompletionTokens: 1,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         30000,
			CachedCreationTokens: 30000,
		},
	}
	priceData := types.PriceData{}
	result, enabled, err := ApplyContextPricingForUsage("service-tier-model", BuildContextPricingUsage(usage, true), &priceData)
	if err != nil {
		t.Fatalf("ApplyContextPricingForUsage returned error: %v", err)
	}
	if !enabled || result == nil {
		t.Fatalf("expected enabled context pricing")
	}
	if result.TierName != ">=200K" || result.ContextTokensForTier != 210000 {
		t.Fatalf("result = %+v, want high tier with 210000 context tokens", result)
	}
	if priceData.ModelRatio != 10 || priceData.CompletionRatio != 20 || priceData.CacheRatio != 5 ||
		priceData.CacheCreationRatio != 12.5 || priceData.AudioRatio != 30 || priceData.AudioCompletionRatio != 40 {
		t.Fatalf("high tier did not apply all prices to priceData: %+v", priceData)
	}
}

func TestContextTokensForTierAvoidsDoubleCountingClaudeCache(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens: 180,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         30,
			CachedCreationTokens: 50,
		},
	}

	contextUsage := BuildContextPricingUsage(usage, true)
	if contextUsage.PromptTokens != 100 {
		t.Fatalf("base prompt tokens = %d, want 100", contextUsage.PromptTokens)
	}
	if got := ContextTokensForTier(contextUsage); got != 180 {
		t.Fatalf("ContextTokensForTier = %d, want 180", got)
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
		PriceData: types.PriceData{
			ContextPricing: &types.ContextPricingResult{
				Enabled:              true,
				ContextTokensForTier: 250000,
				TierIndex:            1,
				TierName:             "200K~1000K",
				MinTokens:            200000,
				MaxTokens:            &maxTokens,
				Prices: types.ContextPricingTierPrices{
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

	other := GenerateTextOtherInfo(ctx, relayInfo, 10, 1.5, 20, 0, 0.5, -1, -1, 1.2)
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
	prices, ok := other["context_pricing_prices"].(types.ContextPricingTierPrices)
	if !ok {
		t.Fatalf("context_pricing_prices type = %T", other["context_pricing_prices"])
	}
	if prices.ModelRatio != 10 || prices.AudioCompletionRatio != 40 {
		t.Fatalf("context_pricing_prices = %+v", prices)
	}
}
