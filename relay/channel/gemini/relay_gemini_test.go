package gemini

import (
	"testing"

	"github.com/zhongruan0522/new-api/dto"
)

func TestStreamResponseGeminiChat2OpenAIPreservesThoughtAndText(t *testing.T) {
	stop := "STOP"
	resp, isStop := streamResponseGeminiChat2OpenAI(&dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{
			Index:        0,
			FinishReason: &stop,
			Content: dto.GeminiChatContent{Parts: []dto.GeminiPart{{
				Text:    "thinking",
				Thought: true,
			}, {
				Text: "answer",
			}}},
		}},
	})

	if !isStop {
		t.Fatal("isStop = false, want true")
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("got %d choices, want 1", len(resp.Choices))
	}
	choice := resp.Choices[0]
	if choice.Delta.GetReasoningContent() != "thinking" {
		t.Fatalf("reasoning content = %q, want thinking", choice.Delta.GetReasoningContent())
	}
	if choice.Delta.GetContentString() != "answer" {
		t.Fatalf("content = %q, want answer", choice.Delta.GetContentString())
	}
	if choice.FinishReason != nil {
		t.Fatalf("finish reason = %v, want nil because STOP is emitted as a separate stop chunk", choice.FinishReason)
	}
}
