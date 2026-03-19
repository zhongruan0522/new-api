package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
)

func TestApplyResponsesSystemPromptAddsInstructionsWhenEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeySystemPromptOverride, false)

	request := &dto.OpenAIResponsesRequest{}
	err := applyResponsesSystemPrompt(ctx, request, dto.ChannelSettings{SystemPrompt: "sys"})
	if err != nil {
		t.Fatalf("applyResponsesSystemPrompt returned error: %v", err)
	}

	var instructions string
	if err := common.Unmarshal(request.Instructions, &instructions); err != nil {
		t.Fatalf("failed to unmarshal instructions: %v", err)
	}
	if instructions != "sys" {
		t.Fatalf("instructions = %q, want %q", instructions, "sys")
	}
	if got := common.GetContextKeyBool(ctx, constant.ContextKeySystemPromptOverride); got {
		t.Fatalf("system prompt override flag = %t, want false", got)
	}
}

func TestApplyResponsesSystemPromptKeepsExistingInstructionsWithoutOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeySystemPromptOverride, false)

	raw, _ := common.Marshal("existing")
	request := &dto.OpenAIResponsesRequest{Instructions: raw}
	err := applyResponsesSystemPrompt(ctx, request, dto.ChannelSettings{SystemPrompt: "sys"})
	if err != nil {
		t.Fatalf("applyResponsesSystemPrompt returned error: %v", err)
	}

	var instructions string
	if err := common.Unmarshal(request.Instructions, &instructions); err != nil {
		t.Fatalf("failed to unmarshal instructions: %v", err)
	}
	if instructions != "existing" {
		t.Fatalf("instructions = %q, want %q", instructions, "existing")
	}
}

func TestApplyResponsesSystemPromptPrependsWhenOverrideEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeySystemPromptOverride, false)

	raw, _ := common.Marshal("existing")
	request := &dto.OpenAIResponsesRequest{Instructions: raw}
	err := applyResponsesSystemPrompt(ctx, request, dto.ChannelSettings{
		SystemPrompt:         "sys",
		SystemPromptOverride: true,
	})
	if err != nil {
		t.Fatalf("applyResponsesSystemPrompt returned error: %v", err)
	}

	var instructions string
	if err := common.Unmarshal(request.Instructions, &instructions); err != nil {
		t.Fatalf("failed to unmarshal instructions: %v", err)
	}
	if instructions != "sys\nexisting" {
		t.Fatalf("instructions = %q, want %q", instructions, "sys\nexisting")
	}
	if got := common.GetContextKeyBool(ctx, constant.ContextKeySystemPromptOverride); !got {
		t.Fatalf("system prompt override flag = %t, want true", got)
	}
}
