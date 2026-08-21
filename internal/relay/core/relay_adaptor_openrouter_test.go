package core

import (
	"encoding/json"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
)

// OpenRouter 原生提供 Anthropic Messages 兼容端点（/api/v1/messages），
// Claude 请求必须直接透传，不允许有损转换为 OpenAI Chat 格式。
func TestGetAdaptorOpenRouterClaudePassthrough(t *testing.T) {
	budgetTokens := 2048
	request := &shared.ClaudeRequest{
		Model:     "anthropic/claude-sonnet-4",
		MaxTokens: 1024,
		Messages: []shared.ClaudeMessage{
			{Role: "user", Content: "你好"},
		},
		Thinking: &shared.Thinking{
			Type:         "enabled",
			BudgetTokens: &budgetTokens,
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			ChannelBaseUrl:    "https://openrouter.ai/api",
			UpstreamModelName: "anthropic/claude-sonnet-4",
		},
		RelayFormat: relayconstant.RelayFormatClaude,
	}

	adaptor := GetAdaptor(constant.APITypeOpenRouter)
	if adaptor == nil {
		t.Fatalf("GetAdaptor(APITypeOpenRouter) = nil")
	}

	converted, err := adaptor.ConvertClaudeRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertClaudeRequest error = %v", err)
	}
	claudeRequest, ok := converted.(*shared.ClaudeRequest)
	if !ok {
		t.Fatalf("ConvertClaudeRequest returned %T, want *shared.ClaudeRequest（原生透传，不转换格式）", converted)
	}
	if claudeRequest != request {
		t.Fatalf("ConvertClaudeRequest 应原样返回 Claude 请求")
	}
	if claudeRequest.Thinking == nil || claudeRequest.Thinking.Type != "enabled" {
		t.Fatalf("透传后 Thinking 应保留原值，got %#v", claudeRequest.Thinking)
	}
}

// OpenRouter 原生提供 Responses 兼容端点（/api/v1/responses），
// Responses 请求应直接透传（不转为 Chat 格式）。
func TestGetAdaptorOpenRouterResponsesPassthrough(t *testing.T) {
	request := shared.OpenAIResponsesRequest{
		Model: "openai/gpt-5",
		Input: json.RawMessage(`"你好"`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			ChannelBaseUrl:    "https://openrouter.ai/api",
			UpstreamModelName: "openai/gpt-5",
		},
		RelayFormat: relayconstant.RelayFormatOpenAIResponses,
		RelayMode:   relayconstant.RelayModeResponses,
	}

	adaptor := GetAdaptor(constant.APITypeOpenRouter)
	if adaptor == nil {
		t.Fatalf("GetAdaptor(APITypeOpenRouter) = nil")
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIResponsesRequest error = %v", err)
	}
	responsesRequest, ok := converted.(shared.OpenAIResponsesRequest)
	if !ok {
		t.Fatalf("ConvertOpenAIResponsesRequest returned %T, want shared.OpenAIResponsesRequest（原生透传）", converted)
	}
	if responsesRequest.Model != "openai/gpt-5" {
		t.Fatalf("透传后 Model = %q, want %q", responsesRequest.Model, "openai/gpt-5")
	}
}
