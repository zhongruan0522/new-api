package controller

import (
	"strings"
	"testing"
)

func TestBuildChannelTestPromptLength(t *testing.T) {
	for i := 0; i < 50; i++ {
		prompt := buildChannelTestPrompt("openai", "gpt-4o-mini", false)
		if !prompt.requiresTextAnswer {
			t.Fatalf("expected text answer validation for chat endpoint")
		}
		length := len([]rune(prompt.prompt))
		if length < 50 || length > 100 {
			t.Fatalf("prompt length %d out of range [50,100]: %q", length, prompt.prompt)
		}
		if !strings.Contains(prompt.prompt, "final integer") {
			t.Fatalf("prompt missing final-integer instruction: %q", prompt.prompt)
		}
	}
}

func TestMatchesChannelTestExpectedAnswer(t *testing.T) {
	if !matchesChannelTestExpectedAnswer("2", 2) {
		t.Fatal("expected plain integer to match")
	}
	if !matchesChannelTestExpectedAnswer(" 2\n", 2) {
		t.Fatal("expected trimmed integer to match")
	}
	if matchesChannelTestExpectedAnswer("2 is the answer", 2) {
		t.Fatal("expected multi-token text to fail")
	}
	if matchesChannelTestExpectedAnswer("3", 2) {
		t.Fatal("expected wrong integer to fail")
	}
}

func TestExtractChannelTestAITextFromChatResponse(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"2"}}]}`)
	text := extractChannelTestAIText(body)
	if text != "2" {
		t.Fatalf("unexpected extracted text: %q", text)
	}
}

func TestResponseHasChannelTestToolCall(t *testing.T) {
	withTool := []byte(`{"choices":[{"message":{"tool_calls":[{"type":"function","function":{"name":"report_result","arguments":"{\"value\":42}"}}]}}]}`)
	if !responseHasChannelTestToolCall(withTool) {
		t.Fatal("expected tool call to be detected")
	}
	withoutTool := []byte(`{"choices":[{"message":{"content":"42"}}]}`)
	if responseHasChannelTestToolCall(withoutTool) {
		t.Fatal("did not expect tool call detection for plain text")
	}
}

func TestValidateChannelTestResponseToolUnsupported(t *testing.T) {
	err := validateChannelTestResponse(
		[]byte(`{"choices":[{"message":{"content":"hello"}}]}`),
		channelTestPrompt{isTool: true},
	)
	if err == nil {
		t.Fatal("expected tool validation failure")
	}
	if !strings.Contains(err.Error(), channelTestToolNotSupported) {
		t.Fatalf("expected tool unsupported message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "hello") {
		t.Fatalf("expected original AI text in error, got %q", err.Error())
	}
}

func TestValidateChannelTestResponseArithmeticUsesOriginalText(t *testing.T) {
	err := validateChannelTestResponse(
		[]byte(`{"choices":[{"message":{"content":"I think it is four"}}]}`),
		channelTestPrompt{requiresTextAnswer: true, expectedAnswer: 2},
	)
	if err == nil {
		t.Fatal("expected arithmetic validation failure")
	}
	if err.Error() != "I think it is four" {
		t.Fatalf("expected original AI text as error message, got %q", err.Error())
	}
}
