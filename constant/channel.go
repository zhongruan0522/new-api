package constant

const (
	ChannelTypeUnknown        = 0
	ChannelTypeOpenAI         = 1
	ChannelTypeMidjourney     = 2
	ChannelTypeAzure          = 3
	ChannelTypeOllama         = 4
	ChannelTypeMidjourneyPlus = 5
	ChannelTypeCustom         = 8
	ChannelTypeAnthropic      = 14
	ChannelTypeOpenRouter     = 20
	ChannelTypeGemini         = 24
	ChannelTypeMoonshot       = 25
	ChannelTypeZhipu_v4       = 26
	ChannelTypeAws            = 33
	ChannelTypeMiniMax        = 35
	ChannelTypeSunoAPI        = 36
	ChannelTypeSiliconFlow    = 40
	ChannelTypeVertexAi       = 41
	ChannelTypeDeepSeek       = 43
	ChannelTypeXiaomi         = 6
	ChannelTypeDummy          = 7 // this one is only for count, do not add any channel after this
)

var ChannelBaseURLs = []string{
	"",                          // 0  Unknown
	"https://api.openai.com",    // 1  OpenAI
	"https://oa.api2d.net",      // 2  Midjourney
	"",                          // 3  Azure
	"http://localhost:11434",    // 4  Ollama
	"https://api.openai-sb.com", // 5  MidjourneyPlus
	"https://api.xiaomimimo.com", // 6  Xiaomi
	"",                          // 7  Dummy
	"",                          // 8  Custom
	"",                          // 9  (removed)
	"",                          // 10 (removed)
	"",                          // 11 (removed)
	"",                          // 12 (removed)
	"",                          // 13 (removed)
	"https://api.anthropic.com", // 14 Anthropic
	"",                          // 15 (removed)
	"",                          // 16 (removed)
	"",                          // 17 (removed)
	"",                          // 18 (removed)
	"",                          // 19 (removed)
	"https://openrouter.ai/api", // 20 OpenRouter
	"",                          // 21 (removed)
	"",                          // 22 (removed)
	"",                          // 23 (removed)
	"https://generativelanguage.googleapis.com", // 24 Gemini
	"https://api.moonshot.cn",                   // 25 Moonshot
	"https://open.bigmodel.cn",                  // 26 ZhipuV4
	"",                                          // 27 (removed)
	"",                                          // 28
	"",                                          // 29
	"",                                          // 30
	"",                                          // 31 (removed)
	"",                                          // 32
	"",                                          // 33 AWS
	"",                                          // 34 (removed)
	"https://api.minimaxi.com/v1",               // 35 MiniMax
	"",                                          // 36 SunoAPI
	"",                                          // 37 (removed)
	"",                                          // 38 (removed)
	"",                                          // 39 (removed)
	"https://api.siliconflow.cn",                // 40 SiliconFlow
	"",                                          // 41 VertexAi
	"",                                          // 42 (removed)
	"https://api.deepseek.com",                  // 43 DeepSeek
	"",                                          // 44 (removed)
	"",                                          // 45 (removed)
	"",                                          // 46 (removed)
	"",                                          // 47 (removed)
	"",                                          // 48 (removed)
}

var ChannelTypeNames = map[int]string{
	ChannelTypeUnknown:     "Unknown",
	ChannelTypeOpenAI:      "OpenAI",
	ChannelTypeAzure:       "Azure",
	ChannelTypeOllama:      "Ollama",
	ChannelTypeCustom:      "Custom",
	ChannelTypeAnthropic:   "Anthropic",
	ChannelTypeOpenRouter:  "OpenRouter",
	ChannelTypeGemini:      "Gemini",
	ChannelTypeMoonshot:    "Moonshot",
	ChannelTypeZhipu_v4:    "ZhipuV4",
	ChannelTypeAws:         "AWS",
	ChannelTypeMiniMax:     "MiniMax",
	ChannelTypeSiliconFlow: "SiliconFlow",
	ChannelTypeVertexAi:    "VertexAI",
	ChannelTypeDeepSeek:    "DeepSeek",
	ChannelTypeXiaomi:      "Xiaomi",
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

// IsChannelPlan 检测给定的 BaseURL 是否为内置套餐地址
func IsChannelPlan(baseURL string) (string, bool) {
	if baseURL == "" {
		return "", false
	}
	_, ok := ChannelSpecialBases[baseURL]
	return baseURL, ok
}

// GetSupportedPlanQuotaProviders 返回支持额度查询的套餐 key 列表
var SupportedPlanQuotaProviders = map[string]bool{
	"glm-coding-plan":                   true,
	"glm-coding-plan-international":     true,
	"kimi-coding-plan":                  true,
	"minimax-coding-plan":               true,
	"minimax-coding-plan-international": true,
	"ollama-coding-plan":               true,
	"xiaomi-coding-plan":               true,
	"xiaomi-coding-plan-sgp":           true,
	"xiaomi-coding-plan-ams":           true,
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
	"minimax-coding-plan": {
		ClaudeBaseURL: "https://api.minimaxi.com/anthropic",
	},
	"minimax-coding-plan-international": {
		ClaudeBaseURL: "https://api.minimaxi.io/anthropic",
	},
	"ollama-coding-plan": {
		ClaudeBaseURL: "https://ollama.com",
		OpenAIBaseURL: "https://ollama.com",
	},
	"xiaomi-coding-plan": {
		ClaudeBaseURL: "https://token-plan-cn.xiaomimimo.com/anthropic",
		OpenAIBaseURL: "https://token-plan-cn.xiaomimimo.com/v1",
	},
	"xiaomi-coding-plan-sgp": {
		ClaudeBaseURL: "https://token-plan-sgp.xiaomimimo.com/anthropic",
		OpenAIBaseURL: "https://token-plan-sgp.xiaomimimo.com/v1",
	},
	"xiaomi-coding-plan-ams": {
		ClaudeBaseURL: "https://token-plan-ams.xiaomimimo.com/anthropic",
		OpenAIBaseURL: "https://token-plan-ams.xiaomimimo.com/v1",
	},
}
