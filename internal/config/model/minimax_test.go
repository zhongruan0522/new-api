package model

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

// ---------- GetCustomVoiceTagsSnapshot：只暴露 redirect map 的 key ----------

func TestGetCustomVoiceTagsSnapshot_DefaultsAndKeysOnly(t *testing.T) {
	prev := *GetMiniMaxSettings()
	defer func() { *GetMiniMaxSettings() = prev }()

	*GetMiniMaxSettings() = MiniMaxSettings{
		Enabled:          true,
		EmotionPattern:   `\(([^()]+)\)`,
		EmotionRedirect:  map[string]string{"1": "2", "b-emo": "happy"},
		ToneWordPattern:  `\(([^()]+)\)`,
		ToneWordRedirect: map[string]string{"z-ton": "target"},
	}

	snap := GetCustomVoiceTagsSnapshot()
	if !snap.Enabled {
		t.Fatalf("Enabled = false, want true")
	}
	if snap.EmotionPattern != `\(([^()]+)\)` {
		t.Fatalf("EmotionPattern = %q, want \\(([^()]+)\\)", snap.EmotionPattern)
	}
	if snap.ToneWordPattern != `\(([^()]+)\)` {
		t.Fatalf("ToneWordPattern = %q, want \\(([^()]+)\\)", snap.ToneWordPattern)
	}
	// emotion_tags 只应包含 redirect map 的 key（已排序）
	wantEmotion := []string{"1", "b-emo"}
	if len(snap.EmotionTags) != len(wantEmotion) {
		t.Fatalf("EmotionTags len = %d, want %d (%v)", len(snap.EmotionTags), len(wantEmotion), snap.EmotionTags)
	}
	for i, want := range wantEmotion {
		if snap.EmotionTags[i] != want {
			t.Errorf("EmotionTags[%d] = %q, want %q", i, snap.EmotionTags[i], want)
		}
	}
	// tone_word_tags 只应包含 redirect map 的 key
	wantTone := []string{"z-ton"}
	if len(snap.ToneWordTags) != len(wantTone) {
		t.Fatalf("ToneWordTags len = %d, want %d (%v)", len(snap.ToneWordTags), len(wantTone), snap.ToneWordTags)
	}
	for i, want := range wantTone {
		if snap.ToneWordTags[i] != want {
			t.Errorf("ToneWordTags[%d] = %q, want %q", i, snap.ToneWordTags[i], want)
		}
	}
}

func TestGetCustomVoiceTagsSnapshot_EmptyMapsReturnNil(t *testing.T) {
	prev := *GetMiniMaxSettings()
	defer func() { *GetMiniMaxSettings() = prev }()

	*GetMiniMaxSettings() = MiniMaxSettings{
		Enabled:          true,
		EmotionRedirect:  map[string]string{},
		ToneWordRedirect: map[string]string{},
	}
	snap := GetCustomVoiceTagsSnapshot()
	if snap.EmotionTags != nil {
		t.Errorf("EmotionTags = %v, want nil when map empty", snap.EmotionTags)
	}
	if snap.ToneWordTags != nil {
		t.Errorf("ToneWordTags = %v, want nil when map empty", snap.ToneWordTags)
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

// ---------- 新默认正则（emotion 属性可选）：<tts> 纯标签也要剥离 ----------

// 新默认正则让 emotion 属性可选：带属性时提取 emotion 值并保留内部文本，
// 不带属性（纯 <tts>...</tts>）时也剥离外层标签只保留内部文本，
// 避免“对外的 <tts> 标签”原样透传给上游。
const optionalEmotionPattern = `<tts(?:\s+emotion="([^"]+)")?>([\s\S]*?)</tts>`

func TestExtractMiniMaxEmotion_PlainTtsTagStripped(t *testing.T) {
	// 纯 <tts>标签，无 emotion 属性：emotion 为空，外层标签被剥离，内部文本保留
	emotion, cleaned := ExtractMiniMaxEmotion(
		`<tts>在线上真实用户流量中，速度提升了 60%。</tts>`,
		optionalEmotionPattern,
		map[string]string{},
	)
	if emotion != "" {
		t.Errorf("emotion = %q, want empty (no emotion attribute)", emotion)
	}
	if cleaned != "在线上真实用户流量中，速度提升了 60%。" {
		t.Errorf("cleaned = %q, want inner text with tags stripped", cleaned)
	}
}

func TestExtractMiniMaxEmotion_OptionalEmotionStillExtracted(t *testing.T) {
	// 带 emotion 属性时仍正常提取并剥离标签，行为与旧默认正则一致
	emotion, cleaned := ExtractMiniMaxEmotion(
		`<tts emotion="happy">你好</tts>`,
		optionalEmotionPattern,
		map[string]string{},
	)
	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy", emotion)
	}
	if cleaned != "你好" {
		t.Errorf("cleaned = %q, want 你好", cleaned)
	}
}

func TestExtractMiniMaxEmotion_MixedPlainAndEmotionTags(t *testing.T) {
	// 混合：第一个是带 emotion 的标签，第二个是纯标签
	// emotion 取第一个匹配的值，所有 <tts> 标签都被剥离
	emotion, cleaned := ExtractMiniMaxEmotion(
		`<tts emotion="happy">开心</tts>中间<tts>普通</tts>`,
		optionalEmotionPattern,
		map[string]string{},
	)
	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy (first match)", emotion)
	}
	if cleaned != "开心中间普通" {
		t.Errorf("cleaned = %q, want 开心中间普通", cleaned)
	}
}

func TestExtractMiniMaxEmotion_PlainTagWithParenToneWord(t *testing.T) {
	// 用户真实场景变体：<tts>纯标签包裹含语气词的文本；emotion 为空，
	// 标签剥离后由后续 ReplaceMiniMaxToneWords 处理 (laugh)
	emotion, cleaned := ExtractMiniMaxEmotion(
		`<tts>文本(laugh)</tts>`,
		optionalEmotionPattern,
		map[string]string{},
	)
	if emotion != "" {
		t.Errorf("emotion = %q, want empty", emotion)
	}
	if cleaned != "文本(laugh)" {
		t.Errorf("cleaned = %q, want 文本(laugh)", cleaned)
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
	// voice 字段不再由设置层重定向，应原样返回
	if res.Voice != "alloy" {
		t.Errorf("Voice = %q, want alloy (no settings-level redirect)", res.Voice)
	}
}

// 新默认正则（emotion 属性可选）端到端：用户真实场景。
// <tts emotion="xxxx">长文本</tts> -> emotion 取 xxxx，标签剥离后只保留内部长文本。
func TestApplyMiniMaxTTSPolicy_OptionalEmotionDefaultPattern(t *testing.T) {
	prev := *GetMiniMaxSettings()
	defer func() { *GetMiniMaxSettings() = prev }()
	*GetMiniMaxSettings() = MiniMaxSettings{
		Enabled:          true,
		EmotionPattern:   defaultMiniMaxSettings.EmotionPattern, // 新默认正则
		EmotionRedirect:  map[string]string{},                   // 空 map：emotion 直传 xxxx
		ToneWordPattern:  defaultMiniMaxSettings.ToneWordPattern,
		ToneWordRedirect: map[string]string{},
	}
	input := `<tts emotion="xxxx">在线上真实用户流量中，速度提升了 60% 至 85%。</tts>`
	wantText := "在线上真实用户流量中，速度提升了 60% 至 85%。"
	res := ApplyMiniMaxTTSPolicy("speech-02-hd", "alloy", input, "mp3")
	if res.Emotion != "xxxx" {
		t.Errorf("Emotion = %q, want xxxx", res.Emotion)
	}
	if res.Text != wantText {
		t.Errorf("Text = %q, want %q", res.Text, wantText)
	}
}

// 新默认正则：纯 <tts>无 emotion 标签也被剥离，不会原样透传给上游。
func TestApplyMiniMaxTTSPolicy_PlainTtsTagStripped(t *testing.T) {
	prev := *GetMiniMaxSettings()
	defer func() { *GetMiniMaxSettings() = prev }()
	*GetMiniMaxSettings() = MiniMaxSettings{
		Enabled:          true,
		EmotionPattern:   defaultMiniMaxSettings.EmotionPattern, // 新默认正则
		EmotionRedirect:  map[string]string{"happy": "happy"},   // 非空 map
		ToneWordPattern:  defaultMiniMaxSettings.ToneWordPattern,
		ToneWordRedirect: map[string]string{},
	}
	input := `<tts>纯标签包裹的文本</tts>`
	wantText := "纯标签包裹的文本"
	res := ApplyMiniMaxTTSPolicy("speech-02-hd", "alloy", input, "mp3")
	if res.Emotion != "" {
		t.Errorf("Emotion = %q, want empty (no emotion attribute)", res.Emotion)
	}
	if res.Text != wantText {
		t.Errorf("Text = %q, want %q (tags stripped, no leak to upstream)", res.Text, wantText)
	}
}
