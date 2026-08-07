package openai

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/NookMux/NookMux/common"
	channelconstant "github.com/NookMux/NookMux/constant"
	"github.com/NookMux/NookMux/dto"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	relayconstant "github.com/NookMux/NookMux/relay/constant"
	"github.com/NookMux/NookMux/setting/model_setting"
)

func withMiniMaxSettings(t *testing.T, cfg model_setting.MiniMaxSettings, fn func()) {
	t.Helper()
	prev := *model_setting.GetMiniMaxSettings()
	defer func() {
		*model_setting.GetMiniMaxSettings() = prev
	}()
	*model_setting.GetMiniMaxSettings() = cfg
	fn()
}

// TestConvertAudioRequestMiniMaxOpenAIPathAppliesSystemPolicy 验证 OpenAI 路径的 MiniMax TTS：
// 模型重定向、情绪、语气词策略生效。音色重定向已迁移到数据库表，此处无 DB 时保留原音色。
func TestConvertAudioRequestMiniMaxOpenAIPathAppliesSystemPolicy(t *testing.T) {
	cfg := model_setting.MiniMaxSettings{
		Enabled:          true,
		ModelRedirect:    map[string]string{"tts-1-hd": "speech-02-hd"},
		EmotionPattern:   `\((happy)\)`,
		EmotionRedirect:  map[string]string{"happy": "happy"},
		ToneWordPattern:  `\((laughs)\)`,
		ToneWordRedirect: map[string]string{"laughs": "aaa"},
	}

	withMiniMaxSettings(t, cfg, func() {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/audio/speech", nil)

		info := &relaycommon.RelayInfo{
			RelayMode: relayconstant.RelayModeAudioSpeech,
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       channelconstant.ChannelTypeMiniMax,
				UpstreamModelName: "tts-1-hd",
			},
		}
		request := dto.AudioRequest{
			Model:          "tts-1-hd",
			Input:          "(happy)hello(laughs)",
			Voice:          "alloy",
			ResponseFormat: "mp3",
		}

		a := &Adaptor{}
		reader, err := a.ConvertAudioRequest(c, info, request)
		if err != nil {
			t.Fatalf("ConvertAudioRequest error: %v", err)
		}
		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var got dto.AudioRequest
		if err := common.Unmarshal(body, &got); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		if got.Model != "speech-02-hd" {
			t.Errorf("Model = %q, want speech-02-hd", got.Model)
		}
		// 无 DB 时音色不被重定向，保留原值。
		if got.Voice != "alloy" {
			t.Errorf("Voice = %q, want alloy (no DB redirect in unit test)", got.Voice)
		}
		if got.Input != "hello(aaa)" {
			t.Errorf("Input = %q, want hello(aaa)", got.Input)
		}
	})
}

// OpenAI /v1/audio/speech 转 MiniMax：<tts emotion> 剥离后保留正文，
// 括号语气词按映射原地替换。
func TestConvertAudioRequestMiniMaxOpenAIPathTTSEmotion(t *testing.T) {
	cfg := model_setting.MiniMaxSettings{
		Enabled:          true,
		EmotionPattern:   `<tts\s+emotion="([^"]+)">([\s\S]*?)</tts>`,
		EmotionRedirect:  map[string]string{},
		ToneWordPattern:  `\(([^()]+)\)`,
		ToneWordRedirect: map[string]string{"laugh": "笑"},
	}

	withMiniMaxSettings(t, cfg, func() {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/audio/speech", nil)

		info := &relaycommon.RelayInfo{
			RelayMode: relayconstant.RelayModeAudioSpeech,
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       channelconstant.ChannelTypeMiniMax,
				UpstreamModelName: "tts-1-hd",
			},
		}
		request := dto.AudioRequest{
			Model:          "tts-1-hd",
			Input:          `<tts emotion="happy">文本(laugh)</tts>`,
			Voice:          "alloy",
			ResponseFormat: "mp3",
		}

		a := &Adaptor{}
		reader, err := a.ConvertAudioRequest(c, info, request)
		if err != nil {
			t.Fatalf("ConvertAudioRequest error: %v", err)
		}
		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var got dto.AudioRequest
		if err := common.Unmarshal(body, &got); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if got.Input != "文本(笑)" {
			t.Errorf("Input = %q, want 文本(笑)", got.Input)
		}
	})
}
