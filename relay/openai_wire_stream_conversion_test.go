package relay

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/dto"
	relaycommon "github.com/NookMux/NookMux/relay/common"
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

// Test text whitespace preservation because markdown, code blocks and tables
// rely on standalone newlines/indentation that may arrive as their own chunks.
func TestResponsesToChatStreamConverter_PreservesTextWhitespaceDeltas(t *testing.T) {
	converter := newResponsesToChatStreamConverter(false)
	deltas := []string{"| A | B |", "\n", "| - | - |", "\n", "| 1 | 2 |"}
	var out strings.Builder
	for _, delta := range deltas {
		event, err := common.Marshal(dto.ResponsesStreamResponse{
			Type:  "response.output_text.delta",
			Delta: delta,
		})
		if err != nil {
			t.Fatalf("marshal text delta %q error = %v", delta, err)
		}
		frame, err := converter.ConvertFrame("response.output_text.delta", string(event), "event: response.output_text.delta\ndata: "+string(event)+"\n\n")
		if err != nil {
			t.Fatalf("ConvertFrame(%q) error = %v", delta, err)
		}
		out.WriteString(frame)
	}

	got := collectChatStreamContent(t, out.String())
	want := strings.Join(deltas, "")
	if got != want {
		t.Fatalf("chat stream content = %q, want %q", got, want)
	}
}

// Test the inverse streaming rewrite for the same class of formatting-sensitive
// content; conversion must not trim per-chunk text before emitting Responses.
func TestChatToResponsesStreamConverter_PreservesTextWhitespaceDeltas(t *testing.T) {
	converter := newChatToResponsesStreamConverter()
	deltas := []string{"| A | B |", "\n", "| - | - |", "\n", "| 1 | 2 |"}
	var out strings.Builder
	for _, delta := range deltas {
		chunk := dto.ChatCompletionsStreamResponse{
			Id:      "chatcmpl_1",
			Object:  "chat.completion.chunk",
			Created: 1700000000,
			Model:   "gpt-4.1",
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Index: 0,
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: &delta},
			}},
		}
		raw, err := common.Marshal(chunk)
		if err != nil {
			t.Fatalf("marshal chunk %q error = %v", delta, err)
		}
		frame, err := converter.ConvertFrame("", string(raw), "data: "+string(raw)+"\n\n")
		if err != nil {
			t.Fatalf("ConvertFrame(%q) error = %v", delta, err)
		}
		out.WriteString(frame)
	}
	done, err := converter.ConvertFrame("", "[DONE]", "data: [DONE]\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame([DONE]) error = %v", err)
	}
	out.WriteString(done)

	deltaText, doneText, completedText := collectResponsesStreamText(t, out.String())
	want := strings.Join(deltas, "")
	if deltaText != want {
		t.Fatalf("responses stream deltas = %q, want %q", deltaText, want)
	}
	if doneText != want {
		t.Fatalf("responses output_item.done text = %q, want %q", doneText, want)
	}
	if completedText != want {
		t.Fatalf("responses completed text = %q, want %q", completedText, want)
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

// Test name-before-id chunks because Responses clients must not observe a
// placeholder tool-call id that later changes to the model-generated id.
func TestChatToResponsesStreamConverter_WaitsForStableToolCallID(t *testing.T) {
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
					Name:      "get_weather",
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
	if strings.Contains(out, `"type":"function_call"`) || strings.Contains(out, `"id":"call_0"`) {
		t.Fatalf("first output = %q, want no placeholder tool call item", out)
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
	if !strings.Contains(out, `"id":"call_real"`) || strings.Contains(out, `"id":"call_0"`) || !strings.Contains(out, `\"city\":\"beijing\"`) {
		t.Fatalf("second output = %q, want stable real id with buffered arguments", out)
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

// Test name-before-call-id events because Chat clients must not receive a
// tool_call id derived from Responses item_id if call_id arrives later.
func TestResponsesToChatStreamConverter_WaitsForStableCallID(t *testing.T) {
	converter := newResponsesToChatStreamConverter(false)
	addedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.output_item.added",
		Item: &dto.ResponsesOutput{
			Type:   "function_call",
			ID:     "fc_1",
			Name:   "get_weather",
			Status: "in_progress",
		},
		ItemID: "fc_1",
	})
	if err != nil {
		t.Fatalf("marshal added event error = %v", err)
	}
	out, err := converter.ConvertFrame("response.output_item.added", string(addedEvent), "event: response.output_item.added\ndata: "+string(addedEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(added) error = %v", err)
	}
	if out != "" {
		t.Fatalf("added output = %q, want no placeholder item_id tool call", out)
	}

	argsEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:   "response.function_call_arguments.delta",
		ItemID: "fc_1",
		Delta:  `{"city":"bei`,
	})
	if err != nil {
		t.Fatalf("marshal args event error = %v", err)
	}
	out, err = converter.ConvertFrame("response.function_call_arguments.delta", string(argsEvent), "event: response.function_call_arguments.delta\ndata: "+string(argsEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(args) error = %v", err)
	}
	if out != "" {
		t.Fatalf("args output = %q, want buffered args until stable call_id", out)
	}

	doneEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.output_item.done",
		Item: &dto.ResponsesOutput{
			Type:      "function_call",
			ID:        "fc_1",
			CallId:    "call_1",
			Name:      "get_weather",
			Arguments: `{"city":"beijing"}`,
			Status:    "completed",
		},
		ItemID: "fc_1",
	})
	if err != nil {
		t.Fatalf("marshal done event error = %v", err)
	}
	out, err = converter.ConvertFrame("response.output_item.done", string(doneEvent), "event: response.output_item.done\ndata: "+string(doneEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(done) error = %v", err)
	}
	if !strings.Contains(out, `"id":"call_1"`) || strings.Contains(out, `"id":"fc_1"`) || !strings.Contains(out, `\"city\":\"bei`) {
		t.Fatalf("done output = %q, want stable call_id with buffered arguments", out)
	}
}

func TestResponsesToChatStreamConverter_UsesCompletedResponseCallIDForPendingToolCall(t *testing.T) {
	converter := newResponsesToChatStreamConverter(false)
	addedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.output_item.added",
		Item: &dto.ResponsesOutput{
			Type:   "function_call",
			ID:     "fc_1",
			Name:   "get_weather",
			Status: "in_progress",
		},
		ItemID: "fc_1",
	})
	if err != nil {
		t.Fatalf("marshal added event error = %v", err)
	}
	if out, err := converter.ConvertFrame("response.output_item.added", string(addedEvent), "event: response.output_item.added\ndata: "+string(addedEvent)+"\n\n"); err != nil || out != "" {
		t.Fatalf("ConvertFrame(added) = (%q, %v), want buffered", out, err)
	}

	argsEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:   "response.function_call_arguments.delta",
		ItemID: "fc_1",
		Delta:  `{"city":"beijing"}`,
	})
	if err != nil {
		t.Fatalf("marshal args event error = %v", err)
	}
	if out, err := converter.ConvertFrame("response.function_call_arguments.delta", string(argsEvent), "event: response.function_call_arguments.delta\ndata: "+string(argsEvent)+"\n\n"); err != nil || out != "" {
		t.Fatalf("ConvertFrame(args) = (%q, %v), want buffered", out, err)
	}

	completedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.completed",
		Response: &dto.OpenAIResponsesResponse{
			Status: "completed",
			Output: []dto.ResponsesOutput{{
				Type:   "function_call",
				ID:     "fc_1",
				CallId: "call_1",
				Name:   "get_weather",
			}},
		},
	})
	if err != nil {
		t.Fatalf("marshal completed event error = %v", err)
	}
	out, err := converter.ConvertFrame("response.completed", string(completedEvent), "event: response.completed\ndata: "+string(completedEvent)+"\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(completed) error = %v", err)
	}
	if !strings.Contains(out, `"id":"call_1"`) || strings.Contains(out, `"id":"fc_1"`) || !strings.Contains(out, `\"city\":\"beijing\"`) {
		t.Fatalf("completed output = %q, want pending tool call emitted with completed call_id", out)
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

// Test Responses custom tools proxied through Chat functions because freeform
// tools must be restored by type/context, not by a specific tool name.
func TestChatToResponsesStreamConverter_CustomToolProxyInputDone(t *testing.T) {
	toolContext := relaycommon.NewOpenAIWireToolContext()
	toolContext.AddCustomToolProxy("shell_exec", "shell_exec")
	converter := newChatToResponsesStreamConverter(toolContext)

	arguments, err := relaycommon.BuildChatArgumentsForResponsesCustomToolInput("printf 'hi'\n")
	if err != nil {
		t.Fatalf("build custom tool arguments error = %v", err)
	}
	split := strings.Index(arguments, ":") + 1
	if split <= 0 || split >= len(arguments) {
		t.Fatalf("unexpected generated arguments = %q", arguments)
	}

	var combined strings.Builder
	convertChunk := func(label string, chunk dto.ChatCompletionsStreamResponse) {
		t.Helper()
		raw, marshalErr := common.Marshal(chunk)
		if marshalErr != nil {
			t.Fatalf("marshal %s chunk error = %v", label, marshalErr)
		}
		out, convertErr := converter.ConvertFrame("", string(raw), "data: "+string(raw)+"\n\n")
		if convertErr != nil {
			t.Fatalf("ConvertFrame(%s) error = %v", label, convertErr)
		}
		combined.WriteString(out)
	}

	chunk := dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl_proxy",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Model:   "gpt-5",
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Index: 0,
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{{
				Index: common.GetPointer(0),
				ID:    "call_shell",
				Type:  "function",
				Function: dto.FunctionResponse{
					Name: "shell_exec",
				},
			}}},
		}},
	}
	convertChunk("name", chunk)

	chunk.Choices[0].Delta.ToolCalls = []dto.ToolCallResponse{{
		Index: common.GetPointer(0),
		Type:  "function",
		Function: dto.FunctionResponse{
			Arguments: arguments[:split],
		},
	}}
	convertChunk("first arguments", chunk)

	chunk.Choices[0].Delta.ToolCalls = []dto.ToolCallResponse{{
		Index: common.GetPointer(0),
		Type:  "function",
		Function: dto.FunctionResponse{
			Arguments: arguments[split:],
		},
	}}
	convertChunk("final arguments", chunk)

	done, err := converter.ConvertFrame("", "[DONE]", "data: [DONE]\n\n")
	if err != nil {
		t.Fatalf("ConvertFrame(done) error = %v", err)
	}
	combined.WriteString(done)
	out := combined.String()

	if !strings.Contains(out, `"type":"custom_tool_call"`) || !strings.Contains(out, `"name":"shell_exec"`) {
		t.Fatalf("converted output = %q, want shell_exec custom_tool_call", out)
	}
	if !strings.Contains(out, "event: response.custom_tool_call_input.delta") || !strings.Contains(out, `"delta":"printf 'hi'\n"`) {
		t.Fatalf("converted output = %q, want full custom input delta", out)
	}
	if !strings.Contains(out, "event: response.custom_tool_call_input.done") || !strings.Contains(out, `"input":"printf 'hi'\n"`) {
		t.Fatalf("converted output = %q, want custom input done", out)
	}
	if strings.Contains(out, "response.function_call_arguments.delta") || strings.Contains(out, "response.function_call_arguments.done") {
		t.Fatalf("converted output = %q, want no function argument events for custom proxy", out)
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

func collectChatStreamContent(t *testing.T, s string) string {
	t.Helper()
	var builder strings.Builder
	for _, frame := range splitSSEFramesForTest(s) {
		_, data, _, err := parseSSEFrame(frame)
		if err != nil {
			t.Fatalf("parse chat frame %q error = %v", frame, err)
		}
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		if err := common.UnmarshalJsonStr(data, &chunk); err != nil {
			t.Fatalf("unmarshal chat chunk %q error = %v", data, err)
		}
		for _, choice := range chunk.Choices {
			builder.WriteString(choice.Delta.GetContentString())
		}
	}
	return builder.String()
}

func collectResponsesStreamText(t *testing.T, s string) (deltaText string, doneText string, completedText string) {
	t.Helper()
	var deltaBuilder strings.Builder
	var doneBuilder strings.Builder
	var completedBuilder strings.Builder
	for _, frame := range splitSSEFramesForTest(s) {
		event, data, _, err := parseSSEFrame(frame)
		if err != nil {
			t.Fatalf("parse responses frame %q error = %v", frame, err)
		}
		if data == "" || data == "[DONE]" {
			continue
		}
		var stream dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &stream); err != nil {
			t.Fatalf("unmarshal responses frame %q error = %v", data, err)
		}
		eventType := stream.Type
		if eventType == "" {
			eventType = event
		}
		switch eventType {
		case "response.output_text.delta":
			deltaBuilder.WriteString(stream.Delta)
		case "response.output_item.done":
			if stream.Item != nil {
				appendResponsesMessageText(&doneBuilder, []dto.ResponsesOutput{*stream.Item})
			}
		case "response.completed":
			if stream.Response == nil {
				continue
			}
			appendResponsesMessageText(&completedBuilder, stream.Response.Output)
		}
	}
	return deltaBuilder.String(), doneBuilder.String(), completedBuilder.String()
}

func appendResponsesMessageText(builder *strings.Builder, outputs []dto.ResponsesOutput) {
	for _, output := range outputs {
		if output.Type != "message" {
			continue
		}
		for _, part := range output.Content {
			if part.Type == "output_text" {
				builder.WriteString(part.Text)
			}
		}
	}
}

func splitSSEFramesForTest(s string) []string {
	parts := strings.Split(s, "\n\n")
	frames := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		frames = append(frames, part+"\n\n")
	}
	return frames
}
