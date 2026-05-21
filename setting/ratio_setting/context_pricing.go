package ratio_setting

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/types"
)

// Claude 1h cache creation keeps the existing fixed relationship to 5m cache creation pricing.
const ClaudeCacheCreation1hMultiplier = 6 / 3.75

var contextPricingMap = types.NewRWMap[string, types.ContextPricingConfig]()

func ContextPricing2JSONString() string {
	return contextPricingMap.MarshalJSONString()
}

func UpdateContextPricingByJSONString(jsonStr string) error {
	if err := ValidateContextPricing(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonString(contextPricingMap, jsonStr)
}

func GetContextPricingCopy() map[string]types.ContextPricingConfig {
	return contextPricingMap.ReadAll()
}

func GetContextPricingConfig(model string) (types.ContextPricingConfig, bool) {
	model = FormatMatchingModelName(model)
	cfg, ok := contextPricingMap.Get(model)
	return cfg, ok
}

func MatchContextPricingTier(model string, contextTokens int) (*types.ContextPricingResult, bool, error) {
	cfg, ok := GetContextPricingConfig(model)
	if !ok || !cfg.Enabled {
		return nil, false, nil
	}
	if contextTokens < 0 {
		contextTokens = 0
	}
	if len(cfg.Tiers) == 0 {
		return nil, true, fmt.Errorf("context pricing enabled for model %s but no tiers configured", model)
	}

	tiers := make([]types.ContextPricingTier, len(cfg.Tiers))
	copy(tiers, cfg.Tiers)
	sort.SliceStable(tiers, func(i, j int) bool {
		return tiers[i].MinTokens < tiers[j].MinTokens
	})

	for idx, tier := range tiers {
		if contextTokens < tier.MinTokens {
			continue
		}
		if tier.MaxTokens != nil && contextTokens >= *tier.MaxTokens {
			continue
		}
		prices := types.ContextPricingTierPrices{
			ModelRatio:           tier.ModelRatio,
			CompletionRatio:      tier.CompletionRatio,
			CacheRatio:           tier.CacheRatio,
			CacheCreationRatio:   tier.CreateCacheRatio,
			CacheCreation5mRatio: tier.CreateCacheRatio,
			CacheCreation1hRatio: tier.CreateCacheRatio * ClaudeCacheCreation1hMultiplier,
			AudioRatio:           tier.AudioRatio,
			AudioCompletionRatio: tier.AudioCompletionRatio,
		}
		return &types.ContextPricingResult{
			Enabled:              true,
			ContextTokensForTier: contextTokens,
			TierIndex:            idx,
			TierName:             tier.Name,
			MinTokens:            tier.MinTokens,
			MaxTokens:            tier.MaxTokens,
			Prices:               prices,
		}, true, nil
	}

	return nil, true, fmt.Errorf("context pricing enabled for model %s but no tier matches %d tokens", model, contextTokens)
}

func ApplyContextPricingResult(priceData *types.PriceData, result *types.ContextPricingResult) {
	if priceData == nil || result == nil || !result.Enabled {
		return
	}
	priceData.UsePrice = false
	priceData.ModelPrice = -1
	priceData.ModelRatio = result.Prices.ModelRatio
	priceData.CompletionRatio = result.Prices.CompletionRatio
	priceData.CacheRatio = result.Prices.CacheRatio
	priceData.CacheCreationRatio = result.Prices.CacheCreationRatio
	priceData.CacheCreation5mRatio = result.Prices.CacheCreation5mRatio
	priceData.CacheCreation1hRatio = result.Prices.CacheCreation1hRatio
	priceData.AudioRatio = result.Prices.AudioRatio
	priceData.AudioCompletionRatio = result.Prices.AudioCompletionRatio
	priceData.ContextPricing = result
}

type rawContextPricingConfig struct {
	Enabled bool                    `json:"enabled"`
	Tiers   []rawContextPricingTier `json:"tiers"`
}

type rawContextPricingTier struct {
	Name                 string   `json:"name,omitempty"`
	MinTokens            *int     `json:"min_tokens"`
	MaxTokens            *int     `json:"max_tokens,omitempty"`
	ModelRatio           *float64 `json:"model_ratio"`
	CompletionRatio      *float64 `json:"completion_ratio"`
	CacheRatio           *float64 `json:"cache_ratio"`
	CreateCacheRatio     *float64 `json:"create_cache_ratio"`
	AudioRatio           *float64 `json:"audio_ratio"`
	AudioCompletionRatio *float64 `json:"audio_completion_ratio"`
}

func ValidateContextPricing(jsonStr string) error {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return errors.New("context pricing must be a JSON object")
	}

	raw := make(map[string]rawContextPricingConfig)
	if err := common.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return err
	}

	for modelName, cfg := range raw {
		if !cfg.Enabled {
			continue
		}
		name := strings.TrimSpace(modelName)
		if name == "" {
			return errors.New("context pricing model name cannot be empty")
		}
		if len(cfg.Tiers) == 0 {
			return fmt.Errorf("context pricing for %s must include at least one tier", name)
		}

		tiers := make([]rawContextPricingTier, len(cfg.Tiers))
		copy(tiers, cfg.Tiers)
		for idx, tier := range tiers {
			if tier.MinTokens == nil {
				return fmt.Errorf("context pricing for %s tier %d missing min_tokens", name, idx+1)
			}
			if *tier.MinTokens < 0 {
				return fmt.Errorf("context pricing for %s tier %d min_tokens must be >= 0", name, idx+1)
			}
			if tier.MaxTokens != nil && *tier.MaxTokens <= *tier.MinTokens {
				return fmt.Errorf("context pricing for %s tier %d max_tokens must be greater than min_tokens", name, idx+1)
			}
			if err := validateContextPricingTierPrices(name, idx+1, tier); err != nil {
				return err
			}
		}

		sort.SliceStable(tiers, func(i, j int) bool {
			return *tiers[i].MinTokens < *tiers[j].MinTokens
		})
		for idx := 1; idx < len(tiers); idx++ {
			prev := tiers[idx-1]
			curr := tiers[idx]
			if prev.MaxTokens == nil {
				return fmt.Errorf("context pricing for %s tier %d has no upper bound and overlaps following tiers", name, idx)
			}
			if *curr.MinTokens < *prev.MaxTokens {
				return fmt.Errorf("context pricing for %s tiers overlap around %d tokens", name, *curr.MinTokens)
			}
		}
	}

	return nil
}

func validateContextPricingTierPrices(modelName string, tierIndex int, tier rawContextPricingTier) error {
	required := []struct {
		name  string
		value *float64
	}{
		{"model_ratio", tier.ModelRatio},
		{"completion_ratio", tier.CompletionRatio},
		{"cache_ratio", tier.CacheRatio},
		{"create_cache_ratio", tier.CreateCacheRatio},
		{"audio_ratio", tier.AudioRatio},
		{"audio_completion_ratio", tier.AudioCompletionRatio},
	}
	for _, item := range required {
		if item.value == nil {
			return fmt.Errorf("context pricing for %s tier %d missing %s", modelName, tierIndex, item.name)
		}
		if *item.value < 0 {
			return fmt.Errorf("context pricing for %s tier %d %s must be >= 0", modelName, tierIndex, item.name)
		}
	}
	return nil
}
