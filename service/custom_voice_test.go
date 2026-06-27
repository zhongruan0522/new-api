package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/model_setting"
)

// withMiniMaxSettings 临时覆盖全局 MiniMax 设置，函数返回后恢复原值。
// 用于隔离 buildVoiceClonePayload 的策略相关测试。
func withMiniMaxSettings(cfg model_setting.MiniMaxSettings, fn func()) {
	prev := *model_setting.GetMiniMaxSettings()
	defer func() { *model_setting.GetMiniMaxSettings() = prev }()
	*model_setting.GetMiniMaxSettings() = cfg
	fn()
}

func resetCustomVoiceDemoAudioCacheForTest() {
	customVoiceDemoAudioCache.Lock()
	defer customVoiceDemoAudioCache.Unlock()
	customVoiceDemoAudioCache.items = make(map[customVoiceDemoAudioCacheKey]customVoiceDemoAudioCacheEntry)
}

func TestCustomVoiceDemoAudioCache_RegisterAndExpire(t *testing.T) {
	resetCustomVoiceDemoAudioCacheForTest()
	defer resetCustomVoiceDemoAudioCacheForTest()

	proxyURL := registerCustomVoiceDemoAudio(12, 34, "https://cdn.example.test/audio.mp3")
	if proxyURL != "/api/custom_voice/preview/34/audio" {
		t.Fatalf("proxyURL = %q, want /api/custom_voice/preview/34/audio", proxyURL)
	}
	got, err := getCustomVoiceDemoAudioURL(12, 34)
	if err != nil {
		t.Fatalf("get cached url failed: %v", err)
	}
	if got != "https://cdn.example.test/audio.mp3" {
		t.Fatalf("cached url = %q, want upstream url", got)
	}

	key := customVoiceDemoAudioCacheKey{userId: 12, recordId: 34}
	customVoiceDemoAudioCache.Lock()
	customVoiceDemoAudioCache.items[key] = customVoiceDemoAudioCacheEntry{
		url:       "https://cdn.example.test/audio.mp3",
		expiresAt: time.Now().Add(-time.Second),
	}
	customVoiceDemoAudioCache.Unlock()

	_, err = getCustomVoiceDemoAudioURL(12, 34)
	if !errors.Is(err, ErrCustomVoiceDemoAudioExpired) {
		t.Fatalf("expired cache err = %v, want ErrCustomVoiceDemoAudioExpired", err)
	}
	customVoiceDemoAudioCache.RLock()
	_, exists := customVoiceDemoAudioCache.items[key]
	customVoiceDemoAudioCache.RUnlock()
	if exists {
		t.Fatalf("expired cache entry should be deleted")
	}
}

// TestBuildVoiceClonePayload_NoPreviewText 无试听文本时只透传基础字段。
func TestBuildVoiceClonePayload_NoPreviewText(t *testing.T) {
	withMiniMaxSettings(model_setting.MiniMaxSettings{Enabled: true}, func() {
		payload := buildVoiceClonePayload("file-1", CustomVoicePreviewRequest{
			Model:   "tts-2-hd",
			VoiceId: "voice-id",
		})
		if payload["file_id"] != "file-1" {
			t.Errorf("file_id = %v, want file-1", payload["file_id"])
		}
		if _, ok := payload["text"]; ok {
			t.Errorf("text should not be set when preview text is empty")
		}
		if _, ok := payload["emotion"]; ok {
			t.Errorf("emotion should not be set when preview text is empty")
		}
		if _, ok := payload["model"]; ok {
			t.Errorf("model should not be set when preview text is empty")
		}
	})
}

// TestBuildVoiceClonePayload_PolicyDisabled 增强未启用时原样透传文本与模型。
func TestBuildVoiceClonePayload_PolicyDisabled(t *testing.T) {
	withMiniMaxSettings(model_setting.MiniMaxSettings{Enabled: false}, func() {
		payload := buildVoiceClonePayload("file-1", CustomVoicePreviewRequest{
			Model:       "tts-2-hd",
			VoiceId:     "voice-id",
			PreviewText: "(happy)你好",
		})
		if payload["text"] != "(happy)你好" {
			t.Errorf("text = %v, want (happy)你好 (passthrough when disabled)", payload["text"])
		}
		if payload["model"] != "tts-2-hd" {
			t.Errorf("model = %v, want tts-2-hd", payload["model"])
		}
		if _, ok := payload["emotion"]; ok {
			t.Errorf("emotion should not be set when disabled")
		}
	})
}

// TestBuildVoiceClonePayload_EmotionRedirect 用户输入源标签 (1)，
// 上游 payload 应使用重定向目标 emotion 值 (2)，并剥离文本中的情绪标签。
func TestBuildVoiceClonePayload_EmotionRedirect(t *testing.T) {
	withMiniMaxSettings(model_setting.MiniMaxSettings{
		Enabled:         true,
		EmotionPattern:  `\(([^()]+)\)`,
		EmotionRedirect: map[string]string{"1": "2"},
	}, func() {
		payload := buildVoiceClonePayload("file-1", CustomVoicePreviewRequest{
			Model:       "tts-2-hd",
			VoiceId:     "voice-id",
			PreviewText: "(1)你好",
		})
		if payload["emotion"] != "2" {
			t.Errorf("emotion = %v, want 2 (redirected value)", payload["emotion"])
		}
		if payload["text"] != "你好" {
			t.Errorf("text = %v, want 你好 (emotion tag stripped)", payload["text"])
		}
	})
}

// TestBuildVoiceClonePayload_ToneWordRedirect 用户输入源标签 (1)，
// 上游 payload 应使用重定向后的语气词值 (2)。
func TestBuildVoiceClonePayload_ToneWordRedirect(t *testing.T) {
	withMiniMaxSettings(model_setting.MiniMaxSettings{
		Enabled:          true,
		ToneWordPattern:  `\(([^()]+)\)`,
		ToneWordRedirect: map[string]string{"1": "2"},
	}, func() {
		payload := buildVoiceClonePayload("file-1", CustomVoicePreviewRequest{
			Model:       "tts-2-hd",
			VoiceId:     "voice-id",
			PreviewText: "前(1)后",
		})
		if payload["text"] != "前(2)后" {
			t.Errorf("text = %v, want 前(2)后 (redirected tone word)", payload["text"])
		}
	})
}

// TestBuildVoiceClonePayload_ModelRedirect 模型重定向表命中时使用上游模型名。
func TestBuildVoiceClonePayload_ModelRedirect(t *testing.T) {
	withMiniMaxSettings(model_setting.MiniMaxSettings{
		Enabled:       true,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
	}, func() {
		payload := buildVoiceClonePayload("file-1", CustomVoicePreviewRequest{
			Model:       "tts-1-hd",
			VoiceId:     "voice-id",
			PreviewText: "你好",
		})
		if payload["model"] != "speech-02-hd" {
			t.Errorf("model = %v, want speech-02-hd (redirected)", payload["model"])
		}
	})
}

// TestBuildVoiceClonePayload_ModelRedirectWhenEnhanceDisabled ensures custom
// voice always applies the administrator model redirect before calling MiniMax,
// even when the broader TTS enhance switch is disabled.
func TestBuildVoiceClonePayload_ModelRedirectWhenEnhanceDisabled(t *testing.T) {
	withMiniMaxSettings(model_setting.MiniMaxSettings{
		Enabled:       false,
		ModelRedirect: map[string]string{"tts-2-hd": "speech-2.8-hd"},
	}, func() {
		payload := buildVoiceClonePayload("file-1", CustomVoicePreviewRequest{
			Model:       "tts-2-hd",
			VoiceId:     "voice-id",
			PreviewText: "你好",
		})
		if payload["model"] != "speech-2.8-hd" {
			t.Errorf("model = %v, want speech-2.8-hd (redirected with enhance disabled)", payload["model"])
		}
	})
}

// TestCustomVoiceFileID_NumericRoundTrip covers MiniMax upload responses where
// file_id is a JSON number. The frontend still receives a string for display,
// but the upstream voice_clone payload must preserve the numeric JSON type.
func TestCustomVoiceFileID_NumericRoundTrip(t *testing.T) {
	var resp struct {
		File struct {
			FileId customVoiceFileID `json:"file_id"`
		} `json:"file"`
	}
	if err := common.Unmarshal([]byte(`{"file":{"file_id":123456789}}`), &resp); err != nil {
		t.Fatalf("unmarshal numeric file_id failed: %v", err)
	}
	if got := resp.File.FileId.String(); got != "123456789" {
		t.Fatalf("display file_id = %q, want 123456789", got)
	}

	payload := buildVoiceClonePayload(resp.File.FileId, CustomVoicePreviewRequest{VoiceId: "voice-id"})
	body, err := common.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal voice_clone payload failed: %v", err)
	}
	if !strings.Contains(string(body), `"file_id":123456789`) {
		t.Fatalf("payload should preserve numeric file_id, got %s", string(body))
	}
}

// TestResolveCustomVoiceGroupRatio 验证分组倍率解析包含动态倍率叠加，
// 且未配置动态倍率时返回纯分组倍率。
func TestResolveCustomVoiceGroupRatio(t *testing.T) {
	// "default" 分组默认倍率为 1.0。
	if r := resolveCustomVoiceGroupRatio("default", "tts-3-turbo"); r != 1.0 {
		t.Errorf("default group ratio = %v, want 1.0", r)
	}
	// 未配置分组时也返回默认 1.0（GetGroupRatio 找不到时回退 1）。
	if r := resolveCustomVoiceGroupRatio("nonexistent-group", "tts-3-turbo"); r != 1.0 {
		t.Errorf("nonexistent group ratio = %v, want 1.0", r)
	}
}
