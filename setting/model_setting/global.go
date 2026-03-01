package model_setting

import (
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

type ChatCompletionsToResponsesPolicy struct {
	Enabled       bool     `json:"enabled"`
	AllChannels   bool     `json:"all_channels"`
	ChannelIDs    []int    `json:"channel_ids,omitempty"`
	ChannelTypes  []int    `json:"channel_types,omitempty"`
	ModelPatterns []string `json:"model_patterns,omitempty"`
}

func (p ChatCompletionsToResponsesPolicy) IsChannelEnabled(channelID int, channelType int) bool {
	if !p.Enabled {
		return false
	}
	if p.AllChannels {
		return true
	}

	if channelID > 0 && len(p.ChannelIDs) > 0 && slices.Contains(p.ChannelIDs, channelID) {
		return true
	}
	if channelType > 0 && len(p.ChannelTypes) > 0 && slices.Contains(p.ChannelTypes, channelType) {
		return true
	}
	return false
}

type GlobalSettings struct {
	PassThroughRequestEnabled        bool                             `json:"pass_through_request_enabled"`
	ThinkingModelBlacklist           []string                         `json:"thinking_model_blacklist"`
	ChatCompletionsToResponsesPolicy ChatCompletionsToResponsesPolicy `json:"chat_completions_to_responses_policy"`

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
}

// 默认配置
var defaultOpenaiSettings = GlobalSettings{
	PassThroughRequestEnabled: false,
	ThinkingModelBlacklist: []string{
		"moonshotai/kimi-k2-thinking",
		"kimi-k2-thinking",
	},
	ChatCompletionsToResponsesPolicy: ChatCompletionsToResponsesPolicy{
		Enabled:     false,
		AllChannels: true,
	},
	ThirdPartyMultimodalModelID:         "",
	ThirdPartyMultimodalCallAPIType:     0,
	ThirdPartyMultimodalSystemPrompt:    "",
	ThirdPartyMultimodalFirstUserPrompt: "",
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
