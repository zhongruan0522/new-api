package claude

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

func TestStreamResponseClaude2OpenAIPreservesSignatureAndRedactedThinking(t *testing.T) {
	resp := StreamResponseClaude2OpenAI(&dto.ClaudeResponse{
		Type: "content_block_delta",
		Delta: &dto.ClaudeMediaMessage{
			Type:      "signature_delta",
			Signature: "sig_123",
		},
	})
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Delta.ReasoningSignature == nil {
		t.Fatal("expected reasoning signature in OpenAI chunk")
	}
	if *resp.Choices[0].Delta.ReasoningSignature != "sig_123" {
		t.Fatalf("ReasoningSignature = %q, want %q", *resp.Choices[0].Delta.ReasoningSignature, "sig_123")
	}

	resp = StreamResponseClaude2OpenAI(&dto.ClaudeResponse{
		Type: "content_block_start",
		ContentBlock: &dto.ClaudeMediaMessage{
			Type: "redacted_thinking",
			Data: "encrypted_payload",
		},
	})
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Delta.RedactedReasoningContent == nil {
		t.Fatal("expected redacted reasoning content in OpenAI chunk")
	}
	if *resp.Choices[0].Delta.RedactedReasoningContent != "encrypted_payload" {
		t.Fatalf("RedactedReasoningContent = %q, want %q", *resp.Choices[0].Delta.RedactedReasoningContent, "encrypted_payload")
	}
}

func TestResponseClaude2OpenAIAggregatesThinkingTextAndTools(t *testing.T) {
	resp := ResponseClaude2OpenAI(&dto.ClaudeResponse{
		Id:         "msg_1",
		Model:      "claude-3-7-sonnet",
		StopReason: "tool_use",
		Usage: &dto.ClaudeUsage{
			InputTokens:              100,
			CacheReadInputTokens:     30,
			CacheCreationInputTokens: 50,
			OutputTokens:             20,
		},
		Content: []dto.ClaudeMediaMessage{
			{Type: "thinking", Thinking: common.GetPointer[string]("plan"), Signature: "sig_123"},
			{Type: "redacted_thinking", Data: "encrypted_payload"},
			{Type: "text", Text: common.GetPointer[string]("hello ")},
			{Type: "text", Text: common.GetPointer[string]("world")},
			{Type: "tool_use", Id: "call_1", Name: "weather", Input: map[string]any{"city": "Shanghai"}},
		},
	})

	if len(resp.Choices) != 1 {
		t.Fatalf("Choices len = %d, want 1", len(resp.Choices))
	}
	message := resp.Choices[0].Message
	if message.StringContent() != "hello world" {
		t.Fatalf("content = %q, want %q", message.StringContent(), "hello world")
	}
	if message.ReasoningContent != "plan" {
		t.Fatalf("ReasoningContent = %q, want %q", message.ReasoningContent, "plan")
	}
	if message.ReasoningSignature != "sig_123" {
		t.Fatalf("ReasoningSignature = %q, want %q", message.ReasoningSignature, "sig_123")
	}
	if message.RedactedReasoningContent != "encrypted_payload" {
		t.Fatalf("RedactedReasoningContent = %q, want %q", message.RedactedReasoningContent, "encrypted_payload")
	}
	if len(message.ParseToolCalls()) != 1 {
		t.Fatalf("tool calls len = %d, want 1", len(message.ParseToolCalls()))
	}
	if resp.Usage.PromptTokens != 180 || resp.Usage.TotalTokens != 200 {
		t.Fatalf("usage = %+v, want prompt=180 total=200", resp.Usage)
	}
}
