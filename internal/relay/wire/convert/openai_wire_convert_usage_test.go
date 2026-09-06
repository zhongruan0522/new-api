package convert

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

// Chat-to-Responses output conversion serves the client-visible wire format.
// Cache-write and provider-specific billing/audit fields are deliberately not
// carried through: billing consumes the original upstream usage before the
// converted response is written.
func TestMapChatUsageToResponsesUsage_ExcludesBillingOnlyAuditFields(t *testing.T) {
	source := shared.Usage{
		PromptTokens:     30,
		CompletionTokens: 12,
		TotalTokens:      42,
		PromptTokensDetails: shared.InputTokenDetails{
			CachedTokens:         4,
			CachedCreationTokens: 8,
		},
		CompletionTokenDetails: shared.OutputTokenDetails{
			ReasoningTokens:          5,
			AcceptedPredictionTokens: 2,
			RejectedPredictionTokens: 1,
		},
		ClaudeCacheCreation5mTokens: 3,
		ClaudeCacheCreation1hTokens: 5,
	}

	got := MapChatUsageToResponsesUsage(source)
	if got == nil {
		t.Fatal("MapChatUsageToResponsesUsage() = nil")
	}
	if got.InputTokensDetails == nil || got.InputTokensDetails.CachedTokens != 4 || got.InputTokensDetails.CachedCreationTokens != 0 {
		t.Fatalf("input details = %+v, want cache read preserved and cache write dropped", got.InputTokensDetails)
	}
	if got.OutputTokensDetails == nil || got.OutputTokensDetails.ReasoningTokens != 5 ||
		got.OutputTokensDetails.AcceptedPredictionTokens != 0 || got.OutputTokensDetails.RejectedPredictionTokens != 0 {
		t.Fatalf("output details = %+v, want reasoning preserved and prediction audit dropped", got.OutputTokensDetails)
	}
	if got.ClaudeCacheCreation5mTokens != 0 || got.ClaudeCacheCreation1hTokens != 0 {
		t.Fatalf("Claude cache fields = %+v, want zero", got)
	}

	raw, err := jsonx.Marshal(got)
	if err != nil {
		t.Fatalf("marshal usage: %v", err)
	}
	for _, forbidden := range []string{
		`claude_cache_creation_5_m_tokens":3`,
		`claude_cache_creation_1_h_tokens":5`,
		`accepted_prediction_tokens":2`,
		`rejected_prediction_tokens":1`,
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("converted usage leaks %s: %s", forbidden, raw)
		}
	}
}

func TestApplyResponsesUsageToChatUsageHonorsExplicitZeroOverFallback(t *testing.T) {
	dst := &shared.Usage{}
	usage := &shared.Usage{
		InputTokens:          10,
		OutputTokens:         5,
		PromptCacheHitTokens: 40,
	}
	usage.PromptTokensDetails.CachedTokens = 0
	usage.PromptTokensDetails.CachedTokensPresent = true

	ApplyResponsesUsageToChatUsage(dst, usage)
	if dst.PromptTokensDetails.CachedTokens != 0 || !dst.PromptTokensDetails.CachedTokensPresent {
		t.Fatalf("prompt details = %+v, want explicit zero preserved", dst.PromptTokensDetails)
	}

	dst = &shared.Usage{}
	usage.InputTokensDetails = &shared.InputTokenDetails{CachedTokens: 12}
	ApplyResponsesUsageToChatUsage(dst, usage)
	if dst.PromptTokensDetails.CachedTokens != 12 {
		t.Fatalf("prompt details = %+v, want Responses input detail fallback", dst.PromptTokensDetails)
	}
}
