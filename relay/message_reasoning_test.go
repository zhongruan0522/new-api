package relay

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/dto"
)

// TestChatCompletionRequestPreservesExplicitEmptyReasoningContent reproduces the
// DeepSeek-reasoner multi-turn scenario: a client resends prior assistant turns
// carrying an explicit empty reasoning_content placeholder. The relay must keep
// that empty value when re-marshalling the request for the upstream channel,
// otherwise DeepSeek rejects the history.
func TestChatCompletionRequestPreservesExplicitEmptyReasoningContent(t *testing.T) {
	raw := []byte(`{
		"model": "deepseek-reasoner",
		"messages": [
			{"role": "user", "content": "hi"},
			{"role": "assistant", "content": "hello", "reasoning_content": "I thought", "reasoning": "step"},
			{"role": "assistant", "content": "follow up", "reasoning_content": "", "reasoning": ""}
		]
	}`)

	var req dto.GeneralOpenAIRequest
	if err := common.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal request error = %v", err)
	}

	if len(req.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(req.Messages))
	}

	third := req.Messages[2]
	if third.ReasoningContent == nil {
		t.Fatal("third message reasoning_content was dropped, want explicit empty pointer")
	}
	if *third.ReasoningContent != "" {
		t.Fatalf("third reasoning_content = %q, want \"\"", *third.ReasoningContent)
	}
	if third.Reasoning == nil {
		t.Fatal("third message reasoning was dropped, want explicit empty pointer")
	}

	// The re-marshalled upstream body must carry the empty placeholders verbatim,
	// because that is exactly what DeepSeek-reasoner expects in the history.
	out, err := common.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request error = %v", err)
	}
	body := string(out)
	if !strings.Contains(body, `"reasoning_content":""`) {
		t.Fatalf("re-marshalled request lost explicit empty reasoning_content: %s", body)
	}

	// The non-empty prior turn must also survive untouched.
	if !strings.Contains(body, `"reasoning_content":"I thought"`) {
		t.Fatalf("re-marshalled request lost non-empty reasoning_content: %s", body)
	}
}
