package reasoning

import (
	"encoding/json"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

func TestNormalizeEffort(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"high", "high"},
		{"HIGH", "high"},
		{"  medium  ", "medium"},
		{"max", "max"},
		{"xhigh", "xhigh"},
		{"none", "none"},
		{"minimal", "minimal"},
		{"invalid", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := NormalizeEffort(tt.raw)
		if got != tt.want {
			t.Errorf("NormalizeEffort(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestExtractEffortFromOpenAIRequest(t *testing.T) {
	// 1. 顶层 reasoning_effort
	req := &shared.GeneralOpenAIRequest{ReasoningEffort: "high"}
	if e := ExtractEffortFromOpenAIRequest(req); e != "high" {
		t.Fatalf("top-level reasoning_effort: got %q, want high", e)
	}

	// 2. Reasoning JSON with effort
	reasoningJSON, _ := jsonx.Marshal(map[string]string{"effort": "medium"})
	req = &shared.GeneralOpenAIRequest{Reasoning: reasoningJSON}
	if e := ExtractEffortFromOpenAIRequest(req); e != "medium" {
		t.Fatalf("reasoning.effort: got %q, want medium", e)
	}

	// 3. Reasoning enabled but no effort -> auto
	reasoningJSON, _ = jsonx.Marshal(map[string]interface{}{"enabled": true})
	req = &shared.GeneralOpenAIRequest{Reasoning: reasoningJSON}
	if e := ExtractEffortFromOpenAIRequest(req); e != "auto" {
		t.Fatalf("reasoning.enabled: got %q, want auto", e)
	}

	// 4. THINKING with type=enabled -> auto
	thinkingJSON, _ := jsonx.Marshal(shared.Thinking{Type: "enabled", BudgetTokens: intPtr(2048)})
	req = &shared.GeneralOpenAIRequest{THINKING: thinkingJSON}
	if e := ExtractEffortFromOpenAIRequest(req); e != "auto" {
		t.Fatalf("thinking.type=enabled: got %q, want auto", e)
	}

	// 5. THINKING with type=adaptive -> auto
	thinkingJSON, _ = jsonx.Marshal(shared.Thinking{Type: "adaptive"})
	req = &shared.GeneralOpenAIRequest{THINKING: thinkingJSON}
	if e := ExtractEffortFromOpenAIRequest(req); e != "auto" {
		t.Fatalf("thinking.type=adaptive: got %q, want auto", e)
	}

	// 6. EnableThinking=true -> auto
	enableThinkingJSON, _ := json.Marshal(true)
	req = &shared.GeneralOpenAIRequest{EnableThinking: enableThinkingJSON}
	if e := ExtractEffortFromOpenAIRequest(req); e != "auto" {
		t.Fatalf("enable_thinking=true: got %q, want auto", e)
	}

	// 7. No reasoning fields -> empty
	req = &shared.GeneralOpenAIRequest{Model: "gpt-4o"}
	if e := ExtractEffortFromOpenAIRequest(req); e != "" {
		t.Fatalf("no reasoning: got %q, want empty", e)
	}

	// 8. Top-level effort takes priority over Reasoning JSON
	reasoningJSON, _ = jsonx.Marshal(map[string]string{"effort": "low"})
	req = &shared.GeneralOpenAIRequest{ReasoningEffort: "high", Reasoning: reasoningJSON}
	if e := ExtractEffortFromOpenAIRequest(req); e != "high" {
		t.Fatalf("priority: got %q, want high", e)
	}

	// 9. nil request
	if e := ExtractEffortFromOpenAIRequest(nil); e != "" {
		t.Fatalf("nil request: got %q, want empty", e)
	}

	// 10. Invalid effort value is rejected
	req = &shared.GeneralOpenAIRequest{ReasoningEffort: "super"}
	if e := ExtractEffortFromOpenAIRequest(req); e != "" {
		t.Fatalf("invalid effort: got %q, want empty", e)
	}
}

func TestExtractEffortFromOpenAIResponsesRequest(t *testing.T) {
	// 1. reasoning.effort
	req := &shared.OpenAIResponsesRequest{
		Reasoning: &shared.Reasoning{Effort: "high"},
	}
	if e := ExtractEffortFromOpenAIResponsesRequest(req); e != "high" {
		t.Fatalf("reasoning.effort: got %q, want high", e)
	}

	// 2. enable_thinking=true -> auto
	enableThinkingJSON, _ := json.Marshal(true)
	req = &shared.OpenAIResponsesRequest{EnableThinking: enableThinkingJSON}
	if e := ExtractEffortFromOpenAIResponsesRequest(req); e != "auto" {
		t.Fatalf("enable_thinking=true: got %q, want auto", e)
	}

	// 3. No reasoning
	req = &shared.OpenAIResponsesRequest{}
	if e := ExtractEffortFromOpenAIResponsesRequest(req); e != "" {
		t.Fatalf("no reasoning: got %q, want empty", e)
	}
}

func TestExtractEffortFromClaudeRequest(t *testing.T) {
	// 1. output_config.effort
	outputConfig, _ := jsonx.Marshal(shared.ClaudeOutputConfig{Effort: "medium"})
	req := &shared.ClaudeRequest{OutputConfig: outputConfig}
	if e := ExtractEffortFromClaudeRequest(req); e != "medium" {
		t.Fatalf("output_config.effort=medium: got %q, want medium", e)
	}

	// 2. output_config.effort=max -> xhigh (本项目统一映射)
	outputConfig, _ = jsonx.Marshal(shared.ClaudeOutputConfig{Effort: "max"})
	req = &shared.ClaudeRequest{OutputConfig: outputConfig}
	if e := ExtractEffortFromClaudeRequest(req); e != "xhigh" {
		t.Fatalf("output_config.effort=max: got %q, want xhigh", e)
	}

	// 3. thinking.type=enabled -> auto
	req = &shared.ClaudeRequest{Thinking: &shared.Thinking{Type: "enabled", BudgetTokens: intPtr(2048)}}
	if e := ExtractEffortFromClaudeRequest(req); e != "auto" {
		t.Fatalf("thinking.type=enabled: got %q, want auto", e)
	}

	// 4. thinking.type=adaptive -> auto
	req = &shared.ClaudeRequest{Thinking: &shared.Thinking{Type: "adaptive"}}
	if e := ExtractEffortFromClaudeRequest(req); e != "auto" {
		t.Fatalf("thinking.type=adaptive: got %q, want auto", e)
	}

	// 5. No thinking fields -> empty
	req = &shared.ClaudeRequest{Model: "claude-sonnet-4-6"}
	if e := ExtractEffortFromClaudeRequest(req); e != "" {
		t.Fatalf("no thinking: got %q, want empty", e)
	}

	// 6. output_config takes priority over thinking
	outputConfig, _ = jsonx.Marshal(shared.ClaudeOutputConfig{Effort: "low"})
	req = &shared.ClaudeRequest{
		OutputConfig: outputConfig,
		Thinking:     &shared.Thinking{Type: "enabled", BudgetTokens: intPtr(2048)},
	}
	if e := ExtractEffortFromClaudeRequest(req); e != "low" {
		t.Fatalf("priority: got %q, want low", e)
	}
}

func TestExtractEffortFromGeminiRequest(t *testing.T) {
	// 1. thinkingLevel
	tc := &shared.GeminiThinkingConfig{ThinkingLevel: "high"}
	req := &shared.GeminiChatRequest{
		GenerationConfig: shared.GeminiChatGenerationConfig{ThinkingConfig: tc},
	}
	if e := ExtractEffortFromGeminiRequest(req); e != "high" {
		t.Fatalf("thinkingLevel=high: got %q, want high", e)
	}

	// 2. IncludeThoughts=true -> auto
	tc = &shared.GeminiThinkingConfig{IncludeThoughts: true}
	req = &shared.GeminiChatRequest{
		GenerationConfig: shared.GeminiChatGenerationConfig{ThinkingConfig: tc},
	}
	if e := ExtractEffortFromGeminiRequest(req); e != "auto" {
		t.Fatalf("includeThoughts=true: got %q, want auto", e)
	}

	// 3. ThinkingBudget > 0 -> auto
	budget := 4096
	tc = &shared.GeminiThinkingConfig{ThinkingBudget: &budget}
	req = &shared.GeminiChatRequest{
		GenerationConfig: shared.GeminiChatGenerationConfig{ThinkingConfig: tc},
	}
	if e := ExtractEffortFromGeminiRequest(req); e != "auto" {
		t.Fatalf("thinkingBudget=4096: got %q, want auto", e)
	}

	// 4. No thinking config
	req = &shared.GeminiChatRequest{}
	if e := ExtractEffortFromGeminiRequest(req); e != "" {
		t.Fatalf("no thinking config: got %q, want empty", e)
	}

	// 5. ThinkingConfig with ThinkingBudget=0 (disabled) -> empty
	budget = 0
	tc = &shared.GeminiThinkingConfig{ThinkingBudget: &budget}
	req = &shared.GeminiChatRequest{
		GenerationConfig: shared.GeminiChatGenerationConfig{ThinkingConfig: tc},
	}
	if e := ExtractEffortFromGeminiRequest(req); e != "" {
		t.Fatalf("thinkingBudget=0: got %q, want empty", e)
	}
}

func intPtr(v int) *int {
	return &v
}
