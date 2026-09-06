package openai

import (
	"testing"

	channelconstant "github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
)

func TestApplyUsagePostProcessingKeepsExplicitZeroCache(t *testing.T) {
	usage := &shared.Usage{
		PromptTokens:         100,
		PromptCacheHitTokens: 40,
	}
	usage.PromptTokensDetails.CachedTokens = 0
	usage.PromptTokensDetails.CachedTokensPresent = true

	applyUsagePostProcessing(
		&relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{ChannelType: channelconstant.ChannelTypeZhipu_v4},
		},
		usage,
		[]byte(`{"usage":{"prompt_tokens_details":{"cached_tokens":0}}}`),
	)
	if usage.PromptTokensDetails.CachedTokens != 0 {
		t.Fatalf("cached tokens = %d, want explicit upstream zero", usage.PromptTokensDetails.CachedTokens)
	}
}

func TestAccumulateRealtimeUsageKeepsExplicitZeroCache(t *testing.T) {
	dst := &shared.RealtimeUsage{InputTokens: 10, OutputTokens: 5}
	src := &shared.RealtimeUsage{}
	src.InputTokenDetails.CachedTokensPresent = true

	accumulateRealtimeUsage(dst, src)
	if !dst.InputTokenDetails.CachedTokensPresent {
		t.Fatal("explicit cached_tokens=0 presence was lost while accumulating realtime usage")
	}
}
