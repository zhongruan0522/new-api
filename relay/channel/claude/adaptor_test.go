package claude

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func TestAdaptorConvertGeminiRequestPreservesThinkingAndToolResults(t *testing.T) {
	budget := 2048
	thoughtPart := dto.GeminiPart{Text: "plan", Thought: true}
	thoughtPart.SetThoughtSignature("sig_123")

	toolResponse := &dto.GeminiFunctionResponse{
		Name:     "weather",
		Response: map[string]interface{}{"temp": "20"},
	}
	toolResponse.SetID("call_1")

	request := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role:  "user",
				Parts: []dto.GeminiPart{{Text: "what is the weather"}},
			},
			{
				Role: "model",
				Parts: []dto.GeminiPart{
					thoughtPart,
					{FunctionCall: &dto.FunctionCall{FunctionName: "weather", Arguments: map[string]interface{}{"city": "Shanghai"}}},
				},
			},
			{
				Role:  "user",
				Parts: []dto.GeminiPart{{FunctionResponse: toolResponse}},
			},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ThinkingConfig: &dto.GeminiThinkingConfig{
				IncludeThoughts: true,
				ThinkingBudget:  common.GetPointer(budget),
			},
		},
		ToolConfig: &dto.ToolConfig{
			FunctionCallingConfig: &dto.FunctionCallingConfig{
				Mode:                 dto.FunctionCallingConfigMode("ANY"),
				AllowedFunctionNames: []string{"weather"},
			},
		},
		SystemInstructions: &dto.GeminiChatContent{Parts: []dto.GeminiPart{{Text: "follow the system"}}},
	}
	request.SetTools([]dto.GeminiChatTool{{
		FunctionDeclarations: []dto.FunctionRequest{{
			Name:       "weather",
			Parameters: map[string]interface{}{"type": "object"},
		}},
	}})

	adaptor := &Adaptor{}
	convertedAny, err := adaptor.ConvertGeminiRequest(nil, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "claude-3-7-sonnet"}}, request)
	if err != nil {
		t.Fatalf("ConvertGeminiRequest error = %v", err)
	}

	converted, ok := convertedAny.(*dto.ClaudeRequest)
	if !ok {
		t.Fatalf("converted request type = %T, want *dto.ClaudeRequest", convertedAny)
	}
	if converted.Thinking == nil || converted.Thinking.GetBudgetTokens() != budget {
		t.Fatalf("thinking = %+v, want enabled budget %d", converted.Thinking, budget)
	}

	toolChoice, ok := converted.ToolChoice.(*dto.ClaudeToolChoice)
	if !ok {
		t.Fatalf("tool choice type = %T, want *dto.ClaudeToolChoice", converted.ToolChoice)
	}
	if toolChoice.Type != "tool" || toolChoice.Name != "weather" {
		t.Fatalf("tool choice = %+v, want named weather tool", toolChoice)
	}

	if len(converted.ParseSystem()) != 1 || converted.ParseSystem()[0].GetText() != "follow the system" {
		t.Fatalf("system = %+v, want one system text block", converted.System)
	}
	if len(converted.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(converted.Messages))
	}

	assistantContent, err := converted.Messages[1].ParseContent()
	if err != nil {
		t.Fatalf("parse assistant content error = %v", err)
	}
	if len(assistantContent) < 2 {
		t.Fatalf("assistant content len = %d, want at least 2, content = %+v", len(assistantContent), assistantContent)
	}
	if assistantContent[0].Type != "thinking" || assistantContent[0].Thinking == nil || *assistantContent[0].Thinking != "plan" || assistantContent[0].Signature != "sig_123" {
		t.Fatalf("assistant thinking block = %+v, want thinking plan with sig_123", assistantContent[0])
	}
	if assistantContent[1].Type != "tool_use" || assistantContent[1].Name != "weather" || assistantContent[1].Id != "call_1" {
		t.Fatalf("assistant tool_use = %+v, want weather/call_1", assistantContent[1])
	}

	toolResultContent, err := converted.Messages[2].ParseContent()
	if err != nil {
		t.Fatalf("parse tool result content error = %v", err)
	}
	if len(toolResultContent) != 1 || toolResultContent[0].Type != "tool_result" || toolResultContent[0].ToolUseId != "call_1" {
		t.Fatalf("tool result content = %+v, want tool_result for call_1", toolResultContent)
	}
}
