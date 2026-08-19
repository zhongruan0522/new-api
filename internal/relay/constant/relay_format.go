package constant

import channelconstant "github.com/NookMux/NookMux/internal/domain/channel/constant"

type RelayFormat string

const (
	RelayFormatOpenAI                    RelayFormat = "openai"
	RelayFormatClaude                                = "claude"
	RelayFormatGemini                                = "gemini"
	RelayFormatOpenAIResponses                       = "openai_responses"
	RelayFormatOpenAIResponsesCompaction             = "openai_responses_compaction"
	RelayFormatOpenAIAudio                           = "openai_audio"
	RelayFormatOpenAIImage                           = "openai_image"
	RelayFormatOpenAIRealtime                        = "openai_realtime"
	RelayFormatRerank                                = "rerank"
	RelayFormatEmbedding                             = "embedding"
)

// RelayFormatToPreferredAPIType returns the preferred API type for a given relay format.
// Returns -1 if the format has no specific API type preference.
func RelayFormatToPreferredAPIType(format RelayFormat) int {
	switch format {
	case RelayFormatOpenAI, RelayFormatOpenAIResponses,
		RelayFormatOpenAIResponsesCompaction,
		RelayFormatOpenAIAudio, RelayFormatOpenAIImage,
		RelayFormatOpenAIRealtime, RelayFormatRerank,
		RelayFormatEmbedding:
		return channelconstant.APITypeOpenAI
	case RelayFormatClaude:
		return channelconstant.APITypeAnthropic
	case RelayFormatGemini:
		return channelconstant.APITypeGemini
	default:
		return -1
	}
}
