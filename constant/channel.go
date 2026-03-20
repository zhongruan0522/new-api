package constant

const (
	ChannelTypeUnknown        = 0
	ChannelTypeOpenAI         = 1
	ChannelTypeMidjourney     = 2
	ChannelTypeAzure          = 3
	ChannelTypeOllama         = 4
	ChannelTypeMidjourneyPlus = 5
	ChannelTypeCustom         = 8
	ChannelTypePaLM           = 11
	ChannelTypeAnthropic      = 14
	ChannelTypeAli            = 17
	ChannelTypeXunfei         = 18
	ChannelTypeOpenRouter     = 20
	ChannelTypeTencent        = 23
	ChannelTypeGemini         = 24
	ChannelTypeMoonshot       = 25
	ChannelTypeZhipu_v4       = 26
	ChannelTypeLingYiWanWu    = 31
	ChannelTypeAws            = 33
	ChannelTypeCohere         = 34
	ChannelTypeMiniMax        = 35
	ChannelTypeSunoAPI        = 36
	ChannelTypeDify           = 37
	ChannelTypeJina           = 38
	ChannelCloudflare         = 39
	ChannelTypeSiliconFlow    = 40
	ChannelTypeVertexAi       = 41
	ChannelTypeMistral        = 42
	ChannelTypeDeepSeek       = 43
	ChannelTypeVolcEngine     = 45
	ChannelTypeXai            = 48
	ChannelTypeDummy          = 49 // this one is only for count, do not add any channel after this
)

var ChannelBaseURLs = []string{
	"",                                    // 0  Unknown
	"https://api.openai.com",              // 1  OpenAI
	"https://oa.api2d.net",                // 2  Midjourney
	"",                                    // 3  Azure
	"http://localhost:11434",              // 4  Ollama
	"https://api.openai-sb.com",           // 5  MidjourneyPlus
	"",                                    // 6  (removed)
	"",                                    // 7  (removed)
	"",                                    // 8  Custom
	"",                                    // 9  (removed)
	"",                                    // 10 (removed)
	"",                                    // 11 PaLM
	"",                                    // 12 (removed)
	"",                                    // 13 (removed)
	"https://api.anthropic.com",           // 14 Anthropic
	"",                                    // 15 (removed)
	"",                                    // 16 (removed)
	"https://dashscope.aliyuncs.com",      // 17 Ali
	"",                                    // 18 Xunfei
	"",                                    // 19 (removed)
	"https://openrouter.ai/api",           // 20 OpenRouter
	"",                                    // 21 (removed)
	"",                                    // 22 (removed)
	"https://hunyuan.tencentcloudapi.com", // 23 Tencent
	"https://generativelanguage.googleapis.com", // 24 Gemini
	"https://api.moonshot.cn",                   // 25 Moonshot
	"https://open.bigmodel.cn",                  // 26 ZhipuV4
	"",                                          // 27 (removed)
	"",                                          // 28
	"",                                          // 29
	"",                                          // 30
	"https://api.lingyiwanwu.com",               // 31 LingYiWanWu
	"",                                          // 32
	"",                                          // 33 AWS
	"https://api.cohere.ai",                     // 34 Cohere
	"https://api.minimax.chat",                  // 35 MiniMax
	"",                                          // 36 SunoAPI
	"https://api.dify.ai",                       // 37 Dify
	"https://api.jina.ai",                       // 38 Jina
	"https://api.cloudflare.com",                // 39 Cloudflare
	"https://api.siliconflow.cn",                // 40 SiliconFlow
	"",                                          // 41 VertexAi
	"https://api.mistral.ai",                    // 42 Mistral
	"https://api.deepseek.com",                  // 43 DeepSeek
	"",                                          // 44 (removed)
	"https://ark.cn-beijing.volces.com",         // 45 VolcEngine
	"",                                          // 46 (removed)
	"",                                          // 47 (removed)
	"https://api.x.ai",                          // 48 xAI
}

var ChannelTypeNames = map[int]string{
	ChannelTypeUnknown:     "Unknown",
	ChannelTypeOpenAI:      "OpenAI",
	ChannelTypeAzure:       "Azure",
	ChannelTypeOllama:      "Ollama",
	ChannelTypeCustom:      "Custom",
	ChannelTypePaLM:        "PaLM",
	ChannelTypeAnthropic:   "Anthropic",
	ChannelTypeAli:         "Ali",
	ChannelTypeXunfei:      "Xunfei",
	ChannelTypeOpenRouter:  "OpenRouter",
	ChannelTypeTencent:     "Tencent",
	ChannelTypeGemini:      "Gemini",
	ChannelTypeMoonshot:    "Moonshot",
	ChannelTypeZhipu_v4:    "ZhipuV4",
	ChannelTypeLingYiWanWu: "LingYiWanWu",
	ChannelTypeAws:         "AWS",
	ChannelTypeCohere:      "Cohere",
	ChannelTypeMiniMax:     "MiniMax",
	ChannelTypeDify:        "Dify",
	ChannelTypeJina:        "Jina",
	ChannelCloudflare:      "Cloudflare",
	ChannelTypeSiliconFlow: "SiliconFlow",
	ChannelTypeVertexAi:    "VertexAI",
	ChannelTypeMistral:     "Mistral",
	ChannelTypeDeepSeek:    "DeepSeek",
	ChannelTypeVolcEngine:  "VolcEngine",
	ChannelTypeXai:         "xAI",
}

func GetChannelTypeName(channelType int) string {
	if name, ok := ChannelTypeNames[channelType]; ok {
		return name
	}
	return "Unknown"
}

type ChannelSpecialBase struct {
	ClaudeBaseURL string
	OpenAIBaseURL string
}

var ChannelSpecialBases = map[string]ChannelSpecialBase{
	"glm-coding-plan": {
		ClaudeBaseURL: "https://open.bigmodel.cn/api/anthropic",
		OpenAIBaseURL: "https://open.bigmodel.cn/api/coding/paas/v4",
	},
	"glm-coding-plan-international": {
		ClaudeBaseURL: "https://api.z.ai/api/anthropic",
		OpenAIBaseURL: "https://api.z.ai/api/coding/paas/v4",
	},
	"kimi-coding-plan": {
		ClaudeBaseURL: "https://api.kimi.com/coding",
		OpenAIBaseURL: "https://api.kimi.com/coding/v1",
	},
	"doubao-coding-plan": {
		ClaudeBaseURL: "https://ark.cn-beijing.volces.com/api/coding",
		OpenAIBaseURL: "https://ark.cn-beijing.volces.com/api/coding/v3",
	},
}
