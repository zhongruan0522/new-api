package ratio_setting

import "testing"

const contextPricingTestConfig = `{
  "tier-test-model": {
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

func TestContextPricingValidationAndLeftClosedBounds(t *testing.T) {
	if err := UpdateContextPricingByJSONString(contextPricingTestConfig); err != nil {
		t.Fatalf("UpdateContextPricingByJSONString returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = UpdateContextPricingByJSONString("{}")
	})

	low, enabled, err := MatchContextPricingTier("tier-test-model", 199999)
	if err != nil {
		t.Fatalf("MatchContextPricingTier low returned error: %v", err)
	}
	if !enabled || low == nil {
		t.Fatalf("expected context pricing to be enabled and matched")
	}
	if low.TierName != "<200K" || low.Prices.ModelRatio != 1 {
		t.Fatalf("low tier = %+v, want <200K with model_ratio 1", low)
	}

	high, enabled, err := MatchContextPricingTier("tier-test-model", 200000)
	if err != nil {
		t.Fatalf("MatchContextPricingTier high returned error: %v", err)
	}
	if !enabled || high == nil {
		t.Fatalf("expected context pricing to be enabled and matched")
	}
	if high.TierName != ">=200K" || high.Prices.ModelRatio != 10 {
		t.Fatalf("high tier = %+v, want >=200K with model_ratio 10", high)
	}
	if high.Prices.CacheCreation1hRatio != high.Prices.CacheCreationRatio*ClaudeCacheCreation1hMultiplier {
		t.Fatalf("1h cache creation ratio = %v, want %v",
			high.Prices.CacheCreation1hRatio,
			high.Prices.CacheCreationRatio*ClaudeCacheCreation1hMultiplier)
	}
}

func TestContextPricingValidationRejectsInvalidConfigs(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "overlap",
			json: `{
			  "bad-model": {
			    "enabled": true,
			    "tiers": [
			      {"min_tokens": 0, "max_tokens": 200000, "model_ratio": 1, "completion_ratio": 1, "cache_ratio": 1, "create_cache_ratio": 1, "audio_ratio": 1, "audio_completion_ratio": 1},
			      {"min_tokens": 199999, "model_ratio": 2, "completion_ratio": 2, "cache_ratio": 2, "create_cache_ratio": 2, "audio_ratio": 2, "audio_completion_ratio": 2}
			    ]
			  }
			}`,
		},
		{
			name: "missing required price",
			json: `{
			  "bad-model": {
			    "enabled": true,
			    "tiers": [
			      {"min_tokens": 0, "model_ratio": 1, "completion_ratio": 1, "cache_ratio": 1, "create_cache_ratio": 1, "audio_ratio": 1}
			    ]
			  }
			}`,
		},
		{
			name: "invalid bounds",
			json: `{
			  "bad-model": {
			    "enabled": true,
			    "tiers": [
			      {"min_tokens": 1000, "max_tokens": 1000, "model_ratio": 1, "completion_ratio": 1, "cache_ratio": 1, "create_cache_ratio": 1, "audio_ratio": 1, "audio_completion_ratio": 1}
			    ]
			  }
			}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateContextPricing(test.json); err == nil {
				t.Fatalf("ValidateContextPricing expected error")
			}
		})
	}
}
