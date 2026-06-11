package claude

import (
	"encoding/json"
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/types"
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

func TestAdaptorConvertOpenAIResponsesRequestUsesSharedRulesForTools(t *testing.T) {
	toolsRaw, err := common.Marshal([]map[string]any{{
		"type":        "function",
		"name":        "weather",
		"description": "Get weather",
		"parameters": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{"type": "string"},
			},
		},
	}})
	if err != nil {
		t.Fatalf("marshal tools error = %v", err)
	}
	inputRaw, err := common.Marshal([]map[string]any{
		{
			"type":    "message",
			"role":    "user",
			"content": "weather in Shanghai?",
		},
		{
			"type":      "function_call",
			"call_id":   "call_weather",
			"name":      "weather",
			"arguments": json.RawMessage(`{"city":"Shanghai"}`),
		},
	})
	if err != nil {
		t.Fatalf("marshal input error = %v", err)
	}

	info := &relaycommon.RelayInfo{
		RelayFormat:            types.RelayFormatOpenAIResponses,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAIResponses},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       1,
			UpstreamModelName: "claude-3-7-sonnet",
		},
	}
	convertedAny, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:           "claude-3-7-sonnet",
		Input:           inputRaw,
		Tools:           toolsRaw,
		MaxOutputTokens: 1024,
	})
	if err != nil {
		t.Fatalf("ConvertOpenAIResponsesRequest error = %v", err)
	}
	converted, ok := convertedAny.(*dto.ClaudeRequest)
	if !ok {
		t.Fatalf("converted type = %T, want *dto.ClaudeRequest", convertedAny)
	}
	if info.OpenAIResponsesToolContext == nil {
		t.Fatal("OpenAIResponsesToolContext is nil")
	}
	if got, want := info.RequestConversionChain, []types.RelayFormat{types.RelayFormatOpenAIResponses, types.RelayFormatOpenAI, types.RelayFormatClaude}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("RequestConversionChain = %#v, want %#v", got, want)
	}
	if converted.MaxTokens != 1024 {
		t.Fatalf("MaxTokens = %d, want 1024", converted.MaxTokens)
	}
	if len(converted.GetTools()) != 1 {
		t.Fatalf("tools len = %d, want 1", len(converted.GetTools()))
	}
	if len(converted.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(converted.Messages))
	}
	content, err := converted.Messages[1].ParseContent()
	if err != nil {
		t.Fatalf("parse assistant content error = %v", err)
	}
	if len(content) != 1 || content[0].Type != "tool_use" || content[0].Id != "call_weather" || content[0].Name != "weather" {
		t.Fatalf("assistant content = %+v, want weather tool_use", content)
	}
}

func TestAdaptorConvertOpenAIRequestClaudeEffortToolCallThinkingEnabled(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAnthropic,
			UpstreamModelName: "claude-opus-4-6-high",
		},
	}

	convertedAny, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, buildOpenAIWeatherToolRequest("claude-opus-4-6-high", ""))
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error = %v", err)
	}
	converted, ok := convertedAny.(*dto.ClaudeRequest)
	if !ok {
		t.Fatalf("converted type = %T, want *dto.ClaudeRequest", convertedAny)
	}

	if converted.Model != "claude-opus-4-6" {
		t.Fatalf("model = %q, want claude-opus-4-6", converted.Model)
	}
	if converted.Thinking == nil || converted.Thinking.Type != "adaptive" || converted.Thinking.BudgetTokens != nil {
		t.Fatalf("thinking = %+v, want adaptive without budget_tokens", converted.Thinking)
	}
	var outputConfig dto.ClaudeOutputConfig
	if err := common.Unmarshal(converted.OutputConfig, &outputConfig); err != nil {
		t.Fatalf("unmarshal output_config error = %v", err)
	}
	if outputConfig.Effort != "high" {
		t.Fatalf("output_config.effort = %q, want high", outputConfig.Effort)
	}
	if got := converted.GetTools(); len(got) != 1 {
		t.Fatalf("tools len = %d, want 1", len(got))
	}
	if info.ReasoningEffort != "high" {
		t.Fatalf("info.ReasoningEffort = %q, want high", info.ReasoningEffort)
	}
}

func TestAdaptorConvertOpenAIRequestClaudeEffortToolCallThinkingDisabled(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAnthropic,
			UpstreamModelName: "claude-opus-4-6",
		},
	}

	convertedAny, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, buildOpenAIWeatherToolRequest("claude-opus-4-6", "none"))
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error = %v", err)
	}
	converted, ok := convertedAny.(*dto.ClaudeRequest)
	if !ok {
		t.Fatalf("converted type = %T, want *dto.ClaudeRequest", convertedAny)
	}

	if converted.Thinking == nil || converted.Thinking.Type != "disabled" {
		t.Fatalf("thinking = %+v, want disabled", converted.Thinking)
	}
	if len(converted.OutputConfig) != 0 {
		t.Fatalf("output_config = %s, want empty when thinking is disabled", string(converted.OutputConfig))
	}
	if got := converted.GetTools(); len(got) != 1 {
		t.Fatalf("tools len = %d, want 1", len(got))
	}
	if info.ReasoningEffort != "none" {
		t.Fatalf("info.ReasoningEffort = %q, want none", info.ReasoningEffort)
	}
}

func TestAdaptorConvertOpenAIRequestDeepSeekClaudeEffortUsesOutputConfigOnly(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeDeepSeek,
			UpstreamModelName: "deepseek-chat",
		},
	}

	convertedAny, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, buildOpenAIWeatherToolRequest("deepseek-chat", "max"))
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error = %v", err)
	}
	converted, ok := convertedAny.(*dto.ClaudeRequest)
	if !ok {
		t.Fatalf("converted type = %T, want *dto.ClaudeRequest", convertedAny)
	}

	if converted.Thinking != nil {
		t.Fatalf("thinking = %+v, want nil for DeepSeek Anthropic-compatible effort", converted.Thinking)
	}
	var outputConfig dto.ClaudeOutputConfig
	if err := common.Unmarshal(converted.OutputConfig, &outputConfig); err != nil {
		t.Fatalf("unmarshal output_config error = %v", err)
	}
	if outputConfig.Effort != "max" {
		t.Fatalf("output_config.effort = %q, want max", outputConfig.Effort)
	}
}

func TestAdaptorConvertOpenAIRequestRejectsUnsupportedClaudeReasoningEffort(t *testing.T) {
	_, err := (&Adaptor{}).ConvertOpenAIRequest(nil, &relaycommon.RelayInfo{}, buildOpenAIWeatherToolRequest("claude-3-7-sonnet-20250219", "banana"))
	if err == nil {
		t.Fatal("ConvertOpenAIRequest error is nil, want unsupported reasoning_effort error")
	}
}

func buildOpenAIWeatherToolRequest(model string, reasoningEffort string) *dto.GeneralOpenAIRequest {
	return &dto.GeneralOpenAIRequest{
		Model:           model,
		ReasoningEffort: reasoningEffort,
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "What is the weather in Tokyo? Call the tool if needed.",
			},
		},
		Tools: []dto.ToolCallRequest{
			{
				Type: "function",
				Function: dto.FunctionRequest{
					Name:        "get_current_weather",
					Description: "Get the current weather for a city",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{
								"type":        "string",
								"description": "City and country, for example Tokyo, Japan",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
	}
}
