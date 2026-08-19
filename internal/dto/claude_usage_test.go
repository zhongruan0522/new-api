package dto

import "testing"

func TestClaudeUsageToOpenAIUsageIncludesCachedPromptTokens(t *testing.T) {
	usage := ClaudeUsageToOpenAIUsage(&ClaudeUsage{
		InputTokens:              100,
		CacheCreationInputTokens: 50,
		CacheReadInputTokens:     30,
		OutputTokens:             20,
	})

	if usage == nil {
		t.Fatal("expected usage")
	}
	if usage.PromptTokens != 180 {
		t.Fatalf("PromptTokens = %d, want 180", usage.PromptTokens)
	}
	if usage.PromptTokensDetails.CachedTokens != 30 {
		t.Fatalf("CachedTokens = %d, want 30", usage.PromptTokensDetails.CachedTokens)
	}
	if usage.PromptTokensDetails.CachedCreationTokens != 50 {
		t.Fatalf("CachedCreationTokens = %d, want 50", usage.PromptTokensDetails.CachedCreationTokens)
	}
	if usage.TotalTokens != 200 {
		t.Fatalf("TotalTokens = %d, want 200", usage.TotalTokens)
	}
}

func TestOpenAIUsageToClaudeUsageSplitsPromptTokens(t *testing.T) {
	usage := OpenAIUsageToClaudeUsage(&Usage{
		PromptTokens:     180,
		CompletionTokens: 20,
		PromptTokensDetails: InputTokenDetails{
			CachedTokens:         30,
			CachedCreationTokens: 50,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 40,
	})

	if usage == nil {
		t.Fatal("expected usage")
	}
	if usage.InputTokens != 100 {
		t.Fatalf("InputTokens = %d, want 100", usage.InputTokens)
	}
	if usage.CacheReadInputTokens != 30 {
		t.Fatalf("CacheReadInputTokens = %d, want 30", usage.CacheReadInputTokens)
	}
	if usage.CacheCreationInputTokens != 50 {
		t.Fatalf("CacheCreationInputTokens = %d, want 50", usage.CacheCreationInputTokens)
	}
	if usage.OutputTokens != 20 {
		t.Fatalf("OutputTokens = %d, want 20", usage.OutputTokens)
	}
	if usage.CacheCreation == nil {
		t.Fatal("expected cache creation breakdown")
	}
	if usage.CacheCreation.Ephemeral5mInputTokens != 10 || usage.CacheCreation.Ephemeral1hInputTokens != 40 {
		t.Fatalf("CacheCreation = %+v, want 10/40", usage.CacheCreation)
	}
}
