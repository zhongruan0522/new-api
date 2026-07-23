package dto

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/common"
)

// TestMessageExplicitEmptyReasoningSurvivesJSON ensures that a client sending
// an explicit empty reasoning_content (as DeepSeek-reasoner requires for
// multi-turn history) is not silently dropped by omitempty.
func TestMessageExplicitEmptyReasoningSurvivesJSON(t *testing.T) {
	raw := []byte(`{"role":"assistant","content":"ok","reasoning_content":"","reasoning":""}`)

	var msg Message
	if err := common.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if msg.ReasoningContent == nil {
		t.Fatal("reasoning_content was dropped, want a non-nil pointer to \"\"")
	}
	if *msg.ReasoningContent != "" {
		t.Fatalf("reasoning_content = %q, want \"\"", *msg.ReasoningContent)
	}
	if msg.Reasoning == nil {
		t.Fatal("reasoning was dropped, want a non-nil pointer to \"\"")
	}
	if *msg.Reasoning != "" {
		t.Fatalf("reasoning = %q, want \"\"", *msg.Reasoning)
	}

	out, err := common.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error = %v", err)
	}
	body := string(out)
	if !strings.Contains(body, `"reasoning_content":""`) {
		t.Fatalf("marshalled body %q does not preserve reasoning_content:\"\"", body)
	}
	if !strings.Contains(body, `"reasoning":""`) {
		t.Fatalf("marshalled body %q does not preserve reasoning:\"\"", body)
	}
}

// TestMessageAbsentReasoningOmitted ensures that absent reasoning fields stay
// omitted (nil pointer), so the pointer change does not bloat every request.
func TestMessageAbsentReasoningOmitted(t *testing.T) {
	msg := Message{Role: "user", Content: "hi"}

	out, err := common.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error = %v", err)
	}
	body := string(out)
	if strings.Contains(body, "reasoning_content") {
		t.Fatalf("marshalled body %q should omit reasoning_content when absent", body)
	}
	if strings.Contains(body, `"reasoning"`) {
		t.Fatalf("marshalled body %q should omit reasoning when absent", body)
	}
}

// TestMessageGetReasoningContentPrefersReasoningContent mirrors the streaming
// delta accessor semantics so provider converters keep a single source of truth.
func TestMessageGetReasoningContentPrefersReasoningContent(t *testing.T) {
	primary := "primary"
	fallback := "fallback"

	msg := Message{ReasoningContent: &primary, Reasoning: &fallback}
	if got := msg.GetReasoningContent(); got != primary {
		t.Fatalf("GetReasoningContent() = %q, want %q", got, primary)
	}

	msg = Message{Reasoning: &fallback}
	if got := msg.GetReasoningContent(); got != fallback {
		t.Fatalf("GetReasoningContent() fallback = %q, want %q", got, fallback)
	}

	empty := ""
	msg = Message{ReasoningContent: &empty}
	if got := msg.GetReasoningContent(); got != "" {
		t.Fatalf("GetReasoningContent() = %q, want empty string", got)
	}

	msg = Message{}
	if got := msg.GetReasoningContent(); got != "" {
		t.Fatalf("GetReasoningContent() nil = %q, want empty string", got)
	}
}
