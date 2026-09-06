package convert

import "github.com/NookMux/NookMux/internal/domain/shared"

// ApplyResponsesUsageToChatUsage maps OpenAI Responses usage fields onto the
// Chat Completions usage fields used internally for quota calculation.
func ApplyResponsesUsageToChatUsage(dst *shared.Usage, usage *shared.Usage) {
	if dst == nil || usage == nil {
		return
	}

	dst.PromptTokens = firstNonZero(usage.InputTokens, usage.PromptTokens)
	dst.CompletionTokens = firstNonZero(usage.OutputTokens, usage.CompletionTokens)
	dst.TotalTokens = usage.TotalTokens
	if dst.TotalTokens == 0 {
		dst.TotalTokens = dst.PromptTokens + dst.CompletionTokens
	}

	if usage.InputTokensDetails != nil {
		dst.PromptTokensDetails = *usage.InputTokensDetails
	} else if hasInputTokenDetailValues(usage.PromptTokensDetails) ||
		usage.PromptTokensDetails.CachedTokensPresent {
		dst.PromptTokensDetails = usage.PromptTokensDetails
	}
	if !dst.PromptTokensDetails.CachedTokensPresent &&
		dst.PromptTokensDetails.CachedTokens == 0 && usage.PromptCacheHitTokens > 0 {
		dst.PromptTokensDetails.CachedTokens = usage.PromptCacheHitTokens
	}
	dst.PromptCacheHitTokens = firstNonZero(usage.PromptCacheHitTokens, dst.PromptTokensDetails.CachedTokens)

	if usage.OutputTokensDetails != nil {
		dst.CompletionTokenDetails = *usage.OutputTokensDetails
	} else if hasOutputTokenDetailValues(usage.CompletionTokenDetails) {
		dst.CompletionTokenDetails = usage.CompletionTokenDetails
	}
}

// MapChatUsageToResponsesUsage maps only the Responses wire-visible usage
// fields. Billing-only or protocol-specific audit fields must stay on the
// original upstream usage consumed by domain/billing; do not smuggle them into
// the converted client response.
func MapChatUsageToResponsesUsage(u shared.Usage) *shared.Usage {
	inputTokens := firstNonZero(u.PromptTokens, u.InputTokens)
	outputTokens := firstNonZero(u.CompletionTokens, u.OutputTokens)
	totalTokens := u.TotalTokens
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}

	inputDetails := u.PromptTokensDetails
	if !hasInputTokenDetailValues(inputDetails) && u.InputTokensDetails != nil {
		inputDetails = *u.InputTokensDetails
	}
	if !inputDetails.CachedTokensPresent &&
		inputDetails.CachedTokens == 0 && u.PromptCacheHitTokens > 0 {
		inputDetails.CachedTokens = u.PromptCacheHitTokens
	}

	outputDetails := u.CompletionTokenDetails
	if !hasOutputTokenDetailValues(outputDetails) && u.OutputTokensDetails != nil {
		outputDetails = *u.OutputTokensDetails
	}

	return &shared.Usage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		InputTokensDetails: &shared.InputTokenDetails{
			CachedTokens: inputDetails.CachedTokens,
			TextTokens:   inputDetails.TextTokens,
			AudioTokens:  inputDetails.AudioTokens,
			ImageTokens:  inputDetails.ImageTokens,
		},
		OutputTokensDetails: &shared.OutputTokenDetails{
			TextTokens:      outputDetails.TextTokens,
			AudioTokens:     outputDetails.AudioTokens,
			ReasoningTokens: outputDetails.ReasoningTokens,
		},
	}
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func hasInputTokenDetailValues(details shared.InputTokenDetails) bool {
	return details.CachedTokens != 0 ||
		details.CachedCreationTokens != 0 ||
		details.TextTokens != 0 ||
		details.AudioTokens != 0 ||
		details.ImageTokens != 0
}

func hasOutputTokenDetailValues(details shared.OutputTokenDetails) bool {
	return details.TextTokens != 0 ||
		details.AudioTokens != 0 ||
		details.ReasoningTokens != 0 ||
		details.AcceptedPredictionTokens != 0 ||
		details.RejectedPredictionTokens != 0
}
