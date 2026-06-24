package model_setting

import "testing"

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
