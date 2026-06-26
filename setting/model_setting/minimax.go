package model_setting

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/config"
)

const (
	maxMiniMaxVoiceWhitelistEntries = 100000
	maxMiniMaxVoiceIDLength         = 512
	maxMiniMaxVoiceWhitelistBytes   = 4 * 1024 * 1024
)

type MiniMaxVoiceWhitelist []string

func (w MiniMaxVoiceWhitelist) MarshalJSON() ([]byte, error) {
	if len(w) == 0 {
		return []byte("{}"), nil
	}
	return common.Marshal([]string(w))
}

func (w *MiniMaxVoiceWhitelist) UnmarshalJSON(data []byte) error {
	voices, _, err := parseMiniMaxVoiceWhitelistJSON(string(data))
	if err != nil {
		return err
	}
	*w = MiniMaxVoiceWhitelist(voices)
	return nil
}

// MiniMaxSettings 定义 MiniMax TTS 增强配置。
// 仅对 MiniMax 渠道的 /v1/audio/speech 生效。
type MiniMaxSettings struct {
	// Enabled 是 TTS 增强总开关，默认关闭。
	Enabled bool `json:"enabled"`
	// ModelRedirect 是 TTS 模型重定向表，键是客户端请求的模型名，值是发给 MiniMax 的真实模型名。
	// 例如 {"tts-1-hd": "speech-02-hd"}
	ModelRedirect map[string]string `json:"model_redirect"`
	// EmotionPattern 识别文本中情绪标签的正则表达式。
	// 默认匹配英文标点包裹的 <tts emotion="...">text</tts>，第一个 emotion 值落地到
	// MiniMax voice_setting.emotion，标签包裹的正文会保留。也兼容旧的括号形式 \((happy|sad)\)。
	EmotionPattern string `json:"emotion_pattern"`
	// EmotionRedirect 是标签值到 MiniMax voice_setting.emotion 的映射表，例如 {"happy": "happy"}。
	// 为空时直接使用正则捕获到的标签值；非空时优先用映射值，未命中则不设置 emotion。
	EmotionRedirect map[string]string `json:"emotion_redirect"`
	// ToneWordPattern 识别文本中语气词标签的正则表达式。
	// 默认匹配英文半角括号包裹的 (laugh)，原地替换括号内文本。
	ToneWordPattern string `json:"tone_word_pattern"`
	// ToneWordRedirect 是语气词标签值到替换值的映射表。
	// 标签**原地替换**：括号位置不变，只替换内容，例如 {"laughs": "笑"}。
	// 若用户直接传入任一映射目标值（替换值），则整个 (替换值) 标签会被删除，不发给上游。
	ToneWordRedirect map[string]string `json:"tone_word_redirect"`
	// VoiceRedirect 是 OpenAI voice 名到 MiniMax voice_id 的映射表。
	// 例如 {"narrator": "male-qn-qingse", "alloy": "female-shaonv"}
	VoiceRedirect map[string]string `json:"voice_redirect"`
	// VoiceWhitelist 是允许客户端请求的音色 ID 列表，空对象/空数组表示不限制。
	VoiceWhitelist MiniMaxVoiceWhitelist `json:"voice_whitelist"`
}

type MiniMaxTTSPolicyResult struct {
	Enabled      bool
	Model        string
	Voice        string
	Text         string
	Emotion      string
	OutputFormat string
}

var defaultMiniMaxSettings = MiniMaxSettings{
	Enabled:          false,
	ModelRedirect:    map[string]string{},
	EmotionPattern:   `<tts\s+emotion="([^"]+)">([\s\S]*?)</tts>`,
	EmotionRedirect:  map[string]string{},
	ToneWordPattern:  `\(([^()]+)\)`,
	ToneWordRedirect: map[string]string{},
	VoiceRedirect:    map[string]string{},
	VoiceWhitelist:   MiniMaxVoiceWhitelist{},
}

var minimaxSettings = defaultMiniMaxSettings
var minimaxSettingsMu sync.RWMutex

func init() {
	config.GlobalConfig.Register("minimax", &minimaxSettings)
}

func GetMiniMaxSettings() *MiniMaxSettings {
	return &minimaxSettings
}

func WithMiniMaxSettingsReadLock(fn func() error) error {
	minimaxSettingsMu.RLock()
	defer minimaxSettingsMu.RUnlock()
	return fn()
}

func WithMiniMaxSettingsWriteLock(fn func() error) error {
	minimaxSettingsMu.Lock()
	defer minimaxSettingsMu.Unlock()
	return fn()
}

func GetMiniMaxVoiceWhitelistSnapshot() MiniMaxVoiceWhitelist {
	minimaxSettingsMu.RLock()
	defer minimaxSettingsMu.RUnlock()
	items := make(MiniMaxVoiceWhitelist, len(minimaxSettings.VoiceWhitelist))
	copy(items, minimaxSettings.VoiceWhitelist)
	return items
}

func snapshotMiniMaxPolicySettings() MiniMaxSettings {
	minimaxSettingsMu.RLock()
	defer minimaxSettingsMu.RUnlock()
	return MiniMaxSettings{
		Enabled:          minimaxSettings.Enabled,
		ModelRedirect:    cloneStringMap(minimaxSettings.ModelRedirect),
		EmotionPattern:   minimaxSettings.EmotionPattern,
		EmotionRedirect:  cloneStringMap(minimaxSettings.EmotionRedirect),
		ToneWordPattern:  minimaxSettings.ToneWordPattern,
		ToneWordRedirect: cloneStringMap(minimaxSettings.ToneWordRedirect),
		VoiceRedirect:    cloneStringMap(minimaxSettings.VoiceRedirect),
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(source))
	for k, v := range source {
		cloned[k] = v
	}
	return cloned
}

func ApplyMiniMaxTTSPolicy(model, voice, text, outputFormat string) MiniMaxTTSPolicyResult {
	cfg := snapshotMiniMaxPolicySettings()
	result := MiniMaxTTSPolicyResult{
		Enabled:      cfg.Enabled,
		Model:        model,
		Voice:        voice,
		Text:         text,
		OutputFormat: outputFormat,
	}
	if !cfg.Enabled {
		return result
	}
	result.Model = ApplyMiniMaxModelRedirect(model, &cfg)
	result.Voice = ApplyMiniMaxVoiceRedirect(voice, &cfg)
	emotion, cleaned := ExtractMiniMaxEmotion(text, cfg.EmotionPattern, cfg.EmotionRedirect)
	result.Emotion = emotion
	result.Text = ReplaceMiniMaxToneWords(cleaned, cfg.ToneWordPattern, cfg.ToneWordRedirect)
	return result
}

func ApplyMiniMaxModelRedirect(originModel string, cfg *MiniMaxSettings) string {
	if cfg == nil || cfg.ModelRedirect == nil {
		return originModel
	}
	if mapped, ok := cfg.ModelRedirect[originModel]; ok && mapped != "" {
		return mapped
	}
	return originModel
}

func ApplyMiniMaxVoiceRedirect(voice string, cfg *MiniMaxSettings) string {
	if cfg == nil || cfg.VoiceRedirect == nil {
		return voice
	}
	if mapped, ok := cfg.VoiceRedirect[voice]; ok && mapped != "" {
		return mapped
	}
	return voice
}

func IsMiniMaxVoiceWhitelistEnabled() bool {
	minimaxSettingsMu.RLock()
	defer minimaxSettingsMu.RUnlock()
	return len(minimaxSettings.VoiceWhitelist) > 0
}

func ValidateMiniMaxVoiceAllowed(voice string) error {
	minimaxSettingsMu.RLock()
	defer minimaxSettingsMu.RUnlock()
	if len(minimaxSettings.VoiceWhitelist) == 0 {
		return nil
	}
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return fmt.Errorf("minimax voice is not allowed: empty voice")
	}
	index := sort.SearchStrings([]string(minimaxSettings.VoiceWhitelist), voice)
	if index < len(minimaxSettings.VoiceWhitelist) && minimaxSettings.VoiceWhitelist[index] == voice {
		return nil
	}
	return fmt.Errorf("minimax voice is not allowed: %s", voice)
}

func ExtractMiniMaxEmotion(text string, pattern string, redirect map[string]string) (emotion string, cleaned string) {
	if pattern == "" {
		return "", text
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", text
	}
	// 多个标签时，仅第一个匹配的 emotion 值落地到 voice_setting.emotion。
	matches := re.FindStringSubmatch(text)
	var emotionValue string
	if len(matches) >= 2 {
		emotionValue = resolveMiniMaxTagValue(matches[1], redirect)
	} else if len(matches) == 1 {
		if tagValue := ExtractMiniMaxParenContent(matches[0]); tagValue != "" {
			emotionValue = resolveMiniMaxTagValue(tagValue, redirect)
		}
	}
	// 清洗：当正则带第 2 个捕获组（如 <tts emotion="...">text</tts>）时，
	// 保留捕获的正文，只剥离标签包裹；否则整体删除匹配。
	cleaned = replaceMiniMaxEmotionTags(text, re)
	cleaned = strings.TrimSpace(cleaned)
	return emotionValue, cleaned
}

// resolveMiniMaxTagValue 解析标签值到最终的 emotion：
// 映射表为空时直接使用捕获到的标签值；非空时优先取映射值，未命中返回空串。
func resolveMiniMaxTagValue(tagValue string, redirect map[string]string) string {
	if tagValue == "" {
		return ""
	}
	if len(redirect) == 0 {
		return tagValue
	}
	if mapped, ok := redirect[tagValue]; ok && mapped != "" {
		return mapped
	}
	return ""
}

// replaceMiniMaxEmotionTags 按正则捕获组数量决定如何清洗文本。
// 至少 2 个捕获组时用第 2 组正文替换整个标签（保留正文），否则删除整个匹配。
func replaceMiniMaxEmotionTags(text string, re *regexp.Regexp) string {
	if re.NumSubexp() >= 2 {
		return re.ReplaceAllString(text, "${2}")
	}
	return re.ReplaceAllString(text, "")
}

func ReplaceMiniMaxToneWords(text string, pattern string, redirect map[string]string) string {
	if pattern == "" {
		return text
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return text
	}
	// 收集所有映射目标值（替换值）：若用户直接传入某个替换值，则整个 (替换值)
	// 标签会被删除，避免再次发给上游或进入日志文本。替换是单趟处理，不会对
	// 本轮替换出的 (替换值) 重复扫描。
	targetValues := make(map[string]struct{}, len(redirect))
	for _, v := range redirect {
		if v != "" {
			targetValues[v] = struct{}{}
		}
	}
	return re.ReplaceAllStringFunc(text, func(match string) string {
		tagValue := ExtractMiniMaxParenContent(match)
		if tagValue == "" {
			return match
		}
		if _, isTarget := targetValues[tagValue]; isTarget {
			return ""
		}
		if mapped, ok := redirect[tagValue]; ok && mapped != "" {
			return "(" + mapped + ")"
		}
		return match
	})
}

func ExtractMiniMaxParenContent(s string) string {
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return ""
	}
	return s[1 : len(s)-1]
}

func IsMiniMaxStringMapOption(key string) bool {
	switch key {
	case "minimax.model_redirect", "minimax.emotion_redirect",
		"minimax.tone_word_redirect", "minimax.voice_redirect":
		return true
	default:
		return false
	}
}

func IsMiniMaxStringArrayOption(key string) bool {
	return key == "minimax.voice_whitelist"
}

func IsMiniMaxLargeOption(key string) bool {
	return IsMiniMaxStringMapOption(key) || IsMiniMaxStringArrayOption(key)
}

// ValidateMiniMaxOptionValue 校验 minimax.* 系统设置项的值。
// 在 controller.UpdateOption 中保存前调用，校验失败返回错误。
// 不属于 minimax.* 的 key 直接返回 nil。
func ValidateMiniMaxOptionValue(key, value string) error {
	switch key {
	case "minimax.model_redirect", "minimax.emotion_redirect",
		"minimax.tone_word_redirect", "minimax.voice_redirect":
		return validateStringMap(value)
	case "minimax.voice_whitelist":
		_, _, err := parseMiniMaxVoiceWhitelistJSON(value)
		return err
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

func NormalizeMiniMaxOptionValue(key, value string) (string, error) {
	if key != "minimax.voice_whitelist" {
		return value, ValidateMiniMaxOptionValue(key, value)
	}
	voices, emptyObject, err := parseMiniMaxVoiceWhitelistJSON(value)
	if err != nil {
		return "", err
	}
	if len(voices) == 0 {
		if emptyObject {
			return "{}", nil
		}
		return "[]", nil
	}
	bytes, err := common.Marshal(voices)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
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

func parseMiniMaxVoiceWhitelistJSON(value string) ([]string, bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, false, fmt.Errorf("invalid JSON for voice whitelist: empty value")
	}
	if trimmed == "null" {
		return nil, false, fmt.Errorf("invalid voice whitelist: use JSON array or {} to disable")
	}
	if len(trimmed) > maxMiniMaxVoiceWhitelistBytes {
		return nil, false, fmt.Errorf("voice whitelist JSON exceeds maximum size: %d bytes", maxMiniMaxVoiceWhitelistBytes)
	}
	var parsed any
	if err := common.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, false, fmt.Errorf("invalid JSON for voice whitelist: %w", err)
	}

	if emptyObject, ok := parsed.(map[string]any); ok {
		if len(emptyObject) == 0 {
			return []string{}, true, nil
		}
		return nil, false, fmt.Errorf("invalid voice whitelist: use JSON array or {} to disable")
	}

	var voices []string
	if err := common.Unmarshal([]byte(trimmed), &voices); err != nil {
		return nil, false, fmt.Errorf("invalid JSON for voice whitelist array: %w", err)
	}
	if len(voices) > maxMiniMaxVoiceWhitelistEntries {
		return nil, false, fmt.Errorf("voice whitelist exceeds maximum entries: %d", maxMiniMaxVoiceWhitelistEntries)
	}

	seen := make(map[string]struct{}, len(voices))
	normalized := make([]string, 0, len(voices))
	for _, voice := range voices {
		voice = strings.TrimSpace(voice)
		if voice == "" {
			return nil, false, fmt.Errorf("voice whitelist contains empty voice id")
		}
		if len(voice) > maxMiniMaxVoiceIDLength {
			return nil, false, fmt.Errorf("voice id exceeds maximum length: %d", maxMiniMaxVoiceIDLength)
		}
		if _, exists := seen[voice]; exists {
			continue
		}
		seen[voice] = struct{}{}
		normalized = append(normalized, voice)
	}
	sort.Strings(normalized)
	return normalized, false, nil
}
