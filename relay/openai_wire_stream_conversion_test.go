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

// Test index-only tool-call chunks because providers can stream arguments under
// a temporary index before a later chunk reveals the stable call id.
func TestChatToResponsesStreamConverter_RekeysBufferedIndexToToolCallID(t *testing.T) {
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
				ID:    "call_real",
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
	if !strings.Contains(out, `"id":"call_real"`) || !strings.Contains(out, `\"city\":\"beijing\"`) {
		t.Fatalf("second output = %q, want buffered args rekeyed to real call id", out)
	}

	_, err = converter.ConvertFrame("", "[DONE]", "data: [DONE]\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(done) error = %v", err)
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

// Test item_id/call_id mapping because Responses streams identify deltas by
// item_id while Chat tool messages must preserve the model-generated call_id.
func TestResponsesToChatStreamConverter_MapsFunctionCallItemIDToCallID(t *testing.T) {
	converter := newResponsesToChatStreamConverter(false)
	addedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.output_item.added",
		Item: &dto.ResponsesOutput{
			Type:   "function_call",
			ID:     "fc_1",
			CallId: "call_1",
			Name:   "get_weather",
			Status: "in_progress",
		},
		ItemID: "fc_1",
	})
	if err != nil {
		t.Fatalf("marshal added event error = %v", err)
	}
	out, err := converter.ConvertFrame("response.output_item.added", string(addedEvent), "event: response.output_item.added\r\ndata: "+string(addedEvent)+"\r\n\r\n")
	if err != nil {
		t.Fatalf("ConvertFrame(added) error = %v", err)
	}
	if !strings.Contains(out, `"id":"call_1"`) || strings.Contains(out, `"id":"fc_1"`) {
		t.Fatalf("added output = %q, want chat tool_call id call_1", out)
	}

	argsEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:   "response.function_call_arguments.delta",
		ItemID: "fc_1",
		Delta:  `{"city":"beijing"}`,
	})
	if err != nil {
		t.Fatalf("marshal args event error = %v", err)
	}
	out, err = converter.ConvertFrame("response.function_call_arguments.delta", string(argsEvent), "event: response.function_call_arguments.delta\r\ndata: "+string(argsEvent)+"\r\n\r\n")
	if err != nil {
		t.Fatalf("ConvertFrame(args) error = %v", err)
	}
	if !strings.Contains(out, `\"city\":\"beijing\"`) {
		t.Fatalf("args output = %q, want arguments delta for mapped call_id", out)
	}
}

// Test out-of-order item_id mapping because Responses may stream arguments
// before output_item.added reveals the stable call_id.
func TestResponsesToChatStreamConverter_RekeysBufferedItemIDToCallID(t *testing.T) {
	converter := newResponsesToChatStreamConverter(false)
	argsEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:   "response.function_call_arguments.delta",
		ItemID: "fc_1",
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
		t.Fatalf("args output = %q, want empty while waiting for call metadata", out)
	}

	addedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.output_item.added",
		Item: &dto.ResponsesOutput{
			Type:   "function_call",
			ID:     "fc_1",
			CallId: "call_1",
			Name:   "get_weather",
			Status: "in_progress",
		},
		ItemID: "fc_1",
	})
	if err != nil {
		t.Fatalf("marshal added event error = %v", err)
	}
	out, err = converter.ConvertFrame("response.output_item.added", string(addedEvent), "event: response.output_item.added\ndata: "+string(addedEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(added) error = %v", err)
	}
	if !strings.Contains(out, `"id":"call_1"`) || !strings.Contains(out, `\"city\":\"bei`) {
		t.Fatalf("added output = %q, want buffered args rekeyed to call_id", out)
	}
}

// Test streamed custom tool calls because Responses and Chat use different
// event names and field names for custom tool input.
func TestChatToResponsesStreamConverter_CustomToolCallInput(t *testing.T) {
	converter := newChatToResponsesStreamConverter()
	custom, err := common.Marshal(map[string]any{"name": "code_exec", "input": "print("})
	if err != nil {
		t.Fatalf("marshal custom tool call error = %v", err)
	}
	chunk := dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl_1",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Model:   "gpt-4.1",
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{{
				Index:  common.GetPointer(0),
				ID:     "call_custom",
				Type:   dto.CustomType,
				Custom: custom,
			}}},
		}},
	}
	raw, err := common.Marshal(chunk)
	if err != nil {
		t.Fatalf("marshal chunk error = %v", err)
	}
	out, err := converter.ConvertFrame("", string(raw), "data: "+string(raw)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(first) error = %v", err)
	}
	if !strings.Contains(out, `"type":"custom_tool_call"`) || !strings.Contains(out, `"name":"code_exec"`) {
		t.Fatalf("first output = %q, want custom_tool_call item", out)
	}
	if !strings.Contains(out, "event: response.custom_tool_call_input.delta") || !strings.Contains(out, `"delta":"print("`) {
		t.Fatalf("first output = %q, want custom input delta", out)
	}

	custom, err = common.Marshal(map[string]any{"input": "\n    "})
	if err != nil {
		t.Fatalf("marshal second custom tool call error = %v", err)
	}
	chunk.Choices[0].Delta.ToolCalls = []dto.ToolCallResponse{{
		Index:  common.GetPointer(0),
		Type:   dto.CustomType,
		Custom: custom,
	}}
	raw, err = common.Marshal(chunk)
	if err != nil {
		t.Fatalf("marshal second chunk error = %v", err)
	}
	out, err = converter.ConvertFrame("", string(raw), "data: "+string(raw)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(second) error = %v", err)
	}
	if !strings.Contains(out, "event: response.custom_tool_call_input.delta") || !strings.Contains(out, `"delta":"\n    "`) {
		t.Fatalf("second output = %q, want whitespace custom input delta", out)
	}

	custom, err = common.Marshal(map[string]any{"input": "1)"})
	if err != nil {
		t.Fatalf("marshal third custom tool call error = %v", err)
	}
	chunk.Choices[0].Delta.ToolCalls = []dto.ToolCallResponse{{
		Index:  common.GetPointer(0),
		Type:   dto.CustomType,
		Custom: custom,
	}}
	raw, err = common.Marshal(chunk)
	if err != nil {
		t.Fatalf("marshal third chunk error = %v", err)
	}
	out, err = converter.ConvertFrame("", string(raw), "data: "+string(raw)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(third) error = %v", err)
	}
	if !strings.Contains(out, "event: response.custom_tool_call_input.delta") || !strings.Contains(out, `"delta":"1)"`) {
		t.Fatalf("third output = %q, want continued custom input delta", out)
	}

	out, err = converter.ConvertFrame("", "[DONE]", "data: [DONE]\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(done) error = %v", err)
	}
	if !strings.Contains(out, "event: response.custom_tool_call_input.done") || !strings.Contains(out, `"input":"print(\n    1)"`) {
		t.Fatalf("done output = %q, want completed custom input", out)
	}
	if !strings.Contains(out, `"type":"custom_tool_call"`) || !strings.Contains(out, `"input":"print(\n    1)"`) {
		t.Fatalf("done output = %q, want custom_tool_call output item", out)
	}
}

// Test the inverse custom-tool stream mapping because Responses emits custom
// input deltas outside function_call_arguments events.
func TestResponsesToChatStreamConverter_CustomToolCallInput(t *testing.T) {
	converter := newResponsesToChatStreamConverter(false)
	addedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.output_item.added",
		Item: &dto.ResponsesOutput{
			Type:   "custom_tool_call",
			ID:     "ct_1",
			CallId: "call_custom",
			Name:   "code_exec",
			Status: "in_progress",
		},
		ItemID: "ct_1",
	})
	if err != nil {
		t.Fatalf("marshal added event error = %v", err)
	}
	out, err := converter.ConvertFrame("response.output_item.added", string(addedEvent), "event: response.output_item.added\ndata: "+string(addedEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(added) error = %v", err)
	}
	if !strings.Contains(out, `"id":"call_custom"`) || !strings.Contains(out, `"type":"custom"`) || !strings.Contains(out, `"custom":{"name":"code_exec"}`) {
		t.Fatalf("added output = %q, want chat custom tool_call with call_id", out)
	}

	deltaEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:   "response.custom_tool_call_input.delta",
		ItemID: "ct_1",
		Delta:  "print(",
	})
	if err != nil {
		t.Fatalf("marshal delta event error = %v", err)
	}
	out, err = converter.ConvertFrame("response.custom_tool_call_input.delta", string(deltaEvent), "event: response.custom_tool_call_input.delta\ndata: "+string(deltaEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(delta) error = %v", err)
	}
	if !strings.Contains(out, `"type":"custom"`) || !strings.Contains(out, `"custom":{"input":"print("}`) {
		t.Fatalf("delta output = %q, want chat custom input delta", out)
	}

	deltaEvent, err = common.Marshal(dto.ResponsesStreamResponse{
		Type:   "response.custom_tool_call_input.delta",
		ItemID: "ct_1",
		Delta:  "\n    ",
	})
	if err != nil {
		t.Fatalf("marshal whitespace delta event error = %v", err)
	}
	out, err = converter.ConvertFrame("response.custom_tool_call_input.delta", string(deltaEvent), "event: response.custom_tool_call_input.delta\ndata: "+string(deltaEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(whitespace delta) error = %v", err)
	}
	if !strings.Contains(out, `"type":"custom"`) || !strings.Contains(out, `"custom":{"input":"\n    "}`) {
		t.Fatalf("whitespace delta output = %q, want chat custom whitespace delta", out)
	}

	doneEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:   "response.custom_tool_call_input.done",
		ItemID: "ct_1",
		Input:  "print(\n    1)",
	})
	if err != nil {
		t.Fatalf("marshal done event error = %v", err)
	}
	out, err = converter.ConvertFrame("response.custom_tool_call_input.done", string(doneEvent), "event: response.custom_tool_call_input.done\ndata: "+string(doneEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(done) error = %v", err)
	}
	if !strings.Contains(out, `"custom":{"input":"1)"}`) {
		t.Fatalf("done output = %q, want remaining custom input delta", out)
	}
}
