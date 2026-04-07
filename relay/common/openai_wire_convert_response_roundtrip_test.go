package common

import (
	"testing"

	"github.com/zhongruan0522/new-api/dto"
)

// Test reasoning preservation because Responses reasoning items are otherwise
// dropped when clients are rewritten back to ChatCompletions.
func TestConvertResponsesResponseToChatCompletionResponse_PreservesReasoningAndToolCalls(t *testing.T) {
	responsesResp := &dto.OpenAIResponsesResponse{
		ID:        "resp_123",
		Object:    "response",
		CreatedAt: 1700000000,
		Status:    "completed",
		Model:     "gpt-4.1",
		Output: []dto.ResponsesOutput{
			{
				Type:    "reasoning",
				ID:      "rs_1",
				Status:  "completed",
				Summary: []dto.ResponsesContentPart{{Type: "summary_text", Text: "Think first"}},
			},
			{
				Type:      "function_call",
				ID:        "fc_1",
				Status:    "completed",
				CallId:    "call_1",
				Name:      "get_weather",
				Arguments: `{"city":"beijing"}`,
			},
			{
				Type:   "message",
				ID:     "msg_1",
				Status: "completed",
				Role:   "assistant",
				Content: []dto.ResponsesOutputContent{{
					Type: "output_text",
					Text: "Here you go",
				}},
			},
		},
	}

	got, err := ConvertResponsesResponseToChatCompletionResponse(responsesResp)
	if err != nil {
		t.Fatalf("ConvertResponsesResponseToChatCompletionResponse() error = %v", err)
	}

	if len(got.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(got.Choices))
	}
	msg := got.Choices[0].Message
	if msg.ReasoningContent != "Think first" {
		t.Fatalf("message.reasoning_content = %q, want %q", msg.ReasoningContent, "Think first")
	}
	if msg.StringContent() != "Here you go" {
		t.Fatalf("message.content = %q, want %q", msg.StringContent(), "Here you go")
	}
	toolCalls := msg.ParseToolCalls()
	if len(toolCalls) != 1 {
		t.Fatalf("tool_calls len = %d, want 1", len(toolCalls))
	}
	if toolCalls[0].Function.Name != "get_weather" {
		t.Fatalf("tool_calls[0].function.name = %q, want %q", toolCalls[0].Function.Name, "get_weather")
	}
	if got.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("finish_reason = %q, want %q", got.Choices[0].FinishReason, "tool_calls")
	}
}

// Test reasoning preservation in the opposite direction because Chat responses
// lose key gpt-5/o-series state if reasoning is discarded during rewrite.
func TestConvertChatCompletionResponseToResponsesResponse_PreservesReasoning(t *testing.T) {
	chatResp := &dto.OpenAITextResponse{
		Id:      "chatcmpl_123",
		Object:  "chat.completion",
		Created: 1700000000,
		Model:   "gpt-4.1",
		Choices: []dto.OpenAITextResponseChoice{{
			Index: 0,
			Message: dto.Message{
				Role:             "assistant",
				Content:          "Here you go",
				ReasoningContent: "Think first",
			},
			FinishReason: "stop",
		}},
	}

	got, err := ConvertChatCompletionResponseToResponsesResponse(chatResp)
	if err != nil {
		t.Fatalf("ConvertChatCompletionResponseToResponsesResponse() error = %v", err)
	}

	if len(got.Output) != 2 {
		t.Fatalf("output len = %d, want 2", len(got.Output))
	}
	if got.Output[0].Type != "reasoning" {
		t.Fatalf("output[0].type = %q, want %q", got.Output[0].Type, "reasoning")
	}
	if len(got.Output[0].Summary) != 1 || got.Output[0].Summary[0].Text != "Think first" {
		t.Fatalf("output[0].summary = %#v, want reasoning summary", got.Output[0].Summary)
	}
	if got.Output[1].Type != "message" {
		t.Fatalf("output[1].type = %q, want %q", got.Output[1].Type, "message")
	}
}
