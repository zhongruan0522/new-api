package constant

type EndpointType string

const (
	EndpointTypeOpenAI         EndpointType = "openai"
	EndpointTypeOpenAIResponse EndpointType = "openai-response"
	EndpointTypeAnthropic      EndpointType = "anthropic"
	EndpointTypeGemini         EndpointType = "gemini"
	EndpointTypeJinaRerank     EndpointType = "jina-rerank"
)
