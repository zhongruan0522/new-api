package service

import (
	"strings"

	"github.com/zhongruan0522/new-api/dto"
)

// HasGeminiUsageMetadata reports whether Gemini returned any usage payload worth mapping.
func HasGeminiUsageMetadata(metadata dto.GeminiUsageMetadata) bool {
	return metadata.PromptTokenCount > 0 ||
		metadata.CandidatesTokenCount > 0 ||
		metadata.TotalTokenCount > 0 ||
		metadata.ThoughtsTokenCount > 0 ||
		metadata.CachedContentTokenCount > 0 ||
		len(metadata.PromptTokensDetails) > 0 ||
		len(metadata.CandidatesTokensDetails) > 0
}

// GeminiUsageMetadataToOpenAIUsage normalizes Gemini usage metadata into the local OpenAI-compatible shape.
func GeminiUsageMetadataToOpenAIUsage(metadata dto.GeminiUsageMetadata) dto.Usage {
	usage := dto.Usage{
		PromptTokens:     metadata.PromptTokenCount,
		CompletionTokens: metadata.CandidatesTokenCount + metadata.ThoughtsTokenCount,
		TotalTokens:      metadata.TotalTokenCount,
		InputTokens:      metadata.PromptTokenCount,
		OutputTokens:     metadata.CandidatesTokenCount + metadata.ThoughtsTokenCount,
	}

	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	usage.PromptCacheHitTokens = metadata.CachedContentTokenCount
	usage.PromptTokensDetails.CachedTokens = metadata.CachedContentTokenCount
	usage.CompletionTokenDetails.ReasoningTokens = metadata.ThoughtsTokenCount

	for _, detail := range metadata.PromptTokensDetails {
		switch strings.ToUpper(strings.TrimSpace(detail.Modality)) {
		case "TEXT":
			usage.PromptTokensDetails.TextTokens = detail.TokenCount
		case "AUDIO":
			usage.PromptTokensDetails.AudioTokens = detail.TokenCount
		case "IMAGE":
			usage.PromptTokensDetails.ImageTokens = detail.TokenCount
		}
	}

	for _, detail := range metadata.CandidatesTokensDetails {
		switch strings.ToUpper(strings.TrimSpace(detail.Modality)) {
		case "TEXT":
			usage.CompletionTokenDetails.TextTokens = detail.TokenCount
		case "AUDIO":
			usage.CompletionTokenDetails.AudioTokens = detail.TokenCount
		}
	}

	// Gemini often omits modality breakdowns; keep a text fallback only when no finer detail exists.
	if len(metadata.PromptTokensDetails) == 0 && usage.PromptTokens > 0 {
		usage.PromptTokensDetails.TextTokens = usage.PromptTokens
	}
	if len(metadata.CandidatesTokensDetails) == 0 && metadata.CandidatesTokenCount > 0 {
		usage.CompletionTokenDetails.TextTokens = metadata.CandidatesTokenCount
	}

	return usage
}

// OpenAIUsageToGeminiUsage rewrites OpenAI-compatible usage into Gemini's usage schema.
func OpenAIUsageToGeminiUsage(usage dto.Usage) dto.GeminiUsageMetadata {
	promptDetails := usage.PromptTokensDetails
	if usage.InputTokensDetails != nil && promptDetails == (dto.InputTokenDetails{}) {
		promptDetails = *usage.InputTokensDetails
	}

	promptTokens := usage.PromptTokens
	if promptTokens == 0 {
		promptTokens = usage.InputTokens
	}
	completionTokens := usage.CompletionTokens
	if completionTokens == 0 {
		completionTokens = usage.OutputTokens
	}

	metadata := dto.GeminiUsageMetadata{
		PromptTokenCount:        promptTokens,
		CandidatesTokenCount:    completionTokens,
		TotalTokenCount:         usage.TotalTokens,
		CachedContentTokenCount: promptDetails.CachedTokens,
		PromptTokensDetails:     buildGeminiPromptTokenDetails(promptDetails),
	}

	if metadata.TotalTokenCount == 0 {
		metadata.TotalTokenCount = promptTokens + completionTokens
	}

	if reasoningTokens := usage.CompletionTokenDetails.ReasoningTokens; reasoningTokens > 0 {
		metadata.ThoughtsTokenCount = reasoningTokens
		if reasoningTokens <= completionTokens {
			metadata.CandidatesTokenCount = completionTokens - reasoningTokens
		}
	}

	metadata.CandidatesTokensDetails = buildGeminiCandidateTokenDetails(usage.CompletionTokenDetails)
	return metadata
}

func buildGeminiPromptTokenDetails(details dto.InputTokenDetails) []dto.GeminiPromptTokensDetails {
	out := make([]dto.GeminiPromptTokensDetails, 0, 3)
	if details.TextTokens > 0 {
		out = append(out, dto.GeminiPromptTokensDetails{Modality: "TEXT", TokenCount: details.TextTokens})
	}
	if details.AudioTokens > 0 {
		out = append(out, dto.GeminiPromptTokensDetails{Modality: "AUDIO", TokenCount: details.AudioTokens})
	}
	if details.ImageTokens > 0 {
		out = append(out, dto.GeminiPromptTokensDetails{Modality: "IMAGE", TokenCount: details.ImageTokens})
	}
	return out
}

func buildGeminiCandidateTokenDetails(details dto.OutputTokenDetails) []dto.GeminiPromptTokensDetails {
	out := make([]dto.GeminiPromptTokensDetails, 0, 2)
	if details.TextTokens > 0 {
		out = append(out, dto.GeminiPromptTokensDetails{Modality: "TEXT", TokenCount: details.TextTokens})
	}
	if details.AudioTokens > 0 {
		out = append(out, dto.GeminiPromptTokensDetails{Modality: "AUDIO", TokenCount: details.AudioTokens})
	}
	return out
}
