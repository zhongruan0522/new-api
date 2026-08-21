package billing

import (
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/httpapi"
	tokenizer "github.com/NookMux/NookMux/internal/infra/tokenizer"
	"github.com/gin-gonic/gin"
)

//func GetPromptTokens(textRequest shared.GeneralOpenAIRequest, relayMode int) (int, error) {
//	switch relayMode {
//	case shared.RelayModeChatCompletions:
//		return CountTokenMessages(textRequest.Messages, textRequest.Model)
//	case shared.RelayModeCompletions:
//		return CountTokenInput(textRequest.Prompt, textRequest.Model), nil
//	case shared.RelayModeModerations:
//		return CountTokenInput(textRequest.Input, textRequest.Model), nil
//	}
//	return 0, errors.New("unknown relay mode")
//}

func ResponseText2Usage(c *gin.Context, responseText string, modeName string, promptTokens int) *shared.Usage {
	httpapi.SetContextKey(c, common.ContextKeyLocalCountTokens, true)
	usage := &shared.Usage{}
	usage.PromptTokens = promptTokens
	usage.CompletionTokens = tokenizer.EstimateTokenByModel(modeName, responseText)
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage
}

func ValidUsage(usage *shared.Usage) bool {
	return usage != nil && (usage.PromptTokens != 0 || usage.CompletionTokens != 0)
}
