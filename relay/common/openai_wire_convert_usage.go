package common

import "github.com/zhongruan0522/new-api/dto"

// ApplyResponsesUsageToChatUsage maps OpenAI Responses usage fields onto the
// Chat Completions usage fields used internally for quota calculation.
func ApplyResponsesUsageToChatUsage(dst *dto.Usage, usage *dto.Usage) {
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
	} else if usage.PromptTokensDetails != (dto.InputTokenDetails{}) {
		dst.PromptTokensDetails = usage.PromptTokensDetails
	}
	if dst.PromptTokensDetails.CachedTokens == 0 && usage.PromptCacheHitTokens > 0 {
		dst.PromptTokensDetails.CachedTokens = usage.PromptCacheHitTokens
	}
	dst.PromptCacheHitTokens = firstNonZero(usage.PromptCacheHitTokens, dst.PromptTokensDetails.CachedTokens)

	if usage.OutputTokensDetails != nil {
		dst.CompletionTokenDetails = *usage.OutputTokensDetails
	} else if usage.CompletionTokenDetails != (dto.OutputTokenDetails{}) {
		dst.CompletionTokenDetails = usage.CompletionTokenDetails
	}
}

// MapChatUsageToResponsesUsage maps Chat Completions usage to the Responses
// usage shape, including token detail fields that affect billing.
func MapChatUsageToResponsesUsage(u dto.Usage) *dto.Usage {
	inputTokens := firstNonZero(u.PromptTokens, u.InputTokens)
	outputTokens := firstNonZero(u.CompletionTokens, u.OutputTokens)
	totalTokens := u.TotalTokens
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}

	inputDetails := u.PromptTokensDetails
	if inputDetails == (dto.InputTokenDetails{}) && u.InputTokensDetails != nil {
		inputDetails = *u.InputTokensDetails
	}
	if inputDetails.CachedTokens == 0 && u.PromptCacheHitTokens > 0 {
		inputDetails.CachedTokens = u.PromptCacheHitTokens
	}

	outputDetails := u.CompletionTokenDetails
	if outputDetails == (dto.OutputTokenDetails{}) && u.OutputTokensDetails != nil {
		outputDetails = *u.OutputTokensDetails
	}

	return &dto.Usage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		InputTokensDetails: &dto.InputTokenDetails{
			CachedTokens: inputDetails.CachedTokens,
			TextTokens:   inputDetails.TextTokens,
			AudioTokens:  inputDetails.AudioTokens,
			ImageTokens:  inputDetails.ImageTokens,
		},
		OutputTokensDetails: &dto.OutputTokenDetails{
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
