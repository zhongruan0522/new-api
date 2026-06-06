package common

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

// Test assistant history and json_schema mapping because both are required for
// multi-turn Responses requests to remain valid after Chat -> Responses rewrite.
func TestConvertChatCompletionsRequestToResponsesRequest_AssistantHistoryAndJSONSchema(t *testing.T) {
	chatReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-4.1",
		Messages: []dto.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
		ResponseFormat: &dto.ResponseFormat{
			Type:       "json_schema",
			JsonSchema: []byte(`{"name":"answer_format","schema":{"type":"object","properties":{"answer":{"type":"string"}}},"strict":true}`),
		},
	}

	got, err := ConvertChatCompletionsRequestToResponsesRequest(chatReq)
	if err != nil {
		t.Fatalf("ConvertChatCompletionsRequestToResponsesRequest() error = %v", err)
	}

	var instructions string
	if err := common.Unmarshal(got.Instructions, &instructions); err != nil {
		t.Fatalf("unmarshal instructions error = %v", err)
	}
	if instructions != "You are helpful." {
		t.Fatalf("instructions = %q, want %q", instructions, "You are helpful.")
	}

	var items []map[string]any
	if err := common.Unmarshal(got.Input, &items); err != nil {
		t.Fatalf("unmarshal input error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("input items len = %d, want 2", len(items))
	}

	userParts, ok := items[0]["content"].([]any)
	if !ok || len(userParts) != 1 {
		t.Fatalf("user content = %#v, want one part", items[0]["content"])
	}
	userPart, _ := userParts[0].(map[string]any)
	if userPart["type"] != "input_text" {
		t.Fatalf("user part type = %v, want %q", userPart["type"], "input_text")
	}

	assistantParts, ok := items[1]["content"].([]any)
	if !ok || len(assistantParts) != 1 {
		t.Fatalf("assistant content = %#v, want one part", items[1]["content"])
	}
	assistantPart, _ := assistantParts[0].(map[string]any)
	if assistantPart["type"] != "output_text" {
		t.Fatalf("assistant part type = %v, want %q", assistantPart["type"], "output_text")
	}

	var textPayload map[string]any
	if err := common.Unmarshal(got.Text, &textPayload); err != nil {
		t.Fatalf("unmarshal text payload error = %v", err)
	}
	format, ok := textPayload["format"].(map[string]any)
	if !ok {
		t.Fatalf("text.format type = %T, want object", textPayload["format"])
	}
	if format["type"] != "json_schema" {
		t.Fatalf("text.format.type = %v, want %q", format["type"], "json_schema")
	}
	if format["name"] != "answer_format" {
		t.Fatalf("text.format.name = %v, want %q", format["name"], "answer_format")
	}
	if _, ok := format["schema"].(map[string]any); !ok {
		t.Fatalf("text.format.schema type = %T, want object", format["schema"])
	}
	if format["strict"] != true {
		t.Fatalf("text.format.strict = %v, want true", format["strict"])
	}
	if _, exists := format["json_schema"]; exists {
		t.Fatalf("text.format should be flattened, but json_schema wrapper still exists: %#v", format)
	}
}

// Test Responses -> Chat conversion for assistant output_text history and
// flattened json_schema because upstream Responses clients rely on both.
func TestConvertResponsesRequestToChatCompletionsRequest_AssistantOutputTextAndJSONSchema(t *testing.T) {
	inputRaw, err := common.Marshal([]map[string]any{
		{
			"type": "message",
			"role": "assistant",
			"content": []map[string]any{{
				"type": "output_text",
				"text": "Prior answer",
			}},
		},
	})
	if err != nil {
		t.Fatalf("marshal input error = %v", err)
	}

	textRaw, err := common.Marshal(map[string]any{
		"format": map[string]any{
			"type":   "json_schema",
			"name":   "answer_format",
			"schema": map[string]any{"type": "object"},
			"strict": true,
		},
	})
	if err != nil {
		t.Fatalf("marshal text error = %v", err)
	}

	responsesReq := &dto.OpenAIResponsesRequest{
		Model: "gpt-4.1",
		Input: inputRaw,
		Text:  textRaw,
	}

	got, err := ConvertResponsesRequestToChatCompletionsRequest(responsesReq)
	if err != nil {
		t.Fatalf("ConvertResponsesRequestToChatCompletionsRequest() error = %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(got.Messages))
	}
	if got.Messages[0].Role != "assistant" {
		t.Fatalf("messages[0].role = %q, want %q", got.Messages[0].Role, "assistant")
	}
	if got.Messages[0].StringContent() != "Prior answer" {
		t.Fatalf("messages[0].content = %q, want %q", got.Messages[0].StringContent(), "Prior answer")
	}

	if got.ResponseFormat == nil {
		t.Fatal("response_format is nil, want json_schema")
	}
	if got.ResponseFormat.Type != "json_schema" {
		t.Fatalf("response_format.type = %q, want %q", got.ResponseFormat.Type, "json_schema")
	}

	var schema dto.FormatJsonSchema
	if err := common.Unmarshal(got.ResponseFormat.JsonSchema, &schema); err != nil {
		t.Fatalf("unmarshal response_format.json_schema error = %v", err)
	}
	if schema.Name != "answer_format" {
		t.Fatalf("response_format.json_schema.name = %q, want %q", schema.Name, "answer_format")
	}
	if len(schema.Strict) == 0 {
		t.Fatal("response_format.json_schema.strict is empty, want true")
	}
}

// Test adjacent Responses function_call items because Chat Completions requires
// parallel tool calls to share one assistant message before tool outputs arrive.
func TestConvertResponsesRequestToChatCompletionsRequest_GroupsParallelToolCallsAndToolOutput(t *testing.T) {
	isError := true
	inputRaw, err := common.Marshal([]map[string]any{
		{
			"type": "reasoning",
			"summary": []map[string]any{{
				"type": "summary_text",
				"text": "Need tools",
			}},
		},
		{
			"type":      "function_call",
			"call_id":   "call_weather",
			"name":      "get_weather",
			"arguments": `{"city":"beijing"}`,
		},
		{
			"type":      "function_call",
			"call_id":   "call_time",
			"name":      "get_time",
			"arguments": `{"tz":"Asia/Shanghai"}`,
		},
		{
			"type":               "function_call_output",
			"call_id":            "call_weather",
			"output":             []map[string]any{{"type": "input_text", "text": "sunny"}},
			"tool_call_is_error": isError,
		},
	})
	if err != nil {
		t.Fatalf("marshal input error = %v", err)
	}

	got, err := ConvertResponsesRequestToChatCompletionsRequest(&dto.OpenAIResponsesRequest{Model: "gpt-4.1", Input: inputRaw})
	if err != nil {
		t.Fatalf("ConvertResponsesRequestToChatCompletionsRequest() error = %v", err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(got.Messages))
	}
	assistant := got.Messages[0]
	if assistant.Role != "assistant" {
		t.Fatalf("messages[0].role = %q, want assistant", assistant.Role)
	}
	if assistant.ReasoningContent != "Need tools" {
		t.Fatalf("reasoning_content = %q, want Need tools", assistant.ReasoningContent)
	}
	toolCalls := assistant.ParseToolCalls()
	if len(toolCalls) != 2 {
		t.Fatalf("tool_calls len = %d, want 2", len(toolCalls))
	}
	if toolCalls[0].ID != "call_weather" || toolCalls[1].ID != "call_time" {
		t.Fatalf("tool_call ids = %#v, want call_weather/call_time", toolCalls)
	}
	toolMsg := got.Messages[1]
	if toolMsg.Role != "tool" || toolMsg.ToolCallId != "call_weather" || toolMsg.StringContent() != "sunny" {
		t.Fatalf("tool message = %#v, want call_weather sunny", toolMsg)
	}
	if toolMsg.ToolCallIsError == nil || !*toolMsg.ToolCallIsError {
		t.Fatal("tool_call_is_error was not preserved")
	}
}

// Test custom tools because the current OpenAI Chat schema supports custom
// tool calls and Responses represents them as custom_tool_call items.
func TestConvertChatCompletionsRequestToResponsesRequest_CustomToolHistory(t *testing.T) {
	customToolRaw, err := common.Marshal(map[string]any{
		"name":        "code_exec",
		"description": "run code",
		"format":      map[string]any{"type": "text"},
	})
	if err != nil {
		t.Fatalf("marshal custom tool error = %v", err)
	}
	customCallRaw, err := common.Marshal([]map[string]any{{
		"id":   "call_custom",
		"type": "custom",
		"custom": map[string]any{
			"name":  "code_exec",
			"input": "print(1)",
		},
	}})
	if err != nil {
		t.Fatalf("marshal custom call error = %v", err)
	}

	got, err := ConvertChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "user", Content: "run code"},
			{Role: "assistant", ToolCalls: customCallRaw},
			{Role: "tool", ToolCallId: "call_custom", Content: "ok"},
		},
		Tools: []dto.ToolCallRequest{{Type: dto.CustomType, Custom: customToolRaw}},
	})
	if err != nil {
		t.Fatalf("ConvertChatCompletionsRequestToResponsesRequest() error = %v", err)
	}

	var tools []map[string]any
	if err := common.Unmarshal(got.Tools, &tools); err != nil {
		t.Fatalf("unmarshal tools error = %v", err)
	}
	if len(tools) != 1 || tools[0]["type"] != "custom" || tools[0]["name"] != "code_exec" {
		t.Fatalf("tools = %#v, want one custom code_exec tool", tools)
	}

	var items []map[string]any
	if err := common.Unmarshal(got.Input, &items); err != nil {
		t.Fatalf("unmarshal input error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("input items len = %d, want 3", len(items))
	}
	if items[1]["type"] != "custom_tool_call" || items[1]["call_id"] != "call_custom" || items[1]["input"] != "print(1)" {
		t.Fatalf("custom call item = %#v, want custom_tool_call", items[1])
	}
	if items[2]["type"] != "custom_tool_call_output" || items[2]["call_id"] != "call_custom" || items[2]["output"] != "ok" {
		t.Fatalf("custom output item = %#v, want custom_tool_call_output", items[2])
	}
}
