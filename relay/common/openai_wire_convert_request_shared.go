package common

const (
	openAIResponsesInputTypeText   = "input_text"
	openAIResponsesOutputTypeText  = "output_text"
	openAIResponsesInputTypeImage  = "input_image"
	openAIResponsesInputTypeFile   = "input_file"
	openAIResponsesSummaryTextType = "summary_text"
)

const (
	openAIResponsesInputItemTypeMessage            = "message"
	openAIResponsesInputItemTypeReasoning          = "reasoning"
	openAIResponsesInputItemTypeFunctionCall       = "function_call"
	openAIResponsesInputItemTypeFunctionCallOutput = "function_call_output"
	openAIResponsesInputItemTypeCustomToolCall     = "custom_tool_call"
	openAIResponsesInputItemTypeCustomToolOutput   = "custom_tool_call_output"
	openAIResponsesInputItemTypeToolSearchCall     = "tool_search_call"
	openAIResponsesInputItemTypeToolSearchOutput   = "tool_search_output"
)

const (
	openAIResponsesToolTypeFunction   = "function"
	openAIResponsesToolTypeCustom     = "custom"
	openAIResponsesToolTypeNamespace  = "namespace"
	openAIResponsesToolTypeToolSearch = "tool_search"

	openAIResponsesToolSearchChatName = "tool_search"
	openAIResponsesCustomInputField   = "input"
)
