package dto

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
)

func TestGeminiChatResponseUnmarshalSnakeCaseUsageMetadata(t *testing.T) {
	raw := []byte(`{
		"candidates": [{
			"index": 0,
			"finish_reason": "STOP",
			"content": {"role": "model", "parts": [{"text": "hello", "thought": true}]}
		}],
		"prompt_feedback": {"block_reason": "SAFETY"},
		"usage_metadata": {
			"prompt_token_count": 11,
			"candidates_token_count": 7,
			"total_token_count": 21,
			"thoughts_token_count": 3,
			"cached_content_token_count": 5,
			"prompt_tokens_details": [{"modality": "TEXT", "token_count": 11}],
			"candidates_tokens_details": [{"modality": "TEXT", "token_count": 7}]
		}
	}`)

	var resp GeminiChatResponse
	if err := common.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}

	if len(resp.Candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(resp.Candidates))
	}
	if resp.Candidates[0].FinishReason == nil || *resp.Candidates[0].FinishReason != "STOP" {
		t.Fatalf("finish reason = %v, want STOP", resp.Candidates[0].FinishReason)
	}
	if resp.PromptFeedback == nil || resp.PromptFeedback.BlockReason == nil || *resp.PromptFeedback.BlockReason != "SAFETY" {
		t.Fatalf("prompt feedback block reason = %v, want SAFETY", resp.PromptFeedback)
	}
	if resp.UsageMetadata.PromptTokenCount != 11 || resp.UsageMetadata.CandidatesTokenCount != 7 || resp.UsageMetadata.TotalTokenCount != 21 {
		t.Fatalf("usage metadata = %+v, want prompt=11 candidates=7 total=21", resp.UsageMetadata)
	}
	if resp.UsageMetadata.ThoughtsTokenCount != 3 || resp.UsageMetadata.CachedContentTokenCount != 5 {
		t.Fatalf("usage metadata = %+v, want thoughts=3 cached=5", resp.UsageMetadata)
	}
	if len(resp.UsageMetadata.PromptTokensDetails) != 1 || resp.UsageMetadata.PromptTokensDetails[0].TokenCount != 11 {
		t.Fatalf("prompt token details = %+v, want one detail with token_count=11", resp.UsageMetadata.PromptTokensDetails)
	}
	if len(resp.UsageMetadata.CandidatesTokensDetails) != 1 || resp.UsageMetadata.CandidatesTokensDetails[0].TokenCount != 7 {
		t.Fatalf("candidate token details = %+v, want one detail with token_count=7", resp.UsageMetadata.CandidatesTokensDetails)
	}
}
