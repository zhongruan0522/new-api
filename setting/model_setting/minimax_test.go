package model_setting

import (
	"regexp"
	"testing"
)

func TestValidateMiniMaxOptionValue(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		// 非相关 key 直接放行
		{"unrelated_key", "other.foo", "{}", false},
		// map 字段：空串必须报错（清空请用 "{}"，避免 ConfigManager 静默跳过）
		{"model_redirect_empty", "minimax.model_redirect", "", true},
		// map 字段：合法 map[string]string
		{"model_redirect_valid", "minimax.model_redirect", `{"tts-1-hd": "speech-02-hd"}`, false},
		// map 字段：数字值必须报错（unmarshal 到 map[string]string 失败）
		{"model_redirect_number_value", "minimax.model_redirect", `{"a": 1}`, true},
		// map 字段：嵌套对象值必须报错
		{"model_redirect_object_value", "minimax.model_redirect", `{"a": {"b": "c"}}`, true},
		// map 字段：纯字符串顶层必须报错
		{"emotion_redirect_string", "minimax.emotion_redirect", `"hello"`, true},
		// map 字段：非法 JSON 必须报错
		{"tone_word_redirect_invalid_json", "minimax.tone_word_redirect", `{not json}`, true},
		// 已废弃的音色白名单/重定向 key：不再作为 map/array 校验，直接放行（写入时由控制器拒绝）
		{"voice_whitelist_legacy_pass_through", "minimax.voice_whitelist", `["a"]`, false},
		{"voice_redirect_legacy_pass_through", "minimax.voice_redirect", `{"a": "b"}`, false},
		// 正则字段：空串放行
		{"emotion_pattern_empty", "minimax.emotion_pattern", "", false},
		// 正则字段：合法正则放行
		{"emotion_pattern_valid", "minimax.emotion_pattern", `\((happy|sad)\)`, false},
		// 正则字段：非法正则必须报错
		{"emotion_pattern_invalid", "minimax.emotion_pattern", `(`, true},
		{"tone_word_pattern_invalid", "minimax.tone_word_pattern", `*invalid`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMiniMaxOptionValue(tc.key, tc.value)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for key=%q value=%q, got nil", tc.key, tc.value)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for key=%q value=%q: %v", tc.key, tc.value, err)
			}
		})
	}
}

func TestIsMiniMaxLegacyRemovedOption(t *testing.T) {
	if !IsMiniMaxLegacyRemovedOption("minimax.voice_whitelist") {
		t.Fatalf("minimax.voice_whitelist should be a removed legacy option")
	}
	if !IsMiniMaxLegacyRemovedOption("minimax.voice_redirect") {
		t.Fatalf("minimax.voice_redirect should be a removed legacy option")
	}
	if IsMiniMaxLegacyRemovedOption("minimax.model_redirect") {
		t.Fatalf("minimax.model_redirect should not be treated as removed")
	}
}

func TestCustomVoiceConfigDefaults(t *testing.T) {
	// 确保默认配置安全：定制音色默认关闭、白名单默认关闭
	if IsCustomVoiceEnabled() {
		t.Fatalf("custom voice should be disabled by default")
	}
	if IsMiniMaxVoiceWhitelistEnabled() {
		t.Fatalf("voice whitelist should be disabled by default")
	}
	group, billing := GetCustomVoiceConfig()
	if group != "" || billing != "" {
		t.Fatalf("custom voice group/billing model id should be empty by default, got %q/%q", group, billing)
	}
}

// ---------- 内置默认正则 ----------

func TestMiniMaxDefaultPatternsCompile(t *testing.T) {
	if defaultMiniMaxSettings.EmotionPattern == "" || defaultMiniMaxSettings.ToneWordPattern == "" {
		t.Fatalf("default EmotionPattern/ToneWordPattern must be non-empty")
	}
	if _, err := regexp.Compile(defaultMiniMaxSettings.EmotionPattern); err != nil {
		t.Fatalf("default EmotionPattern invalid: %v", err)
	}
	if _, err := regexp.Compile(defaultMiniMaxSettings.ToneWordPattern); err != nil {
		t.Fatalf("default ToneWordPattern invalid: %v", err)
	}
}

// ---------- ExtractMiniMaxEmotion：<tts emotion> 形式 ----------

func TestExtractMiniMaxEmotion_TTSEmotionFirstWinsKeepsText(t *testing.T) {
	emotion, cleaned := ExtractMiniMaxEmotion(
		`<tts emotion="happy">你好</tts><tts emotion="sad">世界</tts>`,
		`<tts\s+emotion="([^"]+)">([\s\S]*?)</tts>`,
		map[string]string{},
	)
	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy (first match)", emotion)
	}
	if cleaned != "你好世界" {
		t.Errorf("cleaned = %q, want 你好世界", cleaned)
	}
}

func TestExtractMiniMaxEmotion_TTSEmotionMapping(t *testing.T) {
	emotion, cleaned := ExtractMiniMaxEmotion(
		`前缀<tts emotion="高兴">真实文本</tts>后缀`,
		`<tts\s+emotion="([^"]+)">([\s\S]*?)</tts>`,
		map[string]string{"高兴": "happy"},
	)
	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy", emotion)
	}
	if cleaned != "前缀真实文本后缀" {
		t.Errorf("cleaned = %q, want 前缀真实文本后缀", cleaned)
	}
}

func TestExtractMiniMaxEmotion_TTSEmotionUnmappedIgnored(t *testing.T) {
	emotion, cleaned := ExtractMiniMaxEmotion(
		`<tts emotion="unknown">文本</tts>`,
		`<tts\s+emotion="([^"]+)">([\s\S]*?)</tts>`,
		map[string]string{"happy": "happy"},
	)
	if emotion != "" {
		t.Errorf("emotion = %q, want empty (unmapped)", emotion)
	}
	if cleaned != "文本" {
		t.Errorf("cleaned = %q, want 文本", cleaned)
	}
}

// ---------- ReplaceMiniMaxToneWords：替换值直传删除 ----------

func TestReplaceMiniMaxToneWords_SourceKeyStillReplaced(t *testing.T) {
	got := ReplaceMiniMaxToneWords("前(111)后", `\(([^()]+)\)`, map[string]string{"111": "222"})
	if got != "前(222)后" {
		t.Errorf("got %q, want 前(222)后", got)
	}
}

func TestReplaceMiniMaxToneWords_DeleteDirectTargetValue(t *testing.T) {
	got := ReplaceMiniMaxToneWords("前(222)后", `\(([^()]+)\)`, map[string]string{"111": "222"})
	if got != "前后" {
		t.Errorf("got %q, want 前后 (target value deleted)", got)
	}
}

func TestReplaceMiniMaxToneWords_OnlyTargetDeleted(t *testing.T) {
	got := ReplaceMiniMaxToneWords(
		"a(111)b(222)c(333)d",
		`\(([^()]+)\)`,
		map[string]string{"111": "222"},
	)
	if got != "a(222)bc(333)d" {
		t.Errorf("got %q, want a(222)bc(333)d", got)
	}
}

// ---------- 组合：ApplyMiniMaxTTSPolicy ----------

func TestApplyMiniMaxTTSPolicy_TTSEmotionPlusParenTag(t *testing.T) {
	prev := *GetMiniMaxSettings()
	defer func() { *GetMiniMaxSettings() = prev }()
	*GetMiniMaxSettings() = MiniMaxSettings{
		Enabled:         true,
		EmotionPattern:  `<tts\s+emotion="([^"]+)">([\s\S]*?)</tts>`,
		EmotionRedirect: map[string]string{},
		ToneWordPattern: `\(([^()]+)\)`,
		ToneWordRedirect: map[string]string{},
	}
	res := ApplyMiniMaxTTSPolicy("speech-02-hd", "alloy", `<tts emotion="happy">文本(laugh)</tts>`, "mp3")
	if res.Emotion != "happy" {
		t.Errorf("Emotion = %q, want happy", res.Emotion)
	}
	if res.Text != "文本(laugh)" {
		t.Errorf("Text = %q, want 文本(laugh)", res.Text)
	}
	// voice 字段不再由设置层重定向，应原样返回
	if res.Voice != "alloy" {
		t.Errorf("Voice = %q, want alloy (no settings-level redirect)", res.Voice)
	}
}
