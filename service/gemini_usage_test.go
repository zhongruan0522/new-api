package service

import (
	"testing"

	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func TestGeminiUsageMetadataToOpenAIUsage(t *testing.T) {
	usage := GeminiUsageMetadataToOpenAIUsage(dto.GeminiUsageMetadata{
		PromptTokenCount:        10,
		CandidatesTokenCount:    6,
		ThoughtsTokenCount:      4,
		TotalTokenCount:         20,
		CachedContentTokenCount: 3,
		PromptTokensDetails:     []dto.GeminiPromptTokensDetails{{Modality: "TEXT", TokenCount: 8}, {Modality: "AUDIO", TokenCount: 2}},
		CandidatesTokensDetails: []dto.GeminiPromptTokensDetails{{Modality: "TEXT", TokenCount: 6}},
	})

	if usage.PromptTokens != 10 || usage.CompletionTokens != 10 || usage.TotalTokens != 20 {
		t.Fatalf("usage = %+v, want prompt=10 completion=10 total=20", usage)
	}
	if usage.PromptTokensDetails.CachedTokens != 3 || usage.PromptCacheHitTokens != 3 {
		t.Fatalf("usage cache details = %+v, want cached=3", usage)
	}
	if usage.PromptTokensDetails.TextTokens != 8 || usage.PromptTokensDetails.AudioTokens != 2 {
		t.Fatalf("prompt token details = %+v, want text=8 audio=2", usage.PromptTokensDetails)
	}
	if usage.CompletionTokenDetails.TextTokens != 6 || usage.CompletionTokenDetails.ReasoningTokens != 4 {
		t.Fatalf("completion token details = %+v, want text=6 reasoning=4", usage.CompletionTokenDetails)
	}
}

func TestOpenAIUsageToGeminiUsage(t *testing.T) {
	metadata := OpenAIUsageToGeminiUsage(dto.Usage{
		PromptTokens:     12,
		CompletionTokens: 9,
		TotalTokens:      21,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 2,
			TextTokens:   12,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{
			TextTokens:      6,
			ReasoningTokens: 3,
		},
	})

	if metadata.PromptTokenCount != 12 || metadata.CandidatesTokenCount != 6 || metadata.ThoughtsTokenCount != 3 || metadata.TotalTokenCount != 21 {
		t.Fatalf("metadata = %+v, want prompt=12 candidates=6 thoughts=3 total=21", metadata)
	}
	if metadata.CachedContentTokenCount != 2 {
		t.Fatalf("metadata cached_content_token_count = %d, want 2", metadata.CachedContentTokenCount)
	}
	if len(metadata.PromptTokensDetails) != 1 || metadata.PromptTokensDetails[0].TokenCount != 12 {
		t.Fatalf("prompt token details = %+v, want one text detail=12", metadata.PromptTokensDetails)
	}
	if len(metadata.CandidatesTokensDetails) != 1 || metadata.CandidatesTokensDetails[0].TokenCount != 6 {
		t.Fatalf("candidate token details = %+v, want one text detail=6", metadata.CandidatesTokensDetails)
	}
}

func TestResponseOpenAI2GeminiPreservesReasoningAndUsage(t *testing.T) {
	resp := ResponseOpenAI2Gemini(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{{
			Index: 0,
			Message: dto.Message{
				Role:             "assistant",
				Content:          "answer",
				ReasoningContent: "thinking",
			},
			FinishReason: "stop",
		}},
		Usage: dto.Usage{
			PromptTokens:     7,
			CompletionTokens: 5,
			TotalTokens:      12,
			CompletionTokenDetails: dto.OutputTokenDetails{
				ReasoningTokens: 2,
			},
		},
	}, nil)

	if resp.UsageMetadata.PromptTokenCount != 7 || resp.UsageMetadata.CandidatesTokenCount != 3 || resp.UsageMetadata.ThoughtsTokenCount != 2 {
		t.Fatalf("usage metadata = %+v, want prompt=7 candidates=3 thoughts=2", resp.UsageMetadata)
	}
	if len(resp.Candidates) != 1 || len(resp.Candidates[0].Content.Parts) != 2 {
		t.Fatalf("candidates = %+v, want one candidate with reasoning and text parts", resp.Candidates)
	}
	if !resp.Candidates[0].Content.Parts[0].Thought || resp.Candidates[0].Content.Parts[0].Text != "thinking" {
		t.Fatalf("first part = %+v, want thought part", resp.Candidates[0].Content.Parts[0])
	}
	if resp.Candidates[0].Content.Parts[1].Text != "answer" {
		t.Fatalf("second part = %+v, want text part", resp.Candidates[0].Content.Parts[1])
	}
}

func TestStreamResponseOpenAI2GeminiPreservesReasoningDelta(t *testing.T) {
	reasoning := "thinking"
	content := "answer"
	resp := StreamResponseOpenAI2Gemini(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
				ReasoningContent: &reasoning,
				Content:          &content,
			},
		}},
	}, &relaycommon.RelayInfo{})

	if resp == nil {
		t.Fatal("response is nil, want non-nil chunk")
	}
	if len(resp.Candidates) != 1 || len(resp.Candidates[0].Content.Parts) != 2 {
		t.Fatalf("candidates = %+v, want one candidate with reasoning and text parts", resp.Candidates)
	}
	if !resp.Candidates[0].Content.Parts[0].Thought || resp.Candidates[0].Content.Parts[0].Text != "thinking" {
		t.Fatalf("first part = %+v, want thought delta", resp.Candidates[0].Content.Parts[0])
	}
	if resp.Candidates[0].Content.Parts[1].Text != "answer" {
		t.Fatalf("second part = %+v, want text delta", resp.Candidates[0].Content.Parts[1])
	}
}
