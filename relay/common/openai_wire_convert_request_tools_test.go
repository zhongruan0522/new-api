package common

import (
	"strings"
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
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

func TestConvertResponsesRequestToChatCompletionsRequest_NamespaceTool(t *testing.T) {
	toolsRaw, err := common.Marshal([]map[string]any{
		{
			"type":        "namespace",
			"name":        "mcp__codex_apps__gmail",
			"description": "Find and reference emails from your inbox.",
			"tools": []map[string]any{
				{
					"type":        "function",
					"name":        "_search_emails",
					"description": "Search Gmail for emails matching a query.",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"query":       map[string]any{"type": "string"},
							"max_results": map[string]any{"type": "integer"},
						},
						"required": []string{"query"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal tools error = %v", err)
	}
	choiceRaw, err := common.Marshal(map[string]any{
		"type":      "function",
		"name":      "_search_emails",
		"namespace": "mcp__codex_apps__gmail",
	})
	if err != nil {
		t.Fatalf("marshal tool_choice error = %v", err)
	}
	inputRaw, err := common.Marshal([]map[string]any{{
		"type":      "function_call",
		"call_id":   "call_gmail",
		"namespace": "mcp__codex_apps__gmail",
		"name":      "_search_emails",
		"arguments": map[string]any{"query": "unread"},
	}})
	if err != nil {
		t.Fatalf("marshal input error = %v", err)
	}

	got, err := ConvertResponsesRequestToChatCompletionsRequest(&dto.OpenAIResponsesRequest{
		Model:      "gpt-5",
		Input:      inputRaw,
		Tools:      toolsRaw,
		ToolChoice: choiceRaw,
	})
	if err != nil {
		t.Fatalf("ConvertResponsesRequestToChatCompletionsRequest() error = %v", err)
	}

	const flattenedName = "mcp__codex_apps__gmail___search_emails"
	if len(got.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(got.Tools))
	}
	if got.Tools[0].Type != "function" || got.Tools[0].Function.Name != flattenedName {
		t.Fatalf("tool = %#v, want flattened function %q", got.Tools[0], flattenedName)
	}
	choice, ok := got.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("tool_choice type = %T, want object", got.ToolChoice)
	}
	fn, ok := choice["function"].(map[string]any)
	if !ok || fn["name"] != flattenedName {
		t.Fatalf("tool_choice = %#v, want function %q", choice, flattenedName)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(got.Messages))
	}
	toolCalls := got.Messages[0].ParseToolCalls()
	if len(toolCalls) != 1 {
		t.Fatalf("tool_calls len = %d, want 1", len(toolCalls))
	}
	if toolCalls[0].Function.Name != flattenedName {
		t.Fatalf("tool call name = %q, want %q", toolCalls[0].Function.Name, flattenedName)
	}
	if toolCalls[0].Function.Arguments != `{"query":"unread"}` {
		t.Fatalf("tool call arguments = %q, want compact JSON", toolCalls[0].Function.Arguments)
	}
}

func TestConvertResponsesRequestToChatCompletionsRequest_LoadedNamespaceTools(t *testing.T) {
	inputRaw, err := common.Marshal([]map[string]any{
		{
			"type":      "tool_search_call",
			"call_id":   "call_tool_search_1",
			"arguments": map[string]any{"query": "Gmail search emails", "limit": 5},
		},
		{
			"type":      "tool_search_output",
			"call_id":   "call_tool_search_1",
			"status":    "completed",
			"execution": "client",
			"tools": []map[string]any{{
				"type":        "namespace",
				"name":        "mcp__codex_apps__gmail",
				"description": "Find and reference emails from your inbox.",
				"tools": []map[string]any{{
					"type":        "function",
					"name":        "_search_emails",
					"description": "Search Gmail for emails matching a query.",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"query":       map[string]any{"type": "string"},
							"max_results": map[string]any{"type": "integer"},
						},
						"required": []string{"query"},
					},
				}},
			}},
		},
		{
			"type":    "message",
			"role":    "user",
			"content": "Search unread inbox mail.",
		},
	})
	if err != nil {
		t.Fatalf("marshal input error = %v", err)
	}
	toolsRaw, err := common.Marshal([]map[string]any{{"type": "tool_search"}})
	if err != nil {
		t.Fatalf("marshal tools error = %v", err)
	}

	got, err := ConvertResponsesRequestToChatCompletionsRequest(&dto.OpenAIResponsesRequest{
		Model: "gpt-5",
		Input: inputRaw,
		Tools: toolsRaw,
	})
	if err != nil {
		t.Fatalf("ConvertResponsesRequestToChatCompletionsRequest() error = %v", err)
	}

	toolNames := make(map[string]bool)
	for _, tool := range got.Tools {
		toolNames[tool.Function.Name] = true
	}
	if !toolNames["tool_search"] {
		t.Fatalf("tools = %#v, missing tool_search", got.Tools)
	}
	if !toolNames["mcp__codex_apps__gmail___search_emails"] {
		t.Fatalf("tools = %#v, missing loaded namespace function", got.Tools)
	}
	if len(got.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(got.Messages))
	}
	searchCalls := got.Messages[0].ParseToolCalls()
	if len(searchCalls) != 1 || searchCalls[0].Function.Name != "tool_search" {
		t.Fatalf("first tool call = %#v, want tool_search", searchCalls)
	}
	if got.Messages[1].Role != "tool" ||
		got.Messages[1].ToolCallId != "call_tool_search_1" ||
		!strings.Contains(got.Messages[1].StringContent(), "mcp__codex_apps__gmail") {
		t.Fatalf("tool_search output message = %#v, want raw loaded tools content", got.Messages[1])
	}
}
