package openai

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	channelconstant "github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	relayconstant "github.com/zhongruan0522/new-api/relay/constant"
	"github.com/zhongruan0522/new-api/setting/model_setting"
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

func TestConvertAudioRequestMiniMaxOpenAIPathAppliesSystemPolicy(t *testing.T) {
	cfg := model_setting.MiniMaxSettings{
		Enabled:         true,
		ModelRedirect:   map[string]string{"tts-1-hd": "speech-02-hd"},
		VoiceRedirect:   map[string]string{"alloy": "female-shaonv"},
		EmotionPattern:  `\((happy)\)`,
		EmotionRedirect: map[string]string{"happy": "happy"},
		ToneWordPattern: `\((laughs)\)`,
		ToneWordRedirect: map[string]string{
			"laughs": "aaa",
		},
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
		if got.Voice != "female-shaonv" {
			t.Errorf("Voice = %q, want female-shaonv", got.Voice)
		}
		if got.Input != "hello(aaa)" {
			t.Errorf("Input = %q, want hello(aaa)", got.Input)
		}
		if v := c.GetString("minimax_voice_id"); v != "female-shaonv" {
			t.Errorf("minimax_voice_id = %q, want female-shaonv", v)
		}
	})
}
