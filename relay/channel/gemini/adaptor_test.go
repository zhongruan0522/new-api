package gemini

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func TestAdaptorConvertClaudeRequestPreservesThinkingToolChoiceAndToolResults(t *testing.T) {
	request := &dto.ClaudeRequest{
		Model: "claude-3-7-sonnet",
		Thinking: &dto.Thinking{
			Type:         "enabled",
			BudgetTokens: common.GetPointer(4096),
		},
		ToolChoice: &dto.ClaudeToolChoice{
			Type: "tool",
			Name: "weather",
		},
		Messages: []dto.ClaudeMessage{
			{
				Role: "assistant",
				Content: []dto.ClaudeMediaMessage{
					{Type: "thinking", Thinking: common.GetPointer[string]("plan"), Signature: "sig_123"},
					{Type: "tool_use", Id: "call_1", Name: "weather", Input: map[string]any{"city": "Shanghai"}},
				},
			},
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "tool_result", ToolUseId: "call_1", Content: "{\"temp\":\"20\"}"},
				},
			},
		},
	}
	request.System = []dto.ClaudeMediaMessage{{Type: "text", Text: common.GetPointer[string]("follow the system")}}
	request.AddTool(&dto.Tool{Name: "weather", InputSchema: map[string]interface{}{"type": "object"}})

	adaptor := &Adaptor{}
	convertedAny, err := adaptor.ConvertClaudeRequest(nil, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-2.5-pro"}}, request)
	if err != nil {
		t.Fatalf("ConvertClaudeRequest error = %v", err)
	}

	converted, ok := convertedAny.(*dto.GeminiChatRequest)
	if !ok {
		t.Fatalf("converted request type = %T, want *dto.GeminiChatRequest", convertedAny)
	}
	if converted.GenerationConfig.ThinkingConfig == nil || converted.GenerationConfig.ThinkingConfig.ThinkingBudget == nil || *converted.GenerationConfig.ThinkingConfig.ThinkingBudget != 4096 {
		t.Fatalf("thinking config = %+v, want budget 4096", converted.GenerationConfig.ThinkingConfig)
	}
	if !converted.GenerationConfig.ThinkingConfig.IncludeThoughts {
		t.Fatal("expected includeThoughts to be enabled")
	}
	if converted.ToolConfig == nil || converted.ToolConfig.FunctionCallingConfig == nil {
		t.Fatalf("tool config = %+v, want function calling config", converted.ToolConfig)
	}
	if converted.ToolConfig.FunctionCallingConfig.Mode != dto.FunctionCallingConfigMode("ANY") || len(converted.ToolConfig.FunctionCallingConfig.AllowedFunctionNames) != 1 || converted.ToolConfig.FunctionCallingConfig.AllowedFunctionNames[0] != "weather" {
		t.Fatalf("tool config = %+v, want ANY weather", converted.ToolConfig.FunctionCallingConfig)
	}
	if converted.SystemInstructions == nil || len(converted.SystemInstructions.Parts) != 1 || converted.SystemInstructions.Parts[0].Text != "follow the system" {
		t.Fatalf("system instructions = %+v, want one system text part", converted.SystemInstructions)
	}
	if len(converted.Contents) != 2 {
		t.Fatalf("contents len = %d, want 2", len(converted.Contents))
	}

	modelParts := converted.Contents[0].Parts
	if len(modelParts) < 2 {
		t.Fatalf("model parts len = %d, want at least 2", len(modelParts))
	}
	if !modelParts[0].Thought || modelParts[0].Text != "plan" || modelParts[0].GetThoughtSignature() != "sig_123" {
		t.Fatalf("first model part = %+v, want thought plan with sig_123", modelParts[0])
	}
	if modelParts[1].FunctionCall == nil || modelParts[1].FunctionCall.FunctionName != "weather" {
		t.Fatalf("second model part = %+v, want weather function call", modelParts[1])
	}

	toolParts := converted.Contents[1].Parts
	if len(toolParts) != 1 || toolParts[0].FunctionResponse == nil {
		t.Fatalf("tool response parts = %+v, want one function response", toolParts)
	}
	if toolParts[0].FunctionResponse.GetID() != "call_1" {
		t.Fatalf("function response id = %q, want call_1", toolParts[0].FunctionResponse.GetID())
	}
}
