package service

import (
	"testing"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/dto"
	"github.com/NookMux/NookMux/pkg/jsonx"
	relaycommon "github.com/NookMux/NookMux/relay/common"
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

func TestGeminiUsageMetadataToOpenAIUsageIncludesToolUsePromptTokens(t *testing.T) {
	usage := GeminiUsageMetadataToOpenAIUsage(dto.GeminiUsageMetadata{
		PromptTokenCount:        151,
		ToolUsePromptTokenCount: 18329,
		CandidatesTokenCount:    1089,
		ThoughtsTokenCount:      1120,
		TotalTokenCount:         20689,
		PromptTokensDetails: []dto.GeminiPromptTokensDetails{
			{Modality: "TEXT", TokenCount: 151},
		},
		ToolUsePromptTokensDetails: []dto.GeminiPromptTokensDetails{
			{Modality: "TEXT", TokenCount: 18329},
		},
		CandidatesTokensDetails: []dto.GeminiPromptTokensDetails{
			{Modality: "TEXT", TokenCount: 1089},
		},
	})

	if usage.PromptTokens != 18480 || usage.InputTokens != 18480 {
		t.Fatalf("prompt/input tokens = %d/%d, want 18480/18480", usage.PromptTokens, usage.InputTokens)
	}
	if usage.CompletionTokens != 2209 || usage.OutputTokens != 2209 || usage.TotalTokens != 20689 {
		t.Fatalf("completion/output/total = %d/%d/%d, want 2209/2209/20689", usage.CompletionTokens, usage.OutputTokens, usage.TotalTokens)
	}
	if usage.PromptTokensDetails.TextTokens != 18480 {
		t.Fatalf("prompt text token details = %d, want 18480", usage.PromptTokensDetails.TextTokens)
	}
	if usage.CompletionTokenDetails.TextTokens != 1089 || usage.CompletionTokenDetails.ReasoningTokens != 1120 {
		t.Fatalf("completion token details = %+v, want text=1089 reasoning=1120", usage.CompletionTokenDetails)
	}
}

func TestGeminiUsageMetadataToOpenAIUsageAcceptsSnakeCaseToolUseFields(t *testing.T) {
	var metadata dto.GeminiUsageMetadata
	if err := jsonx.Unmarshal([]byte(`{
		"prompt_token_count": 2,
		"tool_use_prompt_token_count": 3,
		"candidates_token_count": 5,
		"tool_use_prompt_tokens_details": [{"modality":"TEXT","tokenCount":3}]
	}`), &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	usage := GeminiUsageMetadataToOpenAIUsage(metadata)
	if usage.PromptTokens != 5 || usage.InputTokens != 5 {
		t.Fatalf("prompt/input tokens = %d/%d, want 5/5", usage.PromptTokens, usage.InputTokens)
	}
	if usage.PromptTokensDetails.TextTokens != 3 {
		t.Fatalf("tool prompt text details = %d, want 3", usage.PromptTokensDetails.TextTokens)
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
	thinking := "thinking"
	resp := ResponseOpenAI2Gemini(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{{
			Index: 0,
			Message: dto.Message{
				Role:               "assistant",
				Content:            "answer",
				ReasoningContent:   &thinking,
				ReasoningSignature: "sig_123",
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
	if resp.Candidates[0].Content.Parts[0].GetThoughtSignature() != "sig_123" {
		t.Fatalf("thought signature = %q, want sig_123", resp.Candidates[0].Content.Parts[0].GetThoughtSignature())
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

func TestStreamResponseOpenAI2GeminiBuffersToolCallArguments(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	firstChunk := StreamResponseOpenAI2Gemini(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
				ToolCalls: []dto.ToolCallResponse{{
					Index: common.GetPointer(0),
					ID:    "call_1",
					Type:  "function",
					Function: dto.FunctionResponse{
						Name:      "weather",
						Arguments: `{"city":"Shang`,
					},
				}},
			},
		}},
	}, info)

	if firstChunk != nil {
		t.Fatalf("first chunk = %+v, want nil until tool arguments become valid JSON", firstChunk)
	}

	finishReason := "tool_calls"
	secondChunk := StreamResponseOpenAI2Gemini(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
				ToolCalls: []dto.ToolCallResponse{{
					Index: common.GetPointer(0),
					ID:    "call_1",
					Type:  "function",
					Function: dto.FunctionResponse{
						Arguments: `hai"}`,
					},
				}},
			},
			FinishReason: &finishReason,
		}},
	}, info)

	if secondChunk == nil || len(secondChunk.Candidates) != 1 || len(secondChunk.Candidates[0].Content.Parts) != 1 {
		t.Fatalf("second chunk = %+v, want one emitted function call", secondChunk)
	}
	part := secondChunk.Candidates[0].Content.Parts[0]
	if part.FunctionCall == nil || part.FunctionCall.FunctionName != "weather" {
		t.Fatalf("function call = %+v, want weather call", part.FunctionCall)
	}
	args, ok := part.FunctionCall.Arguments.(map[string]interface{})
	if !ok || args["city"] != "Shanghai" {
		t.Fatalf("function call args = %#v, want {city: Shanghai}", part.FunctionCall.Arguments)
	}
}
