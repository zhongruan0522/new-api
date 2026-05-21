package types

type ContextPricingConfig struct {
	Enabled bool                 `json:"enabled"`
	Tiers   []ContextPricingTier `json:"tiers"`
}

type ContextPricingTier struct {
	Name                 string  `json:"name,omitempty"`
	MinTokens            int     `json:"min_tokens"`
	MaxTokens            *int    `json:"max_tokens,omitempty"`
	ModelRatio           float64 `json:"model_ratio"`
	CompletionRatio      float64 `json:"completion_ratio"`
	CacheRatio           float64 `json:"cache_ratio"`
	CreateCacheRatio     float64 `json:"create_cache_ratio"`
	AudioRatio           float64 `json:"audio_ratio"`
	AudioCompletionRatio float64 `json:"audio_completion_ratio"`
}

type ContextPricingResult struct {
	Enabled              bool                     `json:"enabled"`
	ContextTokensForTier int                      `json:"context_tokens_for_tier"`
	TierIndex            int                      `json:"tier_index"`
	TierName             string                   `json:"tier_name,omitempty"`
	MinTokens            int                      `json:"min_tokens"`
	MaxTokens            *int                     `json:"max_tokens,omitempty"`
	Prices               ContextPricingTierPrices `json:"prices"`
}

type ContextPricingTierPrices struct {
	ModelRatio           float64 `json:"model_ratio"`
	CompletionRatio      float64 `json:"completion_ratio"`
	CacheRatio           float64 `json:"cache_ratio"`
	CacheCreationRatio   float64 `json:"cache_creation_ratio"`
	CacheCreation5mRatio float64 `json:"cache_creation_ratio_5m"`
	CacheCreation1hRatio float64 `json:"cache_creation_ratio_1h"`
	AudioRatio           float64 `json:"audio_ratio"`
	AudioCompletionRatio float64 `json:"audio_completion_ratio"`
}
