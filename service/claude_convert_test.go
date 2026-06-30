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
	if assistant.ReasoningContent == nil || *assistant.ReasoningContent != "plan" {
		t.Fatalf("ReasoningContent = %v, want %q", assistant.ReasoningContent, "plan")
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

func TestClaudeToOpenAIRequestMapsOpenRouterEnabledThinking(t *testing.T) {
	budget := 2048
	request := dto.ClaudeRequest{
		Model:    "anthropic/claude-sonnet-4",
		Thinking: &dto.Thinking{Type: "enabled", BudgetTokens: &budget},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "anthropic/claude-sonnet-4",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			UpstreamModelName: "anthropic/claude-sonnet-4",
		},
	}

	openAIRequest, err := ClaudeToOpenAIRequest(request, info)
	if err != nil {
		t.Fatalf("ClaudeToOpenAIRequest error = %v", err)
	}

	var reasoning map[string]any
	if err := common.Unmarshal(openAIRequest.Reasoning, &reasoning); err != nil {
		t.Fatalf("unmarshal reasoning: %v", err)
	}
	if reasoning["enabled"] != true || reasoning["max_tokens"].(float64) != 2048 {
		t.Fatalf("reasoning = %#v, want enabled=true max_tokens=2048", reasoning)
	}
	if openAIRequest.ReasoningEffort != "" {
		t.Fatalf("ReasoningEffort = %q, want empty for OpenRouter", openAIRequest.ReasoningEffort)
	}
}

func TestClaudeToOpenAIRequestMapsOpenRouterAdaptiveThinking(t *testing.T) {
	request := dto.ClaudeRequest{
		Model:    "anthropic/claude-sonnet-4",
		Thinking: &dto.Thinking{Type: "adaptive"},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "anthropic/claude-sonnet-4",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			UpstreamModelName: "anthropic/claude-sonnet-4",
		},
	}

	openAIRequest, err := ClaudeToOpenAIRequest(request, info)
	if err != nil {
		t.Fatalf("ClaudeToOpenAIRequest error = %v", err)
	}

	var reasoning map[string]any
	if err := common.Unmarshal(openAIRequest.Reasoning, &reasoning); err != nil {
		t.Fatalf("unmarshal reasoning: %v", err)
	}
	if reasoning["enabled"] != true || reasoning["effort"] != "high" || len(reasoning) != 2 {
		t.Fatalf("reasoning = %#v, want enabled=true effort=high", reasoning)
	}
}

func TestClaudeToOpenAIRequestMapsOpenRouterOutputConfigEffortToReasoning(t *testing.T) {
	request := dto.ClaudeRequest{
		Model:        "anthropic/claude-sonnet-4",
		OutputConfig: []byte(`{"effort":"high"}`),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "anthropic/claude-sonnet-4",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			UpstreamModelName: "anthropic/claude-sonnet-4",
		},
	}

	openAIRequest, err := ClaudeToOpenAIRequest(request, info)
	if err != nil {
		t.Fatalf("ClaudeToOpenAIRequest error = %v", err)
	}
	var reasoning map[string]any
	if err := common.Unmarshal(openAIRequest.Reasoning, &reasoning); err != nil {
		t.Fatalf("unmarshal reasoning: %v", err)
	}
	if reasoning["enabled"] != true || reasoning["effort"] != "high" || len(reasoning) != 2 {
		t.Fatalf("reasoning = %#v, want enabled=true effort=high", reasoning)
	}
	if len(openAIRequest.Verbosity) != 0 {
		t.Fatalf("Verbosity = %s, want empty for OpenRouter reasoning effort", openAIRequest.Verbosity)
	}
	if openAIRequest.ReasoningEffort != "" {
		t.Fatalf("ReasoningEffort = %q, want empty for OpenRouter", openAIRequest.ReasoningEffort)
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

func TestStreamResponseOpenAI2ClaudeEmitsUsageOnlyFinalMessageDelta(t *testing.T) {
	info := &relaycommon.RelayInfo{
		SendResponseCount: 2,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			FinishReason:     "stop",
			LastMessagesType: relaycommon.LastMessageTypeText,
			Index:            0,
		},
	}
	usage := &dto.Usage{
		PromptTokens:     180,
		CompletionTokens: 53,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         30,
			CachedCreationTokens: 50,
		},
	}

	responses := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{Usage: usage}, info)

	if len(responses) != 3 {
		t.Fatalf("responses len = %d, want content_block_stop + message_delta + message_stop", len(responses))
	}
	if responses[0].Type != "content_block_stop" {
		t.Fatalf("first response type = %q, want content_block_stop", responses[0].Type)
	}
	if responses[1].Type != "message_delta" || responses[1].Usage == nil || responses[1].Delta == nil {
		t.Fatalf("second response = %+v, want message_delta with usage", responses[1])
	}
	if responses[1].Usage.InputTokens != 100 || responses[1].Usage.CacheReadInputTokens != 30 || responses[1].Usage.CacheCreationInputTokens != 50 || responses[1].Usage.OutputTokens != 53 {
		t.Fatalf("claude usage = %+v, want input=100 cache_read=30 cache_creation=50 output=53", responses[1].Usage)
	}
	if responses[1].Usage.CacheCreation == nil || responses[1].Usage.CacheCreation.Ephemeral5mInputTokens != 50 || responses[1].Usage.CacheCreation.Ephemeral1hInputTokens != 0 {
		t.Fatalf("cache creation split = %+v, want aggregate defaulted to 5m", responses[1].Usage.CacheCreation)
	}
	if responses[1].Delta.StopReason == nil || *responses[1].Delta.StopReason != "end_turn" {
		t.Fatalf("stop reason = %v, want end_turn", responses[1].Delta.StopReason)
	}
	if responses[2].Type != "message_stop" {
		t.Fatalf("third response type = %q, want message_stop", responses[2].Type)
	}
	if !info.ClaudeConvertInfo.Done {
		t.Fatal("converter should be marked done after usage-only final chunk")
	}
}

func TestStreamResponseOpenAI2ClaudeDefersStopUntilUsageOnlyChunk(t *testing.T) {
	finish := "stop"
	info := &relaycommon.RelayInfo{
		SendResponseCount: 2,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeText,
			Index:            0,
		},
	}

	first := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{FinishReason: &finish}},
	}, info)
	if len(first) != 0 {
		t.Fatalf("finish-only chunk without usage should be deferred, got %+v", first)
	}
	if info.ClaudeConvertInfo.Done {
		t.Fatal("converter should not be done before final usage arrives")
	}

	usage := &dto.Usage{PromptTokens: 7, CompletionTokens: 3}
	second := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{Usage: usage}, info)
	if len(second) != 3 {
		t.Fatalf("responses len = %d, want deferred close with usage", len(second))
	}
	if second[1].Type != "message_delta" || second[1].Usage == nil || second[1].Usage.InputTokens != 7 || second[1].Usage.OutputTokens != 3 {
		t.Fatalf("message_delta = %+v, want final usage", second[1])
	}
	if second[2].Type != "message_stop" {
		t.Fatalf("last response type = %q, want message_stop", second[2].Type)
	}
}

func TestNormalizeCacheCreationSplitDefaultsRemainderTo5m(t *testing.T) {
	fiveMin, oneHour := NormalizeCacheCreationSplit(50, 10, 20)
	if fiveMin != 30 || oneHour != 20 {
		t.Fatalf("split = %d/%d, want 30/20", fiveMin, oneHour)
	}

	fiveMin, oneHour = NormalizeCacheCreationSplit(50, 0, 0)
	if fiveMin != 50 || oneHour != 0 {
		t.Fatalf("aggregate-only split = %d/%d, want 50/0", fiveMin, oneHour)
	}
}
