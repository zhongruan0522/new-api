package model_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

type GlobalSettings struct {
	PassThroughRequestEnabled bool     `json:"pass_through_request_enabled"`
	ThinkingModelBlacklist    []string `json:"thinking_model_blacklist"`

	// ThirdPartyMultimodal* settings are used by the channel-level
	// "image_auto_convert_to_url_mode=third_party_model" feature:
	//   - Extract images/videos from user messages
	//   - Resolve them into URLs (may store base64 into /mcp/* assets)
	//   - Call a configured multimodal model to convert media into text
	//   - Append the text back to the original user message as:
	//       图片N：xxxxx / 视频N：xxxxx
	ThirdPartyMultimodalModelID         string `json:"third_party_multimodal_model_id"`
	ThirdPartyMultimodalCallAPIType     int    `json:"third_party_multimodal_call_api_type"`
	ThirdPartyMultimodalSystemPrompt    string `json:"third_party_multimodal_system_prompt"`
	ThirdPartyMultimodalFirstUserPrompt string `json:"third_party_multimodal_first_user_prompt"`

	// ThirdPartyMultimodal* identity headers are ONLY used by the internal
	// third-party multimodal request (media-to-text).
	// They will NOT be injected into normal upstream relay requests.
	ThirdPartyMultimodalUserAgent   string `json:"third_party_multimodal_user_agent"`
	ThirdPartyMultimodalXTitle      string `json:"third_party_multimodal_x_title"`
	ThirdPartyMultimodalHTTPReferer string `json:"third_party_multimodal_http_referer"`
}

// 默认配置
var defaultOpenaiSettings = GlobalSettings{
	PassThroughRequestEnabled: false,
	ThinkingModelBlacklist: []string{
		"moonshotai/kimi-k2-thinking",
		"kimi-k2-thinking",
	},
	ThirdPartyMultimodalModelID:         "",
	ThirdPartyMultimodalCallAPIType:     0,
	ThirdPartyMultimodalSystemPrompt:    "",
	ThirdPartyMultimodalFirstUserPrompt: "",
	ThirdPartyMultimodalUserAgent:       "",
	ThirdPartyMultimodalXTitle:          "",
	ThirdPartyMultimodalHTTPReferer:     "",
}

// 全局实例
var globalSettings = defaultOpenaiSettings

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("global", &globalSettings)
}

func GetGlobalSettings() *GlobalSettings {
	return &globalSettings
}

// ShouldPreserveThinkingSuffix 判断模型是否配置为保留 thinking/-nothinking/-low/-high/-medium 后缀
func ShouldPreserveThinkingSuffix(modelName string) bool {
	target := strings.TrimSpace(modelName)
	if target == "" {
		return false
	}

	for _, entry := range globalSettings.ThinkingModelBlacklist {
		if strings.TrimSpace(entry) == target {
			return true
		}
	}
	return false
}
