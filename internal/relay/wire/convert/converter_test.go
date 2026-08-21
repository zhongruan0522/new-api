package convert

import (
	"encoding/json"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

func customToolResponsesRequest(t *testing.T) *shared.OpenAIResponsesRequest {
	t.Helper()
	toolsRaw, err := jsonx.Marshal([]map[string]any{{
		"type":        "custom",
		"name":        "shell_exec",
		"description": "execute shell text",
		"format":      map[string]any{"type": "text"},
	}})
	if err != nil {
		t.Fatalf("marshal tools error = %v", err)
	}
	return &shared.OpenAIResponsesRequest{
		Model: "gpt-5",
		Input: []byte(`"run a command"`),
		Tools: toolsRaw,
	}
}

// 请求方向经 Converter 转换后，tool 代理上下文必须被捕获到会话上，
// 供同一调用栈或经 RelayInfo 中转后的响应方向反解。
func TestConverterCapturesToolContextFromResponsesRequest(t *testing.T) {
	cv := NewConverter(shared.OpenAIWireAPIChat, shared.OpenAIWireAPIResponses)

	chatReq, err := cv.ConvertResponsesToChatRequest(customToolResponsesRequest(t))
	if err != nil {
		t.Fatalf("ConvertResponsesToChatRequest() error = %v", err)
	}
	if len(chatReq.Tools) != 1 || chatReq.Tools[0].Function.Name != "shell_exec" {
		t.Fatalf("chat tools = %#v, want function proxy shell_exec", chatReq.Tools)
	}
	if cv.ToolContext == nil {
		t.Fatal("expected converter to capture tool context")
	}
	if !cv.ToolContext.HasCustomToolProxies() {
		t.Fatal("expected captured tool context to hold custom tool proxies")
	}
}

// 响应方向经同一 Converter（上下文回填）把 chat function call 还原为
// responses custom_tool_call；无上下文时不会还原。
func TestConverterRestoresCustomToolCallInResponse(t *testing.T) {
	cv := NewConverter(shared.OpenAIWireAPIChat, shared.OpenAIWireAPIResponses)
	if _, err := cv.ConvertResponsesToChatRequest(customToolResponsesRequest(t)); err != nil {
		t.Fatalf("ConvertResponsesToChatRequest() error = %v", err)
	}

	arguments, err := BuildChatArgumentsForResponsesCustomToolInput("printf 'hi'\n")
	if err != nil {
		t.Fatalf("BuildChatArgumentsForResponsesCustomToolInput() error = %v", err)
	}
	toolCallsRaw, err := json.Marshal([]map[string]any{{
		"id":   "call_shell",
		"type": "function",
		"function": map[string]any{
			"name":      "shell_exec",
			"arguments": arguments,
		},
	}})
	if err != nil {
		t.Fatalf("marshal chat tool calls error = %v", err)
	}

	got, err := cv.ConvertChatToResponsesResponse(&shared.OpenAITextResponse{
		Id:      "chatcmpl_shell",
		Object:  "chat.completion",
		Created: 1700000000,
		Model:   "gpt-5",
		Choices: []shared.OpenAITextResponseChoice{{
			Index: 0,
			Message: shared.Message{
				Role:      "assistant",
				ToolCalls: toolCallsRaw,
			},
			FinishReason: "tool_calls",
		}},
	})
	if err != nil {
		t.Fatalf("ConvertChatToResponsesResponse() error = %v", err)
	}
	if len(got.Output) != 1 {
		t.Fatalf("output len = %d, want 1", len(got.Output))
	}
	if got.Output[0].Type != "custom_tool_call" || got.Output[0].Name != "shell_exec" || got.Output[0].Input != "printf 'hi'\n" {
		t.Fatalf("output item = %#v, want shell_exec custom_tool_call with freeform input", got.Output[0])
	}
}

// 无工具代理上下文时响应方向的转换仍须可用（等价于无上下文原语函数）。
func TestConverterConvertsPlainChatResponseWithoutToolContext(t *testing.T) {
	cv := NewConverter(shared.OpenAIWireAPIResponses, shared.OpenAIWireAPIChat)

	got, err := cv.ConvertResponsesToChatResponse(&shared.OpenAIResponsesResponse{
		ID:     "resp_plain",
		Object: "response",
		Status: "completed",
		Output: []shared.ResponsesOutput{{
			Type: "message",
			Role: "assistant",
			Content: []shared.ResponsesOutputContent{{
				Type: "output_text",
				Text: "hello",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("ConvertResponsesToChatResponse() error = %v", err)
	}
	if got.Id != "resp_plain" {
		t.Fatalf("id = %q, want resp_plain", got.Id)
	}
	if len(got.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(got.Choices))
	}
}

// chat → responses 请求方向的基础委托。
func TestConverterConvertsChatRequestToResponses(t *testing.T) {
	cv := NewConverter(shared.OpenAIWireAPIResponses, shared.OpenAIWireAPIChat)

	got, err := cv.ConvertChatToResponsesRequest(&shared.GeneralOpenAIRequest{
		Model:    "gpt-5",
		Messages: []shared.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ConvertChatToResponsesRequest() error = %v", err)
	}
	if got.Model != "gpt-5" {
		t.Fatalf("model = %q, want gpt-5", got.Model)
	}
	if cv.ToolContext != nil {
		t.Fatal("chat→responses request conversion must not create a tool context")
	}
}
