package service

import (
	"github.com/zhongruan0522/new-api/dto"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
	"github.com/zhongruan0522/new-api/types"
)

type ContextPricingUsage struct {
	PromptTokens          int
	CompletionTokens      int
	CacheReadTokens       int
	CacheCreationTokens   int
	CacheCreation5mTokens int
	CacheCreation1hTokens int
	AudioInputTokens      int
	AudioOutputTokens     int
	IsClaudeUsageSemantic bool
}

func BuildContextPricingUsage(usage *dto.Usage, isClaudeUsageSemantic bool) ContextPricingUsage {
	if usage == nil {
		return ContextPricingUsage{IsClaudeUsageSemantic: isClaudeUsageSemantic}
	}
	cacheCreationTokens := usage.PromptTokensDetails.CachedCreationTokens
	promptTokens := usage.PromptTokens
	if isClaudeUsageSemantic {
		cacheCreationTokens = usage.ClaudeCacheCreation5mTokens + usage.ClaudeCacheCreation1hTokens
		if cacheCreationTokens == 0 {
			cacheCreationTokens = usage.PromptTokensDetails.CachedCreationTokens
		}
		// Claude usage is normalized with cache tokens already included in PromptTokens.
		// Store only the non-cache input side here; ContextTokensForTier adds cache back once.
		promptTokens = usage.PromptTokens - usage.PromptTokensDetails.CachedTokens - cacheCreationTokens
		if promptTokens < 0 {
			promptTokens = usage.PromptTokens
		}
	}
	return ContextPricingUsage{
		PromptTokens:          promptTokens,
		CompletionTokens:      usage.CompletionTokens,
		CacheReadTokens:       usage.PromptTokensDetails.CachedTokens,
		CacheCreationTokens:   cacheCreationTokens,
		CacheCreation5mTokens: usage.ClaudeCacheCreation5mTokens,
		CacheCreation1hTokens: usage.ClaudeCacheCreation1hTokens,
		AudioInputTokens:      usage.PromptTokensDetails.AudioTokens,
		AudioOutputTokens:     usage.CompletionTokenDetails.AudioTokens,
		IsClaudeUsageSemantic: isClaudeUsageSemantic,
	}
}

func BuildRealtimeContextPricingUsage(usage *dto.RealtimeUsage) ContextPricingUsage {
	if usage == nil {
		return ContextPricingUsage{}
	}
	return ContextPricingUsage{
		PromptTokens:      usage.InputTokens,
		CompletionTokens:  usage.OutputTokens,
		CacheReadTokens:   usage.InputTokenDetails.CachedTokens,
		AudioInputTokens:  usage.InputTokenDetails.AudioTokens,
		AudioOutputTokens: usage.OutputTokenDetails.AudioTokens,
	}
}

func ContextTokensForTier(usage ContextPricingUsage) int {
	if usage.IsClaudeUsageSemantic {
		total := usage.PromptTokens + usage.CacheReadTokens + usage.CacheCreationTokens
		if total < 0 {
			return 0
		}
		return total
	}
	if usage.PromptTokens < 0 {
		return 0
	}
	return usage.PromptTokens
}

func ApplyContextPricingForUsage(modelName string, usage ContextPricingUsage, priceData *types.PriceData) (*types.ContextPricingResult, bool, error) {
	contextTokens := ContextTokensForTier(usage)
	result, enabled, err := ratio_setting.MatchContextPricingTier(modelName, contextTokens)
	if err != nil || !enabled || result == nil {
		return result, enabled, err
	}
	ratio_setting.ApplyContextPricingResult(priceData, result)
	return result, true, nil
}
