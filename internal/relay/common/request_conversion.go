package common

import (
	"github.com/NookMux/NookMux/internal/domain/shared"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
)

func GuessRelayFormatFromRequest(req any) (relayconstant.RelayFormat, bool) {
	switch req.(type) {
	case *shared.GeneralOpenAIRequest, shared.GeneralOpenAIRequest:
		return relayconstant.RelayFormatOpenAI, true
	case *shared.OpenAIResponsesRequest, shared.OpenAIResponsesRequest:
		return relayconstant.RelayFormatOpenAIResponses, true
	case *shared.ClaudeRequest, shared.ClaudeRequest:
		return relayconstant.RelayFormatClaude, true
	case *shared.GeminiChatRequest, shared.GeminiChatRequest:
		return relayconstant.RelayFormatGemini, true
	case *shared.EmbeddingRequest, shared.EmbeddingRequest:
		return relayconstant.RelayFormatEmbedding, true
	case *shared.RerankRequest, shared.RerankRequest:
		return relayconstant.RelayFormatRerank, true
	case *shared.ImageRequest, shared.ImageRequest:
		return relayconstant.RelayFormatOpenAIImage, true
	case *shared.AudioRequest, shared.AudioRequest:
		return relayconstant.RelayFormatOpenAIAudio, true
	default:
		return "", false
	}
}

func AppendRequestConversionFromRequest(info *RelayInfo, req any) {
	if info == nil {
		return
	}
	format, ok := GuessRelayFormatFromRequest(req)
	if !ok {
		return
	}
	info.AppendRequestConversion(format)
}
