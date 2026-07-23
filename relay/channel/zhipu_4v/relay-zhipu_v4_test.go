package zhipu_4v

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/dto"
)

func TestRequestOpenAI2Zhipu_DeveloperRoleNormalizedToSystem(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "glm-4v-plus",
		Messages: []dto.Message{
			{Role: "developer", Content: "system instructions"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "tool", Content: "result", ToolCallId: "call_1"},
		},
	}

	got := requestOpenAI2Zhipu(request)

	roles := make([]string, 0, len(got.Messages))
	for _, m := range got.Messages {
		roles = append(roles, m.Role)
	}
	want := []string{"system", "user", "assistant", "tool"}
	for i, r := range roles {
		if r != want[i] {
			t.Fatalf("messages[%d].role = %q, want %q (full roles: %v)", i, r, want[i], roles)
		}
	}
	if got.Messages[0].StringContent() != "system instructions" {
		t.Fatalf("system content = %q, want %q", got.Messages[0].StringContent(), "system instructions")
	}
}

func TestRequestOpenAI2Zhipu_DeveloperRoleNormalizedCaseInsensitive(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "glm-4v-plus",
		Messages: []dto.Message{
			{Role: "Developer", Content: "x"},
			{Role: "DEVELOPER", Content: "y"},
		},
	}

	got := requestOpenAI2Zhipu(request)
	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1 (merged)", len(got.Messages))
	}
	if got.Messages[0].Role != "system" {
		t.Fatalf("role = %q, want %q", got.Messages[0].Role, "system")
	}
	if !strings.Contains(got.Messages[0].StringContent(), "x") || !strings.Contains(got.Messages[0].StringContent(), "y") {
		t.Fatalf("merged content = %q, want both x and y", got.Messages[0].StringContent())
	}
}

// 连续的 system/developer 消息必须合并为一条，否则智谱会覆盖丢失内容。
func TestRequestOpenAI2Zhipu_MergesConsecutiveSystemMessages(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "glm-4v-plus",
		Messages: []dto.Message{
			{Role: "developer", Content: "instructions from Responses conversion"},
			{Role: "system", Content: "extra system rules"},
			{Role: "user", Content: "hi"},
		},
	}

	got := requestOpenAI2Zhipu(request)

	if len(got.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2 (two system merged into one): %#v", len(got.Messages), got.Messages)
	}
	if got.Messages[0].Role != "system" {
		t.Fatalf("messages[0].role = %q, want system", got.Messages[0].Role)
	}
	merged := got.Messages[0].StringContent()
	if !strings.Contains(merged, "instructions from Responses conversion") {
		t.Fatalf("merged system content lost instructions: %q", merged)
	}
	if !strings.Contains(merged, "extra system rules") {
		t.Fatalf("merged system content lost extra rules: %q", merged)
	}
	if got.Messages[1].Role != "user" {
		t.Fatalf("messages[1].role = %q, want user", got.Messages[1].Role)
	}
}

func TestRequestOpenAI2Zhipu_NonConsecutiveSystemMessagesKeptSeparate(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "glm-4v-plus",
		Messages: []dto.Message{
			{Role: "system", Content: "first"},
			{Role: "user", Content: "hi"},
			{Role: "system", Content: "second"},
		},
	}

	got := requestOpenAI2Zhipu(request)
	if len(got.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(got.Messages))
	}
}

func TestRequestOpenAI2Zhipu_SkipsEmptySystemMessages(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "glm-4v-plus",
		Messages: []dto.Message{
			{Role: "developer", Content: "   "},
			{Role: "user", Content: "hi"},
		},
	}

	got := requestOpenAI2Zhipu(request)
	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1 (empty system skipped): %#v", len(got.Messages), got.Messages)
	}
	if got.Messages[0].Role != "user" {
		t.Fatalf("messages[0].role = %q, want user", got.Messages[0].Role)
	}
}

func TestRequestOpenAI2Zhipu_PreservesValidRoles(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "glm-4v-plus",
		Messages: []dto.Message{
			{Role: "system", Content: "s"},
			{Role: "user", Content: "u"},
		},
	}

	got := requestOpenAI2Zhipu(request)
	if len(got.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(got.Messages))
	}
	if got.Messages[0].Role != "system" || got.Messages[1].Role != "user" {
		t.Fatalf("roles = %v %v, want system user", got.Messages[0].Role, got.Messages[1].Role)
	}
}
