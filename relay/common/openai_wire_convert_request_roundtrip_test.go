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
