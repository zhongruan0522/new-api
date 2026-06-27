package service

import (
	"testing"

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
