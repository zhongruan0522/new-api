package common

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func TestConvertChatCompletionsRequestToResponsesRequest_Tools(t *testing.T) {
	chatReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-4o",
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
		},
		Tools: []dto.ToolCallRequest{
			{
				Type: "function",
				Function: dto.FunctionRequest{
					Name:        "get_weather",
					Description: "get weather",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
		ToolChoice: map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": "get_weather",
			},
		},
	}

	got, err := ConvertChatCompletionsRequestToResponsesRequest(chatReq)
	if err != nil {
		t.Fatalf("ConvertChatCompletionsRequestToResponsesRequest() error = %v", err)
	}

	var tools []openAIResponsesFunctionTool
	if err := common.Unmarshal(got.Tools, &tools); err != nil {
		t.Fatalf("unmarshal tools error = %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}
	if tools[0].Name != "get_weather" {
		t.Fatalf("tools[0].name = %q, want %q", tools[0].Name, "get_weather")
	}

	var toolChoice map[string]any
	if err := common.Unmarshal(got.ToolChoice, &toolChoice); err != nil {
		t.Fatalf("unmarshal tool_choice error = %v", err)
	}
	if toolChoice["type"] != "function" {
		t.Fatalf("tool_choice.type = %v, want %q", toolChoice["type"], "function")
	}
	if toolChoice["name"] != "get_weather" {
		t.Fatalf("tool_choice.name = %v, want %q", toolChoice["name"], "get_weather")
	}
}

func TestConvertResponsesRequestToChatCompletionsRequest_Tools(t *testing.T) {
	toolsRaw, err := common.Marshal([]openAIResponsesFunctionTool{
		{
			Type:        "function",
			Name:        "get_weather",
			Description: "get weather",
			Parameters:  []byte(`{"type":"object"}`),
		},
	})
	if err != nil {
		t.Fatalf("marshal tools error = %v", err)
	}
	choiceRaw, err := common.Marshal(map[string]any{"type": "function", "name": "get_weather"})
	if err != nil {
		t.Fatalf("marshal tool_choice error = %v", err)
	}
	inputRaw, err := common.Marshal("hi")
	if err != nil {
		t.Fatalf("marshal input error = %v", err)
	}

	responsesReq := &dto.OpenAIResponsesRequest{
		Model:      "gpt-4o",
		Input:      inputRaw,
		Tools:      toolsRaw,
		ToolChoice: choiceRaw,
	}

	got, err := ConvertResponsesRequestToChatCompletionsRequest(responsesReq)
	if err != nil {
		t.Fatalf("ConvertResponsesRequestToChatCompletionsRequest() error = %v", err)
	}

	if len(got.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(got.Tools))
	}
	if got.Tools[0].Function.Name != "get_weather" {
		t.Fatalf("tools[0].function.name = %q, want %q", got.Tools[0].Function.Name, "get_weather")
	}

	choice, ok := got.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("tool_choice type = %T, want object", got.ToolChoice)
	}
	if choice["type"] != "function" {
		t.Fatalf("tool_choice.type = %v, want %q", choice["type"], "function")
	}
	fn, ok := choice["function"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice.function type = %T, want object", choice["function"])
	}
	if fn["name"] != "get_weather" {
		t.Fatalf("tool_choice.function.name = %v, want %q", fn["name"], "get_weather")
	}
}
