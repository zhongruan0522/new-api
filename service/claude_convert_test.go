package service

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func TestClaudeToOpenAIRequestPreservesThinkingSignatureAndToolErrors(t *testing.T) {
	toolError := true
	request := dto.ClaudeRequest{
		Model:        "claude-3-7-sonnet",
		OutputConfig: []byte(`{"effort":"max"}`),
		Messages: []dto.ClaudeMessage{
			{
				Role: "assistant",
				Content: []dto.ClaudeMediaMessage{
					{Type: "thinking", Thinking: common.GetPointer[string]("plan"), Signature: "sig_123"},
					{Type: "redacted_thinking", Data: "encrypted_payload"},
					{Type: "text", Text: common.GetPointer[string]("hello")},
					{Type: "tool_use", Id: "call_1", Name: "weather", Input: map[string]any{"city": "Shanghai"}},
				},
			},
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "tool_result", ToolUseId: "call_1", Content: "tool failed", IsError: &toolError},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: "gpt-4o",
		},
	}

	openAIRequest, err := ClaudeToOpenAIRequest(request, info)
	if err != nil {
		t.Fatalf("ClaudeToOpenAIRequest error = %v", err)
	}
	if openAIRequest.ReasoningEffort != "xhigh" {
		t.Fatalf("ReasoningEffort = %q, want %q", openAIRequest.ReasoningEffort, "xhigh")
	}
	if len(openAIRequest.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(openAIRequest.Messages))
	}
	assistant := openAIRequest.Messages[0]
	if assistant.ReasoningContent != "plan" {
		t.Fatalf("ReasoningContent = %q, want %q", assistant.ReasoningContent, "plan")
	}
	if assistant.ReasoningSignature != "sig_123" {
		t.Fatalf("ReasoningSignature = %q, want %q", assistant.ReasoningSignature, "sig_123")
	}
	if assistant.RedactedReasoningContent != "encrypted_payload" {
		t.Fatalf("RedactedReasoningContent = %q, want %q", assistant.RedactedReasoningContent, "encrypted_payload")
	}
	if len(assistant.ParseToolCalls()) != 1 {
		t.Fatalf("assistant tool calls len = %d, want 1", len(assistant.ParseToolCalls()))
	}
	toolMessage := openAIRequest.Messages[1]
	if toolMessage.Role != "tool" {
		t.Fatalf("tool role = %q, want %q", toolMessage.Role, "tool")
	}
	if toolMessage.ToolCallIsError == nil || !*toolMessage.ToolCallIsError {
		t.Fatal("expected tool error flag to be preserved")
	}
	if toolMessage.ToolCallId != "call_1" {
		t.Fatalf("ToolCallId = %q, want %q", toolMessage.ToolCallId, "call_1")
	}
}

func TestStreamResponseOpenAI2ClaudeEmitsThinkingSignatureAfterThinkingDelta(t *testing.T) {
	reasoning := "plan"
	signature := "sig_123"
	info := &relaycommon.RelayInfo{
		SendResponseCount: 2,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{},
	}

	responses := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					ReasoningContent:   &reasoning,
					ReasoningSignature: &signature,
				},
			},
		},
	}, info)

	if len(responses) != 3 {
		t.Fatalf("responses len = %d, want 3", len(responses))
	}
	if responses[0].Type != "content_block_start" || responses[0].ContentBlock == nil || responses[0].ContentBlock.Type != "thinking" {
		t.Fatalf("first response = %+v, want thinking content_block_start", responses[0])
	}
	if responses[1].Delta == nil || responses[1].Delta.Type != "thinking_delta" {
		t.Fatalf("second response = %+v, want thinking_delta", responses[1])
	}
	if responses[2].Delta == nil || responses[2].Delta.Type != "signature_delta" || responses[2].Delta.Signature != "sig_123" {
		t.Fatalf("third response = %+v, want signature_delta(sig_123)", responses[2])
	}
}
