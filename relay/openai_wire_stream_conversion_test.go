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
