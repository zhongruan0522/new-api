package relay

import (
	"strings"
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

// Test reasoning and incomplete status because Responses streaming clients need
// those fields to keep parity after Responses -> Chat rewriting.
func TestResponsesToChatStreamConverter_ReasoningAndIncompleteStatus(t *testing.T) {
	converter := newResponsesToChatStreamConverter(true)

	reasoningEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:  "response.reasoning_summary_text.delta",
		Delta: "thinking...",
	})
	if err != nil {
		t.Fatalf("marshal reasoning event error = %v", err)
	}

	out, err := converter.ConvertFrame("response.reasoning_summary_text.delta", string(reasoningEvent), "event: response.reasoning_summary_text.delta\ndata: "+string(reasoningEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(reasoning) error = %v", err)
	}
	if !strings.Contains(out, `"reasoning_content":"thinking..."`) {
		t.Fatalf("reasoning chunk = %q, want reasoning_content delta", out)
	}

	completedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.completed",
		Response: &dto.OpenAIResponsesResponse{
			ID:        "resp_1",
			Model:     "gpt-4.1",
			CreatedAt: 1700000000,
			Status:    "incomplete",
			Usage: &dto.Usage{
				InputTokens:  10,
				OutputTokens: 4,
				TotalTokens:  14,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal completed event error = %v", err)
	}

	out, err = converter.ConvertFrame("response.completed", string(completedEvent), "event: response.completed\ndata: "+string(completedEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(completed) error = %v", err)
	}
	if !strings.Contains(out, `"finish_reason":"length"`) {
		t.Fatalf("final chunk = %q, want finish_reason length", out)
	}
	if !strings.Contains(out, `"usage":{"prompt_tokens":10`) {
		t.Fatalf("final chunk = %q, want usage chunk", out)
	}
	if !strings.Contains(out, "data: [DONE]") {
		t.Fatalf("final chunk = %q, want DONE marker", out)
	}
}

// Test reasoning stream rewriting because Chat -> Responses streaming is not
// usable when reasoning deltas are silently dropped.
func TestChatToResponsesStreamConverter_ReasoningDelta(t *testing.T) {
	converter := newChatToResponsesStreamConverter()
	reasoning := "thinking..."
	chunk := dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl_1",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Model:   "gpt-4.1",
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ReasoningContent: &reasoning},
		}},
	}
	raw, err := common.Marshal(chunk)
	if err != nil {
		t.Fatalf("marshal chunk error = %v", err)
	}

	out, err := converter.ConvertFrame("", string(raw), "data: "+string(raw)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame() error = %v", err)
	}
	if !strings.Contains(out, "event: response.reasoning_summary_text.delta") {
		t.Fatalf("converted frame = %q, want reasoning_summary_text.delta", out)
	}
	if !strings.Contains(out, `"delta":"thinking..."`) {
		t.Fatalf("converted frame = %q, want reasoning delta payload", out)
	}
	if !strings.Contains(out, "event: response.created") {
		t.Fatalf("converted frame = %q, want response.created", out)
	}
}

// Test split tool-call metadata because some providers emit arguments before the
// function name lands in a later chunk; emitting the tool call too early drops
// the name and breaks the next request turn.
func TestChatToResponsesStreamConverter_BuffersToolCallUntilNameKnown(t *testing.T) {
	converter := newChatToResponsesStreamConverter()
	firstChunk := dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl_1",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Model:   "gpt-4.1",
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{{
				Index: common.GetPointer(0),
				ID:    "call_1",
				Type:  "function",
				Function: dto.FunctionResponse{
					Arguments: `{"city":"bei`,
				},
			}}},
		}},
	}
	raw, err := common.Marshal(firstChunk)
	if err != nil {
		t.Fatalf("marshal first chunk error = %v", err)
	}
	out, err := converter.ConvertFrame("", string(raw), "data: "+string(raw)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(first) error = %v", err)
	}
	if strings.Contains(out, `"type":"function_call"`) {
		t.Fatalf("first output = %q, want tool call buffered until name is known", out)
	}

	secondChunk := dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl_1",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Model:   "gpt-4.1",
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{{
				Index: common.GetPointer(0),
				ID:    "call_1",
				Type:  "function",
				Function: dto.FunctionResponse{
					Name:      "get_weather",
					Arguments: `jing"}`,
				},
			}}},
		}},
	}
	raw, err = common.Marshal(secondChunk)
	if err != nil {
		t.Fatalf("marshal second chunk error = %v", err)
	}
	out, err = converter.ConvertFrame("", string(raw), "data: "+string(raw)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(second) error = %v", err)
	}
	if !strings.Contains(out, `"type":"function_call"`) || !strings.Contains(out, `"name":"get_weather"`) {
		t.Fatalf("second output = %q, want function_call item with name", out)
	}
	if !strings.Contains(out, `\"city\":\"beijing\"`) {
		t.Fatalf("second output = %q, want buffered full arguments", out)
	}
}

// Test Responses tool-call buffering for the inverse rewrite path because some
// streams surface arguments deltas before the item.added metadata with name.
func TestResponsesToChatStreamConverter_BuffersToolCallUntilNameKnown(t *testing.T) {
	converter := newResponsesToChatStreamConverter(false)
	argsEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:   "response.function_call_arguments.delta",
		ItemID: "call_1",
		Delta:  `{"city":"bei`,
	})
	if err != nil {
		t.Fatalf("marshal args event error = %v", err)
	}
	out, err := converter.ConvertFrame("response.function_call_arguments.delta", string(argsEvent), "event: response.function_call_arguments.delta\ndata: "+string(argsEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(args) error = %v", err)
	}
	if out != "" {
		t.Fatalf("args output = %q, want empty while waiting for name", out)
	}

	addedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.output_item.added",
		Item: &dto.ResponsesOutput{
			Type:   "function_call",
			ID:     "call_1",
			CallId: "call_1",
			Name:   "get_weather",
			Status: "in_progress",
		},
		ItemID: "call_1",
	})
	if err != nil {
		t.Fatalf("marshal added event error = %v", err)
	}
	out, err = converter.ConvertFrame("response.output_item.added", string(addedEvent), "event: response.output_item.added\ndata: "+string(addedEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(added) error = %v", err)
	}
	if !strings.Contains(out, `"name":"get_weather"`) {
		t.Fatalf("added output = %q, want function name", out)
	}
	if !strings.Contains(out, `\"city\":\"bei`) {
		t.Fatalf("added output = %q, want buffered arguments delta", out)
	}
}
