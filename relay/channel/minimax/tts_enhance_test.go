package minimax

import (
	"testing"

	"github.com/zhongruan0522/new-api/setting/model_setting"
)

// ---------- applyModelRedirect ----------

func TestApplyModelRedirect_Hit(t *testing.T) {
	cfg := &model_setting.MiniMaxSettings{
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
	}
	got := applyModelRedirect("tts-1-hd", cfg)
	if got != "speech-02-hd" {
		t.Errorf("got %q, want speech-02-hd", got)
	}
}

func TestApplyModelRedirect_Miss(t *testing.T) {
	cfg := &model_setting.MiniMaxSettings{
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
	}
	got := applyModelRedirect("tts-1", cfg)
	if got != "tts-1" {
		t.Errorf("got %q, want tts-1 (unmapped)", got)
	}
}

func TestApplyModelRedirect_EmptyTarget(t *testing.T) {
	cfg := &model_setting.MiniMaxSettings{
		ModelRedirect: map[string]string{"tts-1-hd": ""},
	}
	got := applyModelRedirect("tts-1-hd", cfg)
	if got != "tts-1-hd" {
		t.Errorf("got %q, want tts-1-hd (empty target treated as miss)", got)
	}
}

func TestApplyModelRedirect_NilCfg(t *testing.T) {
	got := applyModelRedirect("tts-1-hd", nil)
	if got != "tts-1-hd" {
		t.Errorf("got %q, want tts-1-hd", got)
	}
}

func TestApplyModelRedirect_NilMap(t *testing.T) {
	cfg := &model_setting.MiniMaxSettings{}
	got := applyModelRedirect("tts-1-hd", cfg)
	if got != "tts-1-hd" {
		t.Errorf("got %q, want tts-1-hd", got)
	}
}

// ---------- extractEmotion ----------

func TestExtractEmotion_SingleHit(t *testing.T) {
	emotion, cleaned := extractEmotion(
		"今天是不是很开心呀(happy)，当然了！",
		`\((happy|sad)\)`,
		map[string]string{"happy": "happy", "sad": "sad"},
	)
	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy", emotion)
	}
	if cleaned != "今天是不是很开心呀，当然了！" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestExtractEmotion_MultiTagsFirstWins(t *testing.T) {
	// 多个情绪标签时，取第一个匹配的映射值；但所有标签都被剥离
	emotion, cleaned := extractEmotion(
		"(happy)你好(sad)世界",
		`\((happy|sad)\)`,
		map[string]string{"happy": "happy", "sad": "sad"},
	)
	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy (first match)", emotion)
	}
	if cleaned != "你好世界" {
		t.Errorf("cleaned = %q, want 你好世界", cleaned)
	}
}

func TestExtractEmotion_TagPresentNoMapping(t *testing.T) {
	// 标签存在但无映射：emotion 空串，但标签仍被剥离
	emotion, cleaned := extractEmotion(
		"hello(angry)world",
		`\((happy|sad|angry)\)`,
		map[string]string{"happy": "happy"}, // angry 未配置
	)
	if emotion != "" {
		t.Errorf("emotion = %q, want empty (no mapping)", emotion)
	}
	if cleaned != "helloworld" {
		t.Errorf("cleaned = %q, want helloworld (tag stripped)", cleaned)
	}
}

func TestExtractEmotion_NoTag(t *testing.T) {
	emotion, cleaned := extractEmotion(
		"纯文本无标签",
		`\((happy|sad)\)`,
		map[string]string{"happy": "happy"},
	)
	if emotion != "" {
		t.Errorf("emotion = %q, want empty", emotion)
	}
	if cleaned != "纯文本无标签" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestExtractEmotion_EmptyPattern(t *testing.T) {
	emotion, cleaned := extractEmotion("hello(happy)world", "", map[string]string{"happy": "happy"})
	if emotion != "" {
		t.Errorf("emotion = %q, want empty", emotion)
	}
	if cleaned != "hello(happy)world" {
		t.Errorf("cleaned = %q, want original", cleaned)
	}
}

func TestExtractEmotion_InvalidRegex(t *testing.T) {
	// 非法正则不崩溃，返回原文本和空 emotion
	emotion, cleaned := extractEmotion("hello(happy)world", `(((`, map[string]string{"happy": "happy"})
	if emotion != "" {
		t.Errorf("emotion = %q, want empty", emotion)
	}
	if cleaned != "hello(happy)world" {
		t.Errorf("cleaned = %q, want original", cleaned)
	}
}

func TestExtractEmotion_ChineseMapping(t *testing.T) {
	emotion, cleaned := extractEmotion(
		"(高兴)今天天气真好",
		`\((高兴|伤心)\)`,
		map[string]string{"高兴": "happy", "伤心": "sad"},
	)
	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy", emotion)
	}
	if cleaned != "今天天气真好" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestExtractEmotion_NoCaptureGroup(t *testing.T) {
	// 正则无捕获组时，从整体匹配提取括号内容
	emotion, cleaned := extractEmotion(
		"(happy)hello",
		`\((?:happy|sad)\)`,
		map[string]string{"happy": "happy"},
	)
	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy", emotion)
	}
	if cleaned != "hello" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

// ---------- replaceToneWords ----------

func TestReplaceToneWords_Single(t *testing.T) {
	got := replaceToneWords("abc(laughs)def", `\((laughs|crying)\)`, map[string]string{"laughs": "aaa"})
	if got != "abc(aaa)def" {
		t.Errorf("got %q", got)
	}
}

func TestReplaceToneWords_Multi(t *testing.T) {
	got := replaceToneWords(
		"sadhsajkhdj(laughs)hcsiahbcjkbd(xxx)",
		`\((laughs|crying|xxx)\)`,
		map[string]string{"laughs": "aaa", "xxx": "bbb"},
	)
	if got != "sadhsajkhdj(aaa)hcsiahbcjkbd(bbb)" {
		t.Errorf("got %q", got)
	}
}

func TestReplaceToneWords_PartialMapping(t *testing.T) {
	// 部分命中，未命中的保留原标签
	got := replaceToneWords(
		"a(laughs)b(unknown)c",
		`\((laughs|unknown)\)`,
		map[string]string{"laughs": "aaa"}, // unknown 未配置
	)
	if got != "a(aaa)b(unknown)c" {
		t.Errorf("got %q", got)
	}
}

func TestReplaceToneWords_NoTag(t *testing.T) {
	got := replaceToneWords("纯文本", `\((laughs|crying)\)`, map[string]string{"laughs": "aaa"})
	if got != "纯文本" {
		t.Errorf("got %q", got)
	}
}

func TestReplaceToneWords_EmptyPattern(t *testing.T) {
	got := replaceToneWords("a(laughs)b", "", map[string]string{"laughs": "aaa"})
	if got != "a(laughs)b" {
		t.Errorf("got %q", got)
	}
}

func TestReplaceToneWords_InvalidRegex(t *testing.T) {
	got := replaceToneWords("a(laughs)b", `(((`, map[string]string{"laughs": "aaa"})
	if got != "a(laughs)b" {
		t.Errorf("got %q, want original on invalid regex", got)
	}
}

func TestReplaceToneWords_EmptyTarget(t *testing.T) {
	// 映射值为空串时视为未命中，保留原标签
	got := replaceToneWords("a(laughs)b", `\((laughs)\)`, map[string]string{"laughs": ""})
	if got != "a(laughs)b" {
		t.Errorf("got %q, want original (empty target = miss)", got)
	}
}

func TestReplaceToneWords_DeleteDirectTargetValue(t *testing.T) {
	// 源 key (111) 命中 -> 替换为 (222)；用户直接传入映射目标值 (222) -> 整个删除
	got := replaceToneWords("a(111)b(222)c", `\((111|222)\)`, map[string]string{"111": "222"})
	if got != "a(222)bc" {
		t.Errorf("got %q, want a(222)bc", got)
	}
}

// ---------- 组合场景：情绪剥离 + 语气词替换 ----------

func TestCombined_EmotionThenToneWord(t *testing.T) {
	cfg := &model_setting.MiniMaxSettings{
		Enabled:         true,
		EmotionPattern:  `\((happy|sad)\)`,
		EmotionRedirect: map[string]string{"happy": "happy"},
		ToneWordPattern: `\((laughs|crying)\)`,
		ToneWordRedirect: map[string]string{
			"laughs": "aaa",
			"crying": "bbb",
		},
	}

	text := "(happy)你好(laughs)世界"

	emotion, cleaned := extractEmotion(text, cfg.EmotionPattern, cfg.EmotionRedirect)
	cleaned = replaceToneWords(cleaned, cfg.ToneWordPattern, cfg.ToneWordRedirect)

	if emotion != "happy" {
		t.Errorf("emotion = %q, want happy", emotion)
	}
	if cleaned != "你好(aaa)世界" {
		t.Errorf("cleaned = %q, want 你好(aaa)世界", cleaned)
	}
}

// ---------- extractParenContent ----------

func TestExtractParenContent(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"(happy)", "happy"},
		{"(高兴)", "高兴"},
		{"()", ""},
		{"no paren", ""},
		{"(incomplete", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := extractParenContent(tc.input)
		if got != tc.want {
			t.Errorf("extractParenContent(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
