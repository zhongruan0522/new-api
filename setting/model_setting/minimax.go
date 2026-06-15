package model_setting

import (
	"fmt"
	"regexp"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/config"
)

// MiniMaxSettings 定义 MiniMax TTS 增强配置。
// 仅对 MiniMax 渠道的 /v1/audio/speech 生效。
type MiniMaxSettings struct {
	// Enabled 是 TTS 增强总开关，默认关闭。
	Enabled bool `json:"enabled"`
	// ModelRedirect 是 TTS 模型重定向表，键是客户端请求的模型名，值是发给 MiniMax 的真实模型名。
	// 例如 {"tts-1-hd": "speech-02-hd"}
	ModelRedirect map[string]string `json:"model_redirect"`
	// EmotionPattern 识别文本中情绪标签的正则表达式，如 `\((happy|sad|高兴)\)`。
	EmotionPattern string `json:"emotion_pattern"`
	// EmotionRedirect 是情绪标签值（括号内文本）到 MiniMax voice_setting.emotion 的映射表。
	// 例如 {"happy": "happy", "高兴": "happy"}
	EmotionRedirect map[string]string `json:"emotion_redirect"`
	// ToneWordPattern 识别文本中语气词标签的正则表达式，如 `\((laughs|crying)\)`。
	ToneWordPattern string `json:"tone_word_pattern"`
	// ToneWordRedirect 是语气词标签值（括号内文本）到替换值的映射表。
	// 标签**原地替换**：括号位置不变，只替换内容。例如 {"laughs": "aaa", "xxx": "bbb"}
	ToneWordRedirect map[string]string `json:"tone_word_redirect"`
	// VoiceRedirect 是 OpenAI voice 名到 MiniMax voice_id 的映射表。
	// 例如 {"narrator": "male-qn-qingse", "alloy": "female-shaonv"}
	VoiceRedirect map[string]string `json:"voice_redirect"`
}

var defaultMiniMaxSettings = MiniMaxSettings{
	Enabled:          false,
	ModelRedirect:    map[string]string{},
	EmotionRedirect:  map[string]string{},
	ToneWordRedirect: map[string]string{},
	VoiceRedirect:    map[string]string{},
}

var minimaxSettings = defaultMiniMaxSettings

func init() {
	config.GlobalConfig.Register("minimax", &minimaxSettings)
}

func GetMiniMaxSettings() *MiniMaxSettings {
	return &minimaxSettings
}

// ValidateMiniMaxOptionValue 校验 minimax.* 系统设置项的值。
// 在 controller.UpdateOption 中保存前调用，校验失败返回错误。
// 不属于 minimax.* 的 key 直接返回 nil。
func ValidateMiniMaxOptionValue(key, value string) error {
	switch key {
	case "minimax.model_redirect", "minimax.emotion_redirect",
		"minimax.tone_word_redirect", "minimax.voice_redirect":
		return validateStringMap(value)
	case "minimax.emotion_pattern", "minimax.tone_word_pattern":
		if value == "" {
			return nil
		}
		if _, err := regexp.Compile(value); err != nil {
			return fmt.Errorf("invalid regex %s: %w", key, err)
		}
		return nil
	default:
		return nil
	}
}

// validateStringMap 校验 value 是合法 JSON 且类型为 map[string]string。
// 拒绝空串：空串在 config.UpdateConfigFromMap 中 unmarshal 失败会被静默跳过，
// 导致配置显示已清空但运行时仍使用旧值。清空请传 "{}"。
func validateStringMap(value string) error {
	var m map[string]string
	if err := common.UnmarshalJsonStr(value, &m); err != nil {
		return fmt.Errorf("invalid JSON for string map: %w", err)
	}
	return nil
}
