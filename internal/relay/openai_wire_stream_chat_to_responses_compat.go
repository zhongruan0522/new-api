package relay

import relaycommon "github.com/NookMux/NookMux/internal/relay/common"

func newChatToResponsesStreamConverter(toolContext ...*relaycommon.OpenAIWireToolContext) openAIWireStreamConverter {
	return relaycommon.NewChatToResponsesStreamConverter(toolContext...)
}

func newResponsesToChatStreamConverter(includeUsage bool) openAIWireStreamConverter {
	return relaycommon.NewResponsesToChatStreamConverter(includeUsage)
}
