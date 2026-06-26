package model_setting

import (
	"regexp"
	"strings"
	"testing"
)

func TestValidateMiniMaxOptionValue(t *testing.T) {
	oversizedVoiceWhitelist := `[` + `"` + strings.Repeat("a", maxMiniMaxVoiceWhitelistBytes) + `"` + `]`
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
		// map 字段：数组顶层必须报错
		{"voice_redirect_array", "minimax.voice_redirect", `["a", "b"]`, true},
		// map 字段：纯字符串顶层必须报错
		{"emotion_redirect_string", "minimax.emotion_redirect", `"hello"`, true},
		// map 字段：非法 JSON 必须报错
		{"tone_word_redirect_invalid_json", "minimax.tone_word_redirect", `{not json}`, true},
		// 音色白名单：{} 和 [] 都表示关闭
		{"voice_whitelist_empty_object", "minimax.voice_whitelist", `{}`, false},
		{"voice_whitelist_empty_array", "minimax.voice_whitelist", `[]`, false},
		// 音色白名单：合法数组放行
		{"voice_whitelist_valid", "minimax.voice_whitelist", `["male-qn-qingse", "female-shaonv"]`, false},
		// 音色白名单：空串、非字符串、非空对象、空白项必须报错
		{"voice_whitelist_empty", "minimax.voice_whitelist", ``, true},
		{"voice_whitelist_number", "minimax.voice_whitelist", `[1]`, true},
		{"voice_whitelist_object", "minimax.voice_whitelist", `{"a":"b"}`, true},
		{"voice_whitelist_blank_item", "minimax.voice_whitelist", `[" "]`, true},
		{"voice_whitelist_invalid_json", "minimax.voice_whitelist", `{not json}`, true},
		{"voice_whitelist_too_large", "minimax.voice_whitelist", oversizedVoiceWhitelist, true},
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

func TestNormalizeMiniMaxVoiceWhitelist(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"empty_object", `{}`, `{}`},
		{"empty_array", `[]`, `[]`},
		{"sort_and_deduplicate", `["b","a","b"]`, `["a","b"]`},
		{"trim_items", `[" b ","a"]`, `["a","b"]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeMiniMaxOptionValue("minimax.voice_whitelist", tc.value)
			if err != nil {
				t.Fatalf("NormalizeMiniMaxOptionValue error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeMiniMaxOptionValue = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestValidateMiniMaxVoiceAllowed(t *testing.T) {
	prev := *GetMiniMaxSettings()
	defer func() {
		*GetMiniMaxSettings() = prev
	}()

	*GetMiniMaxSettings() = MiniMaxSettings{VoiceWhitelist: MiniMaxVoiceWhitelist{}}
	if err := ValidateMiniMaxVoiceAllowed("anything"); err != nil {
		t.Fatalf("empty whitelist should allow any voice: %v", err)
	}

	*GetMiniMaxSettings() = MiniMaxSettings{VoiceWhitelist: MiniMaxVoiceWhitelist{"a", "b"}}
	if err := ValidateMiniMaxVoiceAllowed("b"); err != nil {
		t.Fatalf("allowed voice rejected: %v", err)
	}
	if err := ValidateMiniMaxVoiceAllowed("c"); err == nil {
		t.Fatalf("disallowed voice should be rejected")
	}
	if err := ValidateMiniMaxVoiceAllowed(""); err == nil {
		t.Fatalf("empty voice should be rejected when whitelist is enabled")
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
	// 多个 <tts emotion> 时，仅第一个 emotion 落地；正则带第 2 个捕获组时保留正文
	emotion, cleaned := ExtractMiniMaxEmotion(
		`<tts emotion="happy">你好</tts><tts emotion="sad">世界</tts>`,
		`<tts\s+emotion="([^"]+)">([\s\S]*?)</tts>`,
		map[string]string{}, // 空映射直接用捕获值
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
	// 非空映射表 + 未命中：emotion 为空，正文仍保留
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
	// 源 key (111) 命中映射 -> 替换为 (222)
	got := ReplaceMiniMaxToneWords("前(111)后", `\(([^()]+)\)`, map[string]string{"111": "222"})
	if got != "前(222)后" {
		t.Errorf("got %q, want 前(222)后", got)
	}
}

func TestReplaceMiniMaxToneWords_DeleteDirectTargetValue(t *testing.T) {
	// 用户直接传入映射目标值 (222) -> 整个标签被删除
	got := ReplaceMiniMaxToneWords("前(222)后", `\(([^()]+)\)`, map[string]string{"111": "222"})
	if got != "前后" {
		t.Errorf("got %q, want 前后 (target value deleted)", got)
	}
}

func TestReplaceMiniMaxToneWords_OnlyTargetDeleted(t *testing.T) {
	// 未配置映射的标签保持原样；只有映射目标值被删除
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
	// 用户样例：<tts emotion="happy">文本(laugh)</tts>
	// emotion 落地到 voice_setting.emotion；括号语气词无映射时保留
	prev := *GetMiniMaxSettings()
	defer func() { *GetMiniMaxSettings() = prev }()
	*GetMiniMaxSettings() = MiniMaxSettings{
		Enabled:          true,
		EmotionPattern:   `<tts\s+emotion="([^"]+)">([\s\S]*?)</tts>`,
		EmotionRedirect:  map[string]string{},
		ToneWordPattern:  `\(([^()]+)\)`,
		ToneWordRedirect: map[string]string{},
	}
	res := ApplyMiniMaxTTSPolicy("speech-02-hd", "alloy", `<tts emotion="happy">文本(laugh)</tts>`, "mp3")
	if res.Emotion != "happy" {
		t.Errorf("Emotion = %q, want happy", res.Emotion)
	}
	if res.Text != "文本(laugh)" {
		t.Errorf("Text = %q, want 文本(laugh)", res.Text)
	}
}
