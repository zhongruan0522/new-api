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

// Test usage detail mapping because cached and reasoning token fields affect
// billing when Chat upstream responses are rewritten to Responses clients.
func TestConvertChatCompletionResponseToResponsesResponse_MapsUsageDetails(t *testing.T) {
	chatResp := &dto.OpenAITextResponse{
		Id:      "chatcmpl_usage",
		Object:  "chat.completion",
		Created: 1700000000,
		Model:   "gpt-4.1",
		Choices: []dto.OpenAITextResponseChoice{{
			Index:        0,
			Message:      dto.Message{Role: "assistant", Content: "ok"},
			FinishReason: "stop",
		}},
		Usage: dto.Usage{
			PromptTokens:     10,
			CompletionTokens: 7,
			TotalTokens:      17,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens: 3,
				TextTokens:   10,
			},
			CompletionTokenDetails: dto.OutputTokenDetails{
				ReasoningTokens: 4,
				TextTokens:      3,
			},
		},
	}

	got, err := ConvertChatCompletionResponseToResponsesResponse(chatResp)
	if err != nil {
		t.Fatalf("ConvertChatCompletionResponseToResponsesResponse() error = %v", err)
	}
	if got.Usage == nil {
		t.Fatal("usage is nil")
	}
	if got.Usage.InputTokens != 10 || got.Usage.OutputTokens != 7 || got.Usage.TotalTokens != 17 {
		t.Fatalf("usage = %+v, want input=10 output=7 total=17", got.Usage)
	}
	if got.Usage.InputTokensDetails == nil || got.Usage.InputTokensDetails.CachedTokens != 3 || got.Usage.InputTokensDetails.TextTokens != 10 {
		t.Fatalf("input token details = %+v, want cached=3 text=10", got.Usage.InputTokensDetails)
	}
	if got.Usage.OutputTokensDetails == nil || got.Usage.OutputTokensDetails.ReasoningTokens != 4 || got.Usage.OutputTokensDetails.TextTokens != 3 {
		t.Fatalf("output token details = %+v, want reasoning=4 text=3", got.Usage.OutputTokensDetails)
	}
}

// Test the inverse usage mapping because Responses upstream usage is converted
// back into Chat usage before quota and downstream response handling.
func TestConvertResponsesResponseToChatCompletionResponse_MapsUsageDetails(t *testing.T) {
	responsesResp := &dto.OpenAIResponsesResponse{
		ID:        "resp_usage",
		Object:    "response",
		CreatedAt: 1700000000,
		Status:    "completed",
		Model:     "gpt-4.1",
		Output: []dto.ResponsesOutput{{
			Type:   "message",
			ID:     "msg_1",
			Status: "completed",
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{{
				Type: "output_text",
				Text: "ok",
			}},
		}},
		Usage: &dto.Usage{
			InputTokens:  11,
			OutputTokens: 8,
			TotalTokens:  19,
			InputTokensDetails: &dto.InputTokenDetails{
				CachedTokens: 5,
				ImageTokens:  2,
			},
			OutputTokensDetails: &dto.OutputTokenDetails{
				ReasoningTokens: 6,
				AudioTokens:     1,
			},
		},
	}

	got, err := ConvertResponsesResponseToChatCompletionResponse(responsesResp)
	if err != nil {
		t.Fatalf("ConvertResponsesResponseToChatCompletionResponse() error = %v", err)
	}
	if got.Usage.PromptTokens != 11 || got.Usage.CompletionTokens != 8 || got.Usage.TotalTokens != 19 {
		t.Fatalf("usage = %+v, want prompt=11 completion=8 total=19", got.Usage)
	}
	if got.Usage.PromptTokensDetails.CachedTokens != 5 || got.Usage.PromptTokensDetails.ImageTokens != 2 {
		t.Fatalf("prompt token details = %+v, want cached=5 image=2", got.Usage.PromptTokensDetails)
	}
	if got.Usage.CompletionTokenDetails.ReasoningTokens != 6 || got.Usage.CompletionTokenDetails.AudioTokens != 1 {
		t.Fatalf("completion token details = %+v, want reasoning=6 audio=1", got.Usage.CompletionTokenDetails)
	}
}

// Test custom tool response mapping because Chat custom calls and Responses
// custom_tool_call items use different field names for the same model action.
func TestConvertChatCompletionResponseToResponsesResponse_CustomToolCall(t *testing.T) {
	chatResp := &dto.OpenAITextResponse{
		Id:      "chatcmpl_custom",
		Object:  "chat.completion",
		Created: 1700000000,
		Model:   "gpt-5",
		Choices: []dto.OpenAITextResponseChoice{{
			Index: 0,
			Message: dto.Message{
				Role:      "assistant",
				Content:   nil,
				ToolCalls: []byte(`[{"id":"call_custom","type":"custom","custom":{"name":"code_exec","input":"print(1)"}}]`),
			},
			FinishReason: "tool_calls",
		}},
	}

	got, err := ConvertChatCompletionResponseToResponsesResponse(chatResp)
	if err != nil {
		t.Fatalf("ConvertChatCompletionResponseToResponsesResponse() error = %v", err)
	}
	if len(got.Output) != 1 {
		t.Fatalf("output len = %d, want 1", len(got.Output))
	}
	if got.Output[0].Type != "custom_tool_call" || got.Output[0].CallId != "call_custom" || got.Output[0].Name != "code_exec" || got.Output[0].Input != "print(1)" {
		t.Fatalf("custom output = %#v, want custom_tool_call", got.Output[0])
	}
}
