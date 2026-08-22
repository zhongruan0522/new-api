package common

import (
	"encoding/json"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

func TestEnsureReasoningEffort(t *testing.T) {
	// 1. info 已有值时保持不变
	info := &RelayInfo{ReasoningEffort: "high"}
	req := &shared.GeneralOpenAIRequest{ReasoningEffort: "low"}
	EnsureReasoningEffort(info, req)
	if info.ReasoningEffort != "high" {
		t.Fatalf("existing effort overwritten: got %q, want high", info.ReasoningEffort)
	}

	// 2. OpenAI 请求兜底解析
	info = &RelayInfo{}
	req = &shared.GeneralOpenAIRequest{ReasoningEffort: "medium"}
	EnsureReasoningEffort(info, req)
	if info.ReasoningEffort != "medium" {
		t.Fatalf("OpenAI fallback: got %q, want medium", info.ReasoningEffort)
	}

	// 3. OpenAI 启用思考但无强度 -> auto
	info = &RelayInfo{}
	enableThinkingJSON, _ := json.Marshal(true)
	req = &shared.GeneralOpenAIRequest{EnableThinking: enableThinkingJSON}
	EnsureReasoningEffort(info, req)
	if info.ReasoningEffort != "auto" {
		t.Fatalf("OpenAI auto: got %q, want auto", info.ReasoningEffort)
	}

	// 4. Claude 请求兜底解析
	info = &RelayInfo{}
	outputConfig, _ := jsonx.Marshal(shared.ClaudeOutputConfig{Effort: "low"})
	claudeReq := &shared.ClaudeRequest{OutputConfig: outputConfig}
	EnsureReasoningEffort(info, claudeReq)
	if info.ReasoningEffort != "low" {
		t.Fatalf("Claude fallback: got %q, want low", info.ReasoningEffort)
	}

	// 5. Claude thinking.type=adaptive -> auto
	info = &RelayInfo{}
	claudeReq = &shared.ClaudeRequest{Thinking: &shared.Thinking{Type: "adaptive"}}
	EnsureReasoningEffort(info, claudeReq)
	if info.ReasoningEffort != "auto" {
		t.Fatalf("Claude adaptive: got %q, want auto", info.ReasoningEffort)
	}

	// 6. Gemini 请求兜底解析
	info = &RelayInfo{}
	geminiReq := &shared.GeminiChatRequest{
		GenerationConfig: shared.GeminiChatGenerationConfig{
			ThinkingConfig: &shared.GeminiThinkingConfig{ThinkingLevel: "high"},
		},
	}
	EnsureReasoningEffort(info, geminiReq)
	if info.ReasoningEffort != "high" {
		t.Fatalf("Gemini fallback: got %q, want high", info.ReasoningEffort)
	}

	// 7. Gemini includeThoughts -> auto
	info = &RelayInfo{}
	geminiReq = &shared.GeminiChatRequest{
		GenerationConfig: shared.GeminiChatGenerationConfig{
			ThinkingConfig: &shared.GeminiThinkingConfig{IncludeThoughts: true},
		},
	}
	EnsureReasoningEffort(info, geminiReq)
	if info.ReasoningEffort != "auto" {
		t.Fatalf("Gemini auto: got %q, want auto", info.ReasoningEffort)
	}

	// 8. 无 reasoning 字段时不回填
	info = &RelayInfo{}
	req = &shared.GeneralOpenAIRequest{Model: "gpt-4o"}
	EnsureReasoningEffort(info, req)
	if info.ReasoningEffort != "" {
		t.Fatalf("no reasoning: got %q, want empty", info.ReasoningEffort)
	}

	// 9. nil info / nil request 不 panic
	EnsureReasoningEffort(nil, req)
	EnsureReasoningEffort(info, nil)
}

func TestEnsureReasoningEffortOpenAIResponses(t *testing.T) {
	// Responses API 请求兜底解析
	info := &RelayInfo{}
	req := &shared.OpenAIResponsesRequest{
		Reasoning: &shared.Reasoning{Effort: "high"},
	}
	EnsureReasoningEffort(info, req)
	if info.ReasoningEffort != "high" {
		t.Fatalf("Responses fallback: got %q, want high", info.ReasoningEffort)
	}
}
