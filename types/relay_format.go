package types

import "github.com/zhongruan0522/new-api/constant"

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

	RelayFormatTask    = "task"
	RelayFormatMjProxy = "mj_proxy"
)

// RelayFormatToPreferredAPIType returns the preferred API type for a given relay format.
// Returns -1 if the format has no specific API type preference (e.g. task, mj_proxy).
func RelayFormatToPreferredAPIType(format RelayFormat) int {
	switch format {
	case RelayFormatOpenAI, RelayFormatOpenAIResponses,
		RelayFormatOpenAIResponsesCompaction,
		RelayFormatOpenAIAudio, RelayFormatOpenAIImage,
		RelayFormatOpenAIRealtime, RelayFormatRerank,
		RelayFormatEmbedding:
		return constant.APITypeOpenAI
	case RelayFormatClaude:
		return constant.APITypeAnthropic
	case RelayFormatGemini:
		return constant.APITypeGemini
	default:
		return -1
	}
}
