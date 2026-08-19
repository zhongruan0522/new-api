package minimax

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/config/model"
	"github.com/NookMux/NookMux/internal/dto"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
)

// withMiniMaxSettings 临时替换全局 MiniMaxSettings，测试结束自动恢复。
func withMiniMaxSettings(t *testing.T, cfg model.MiniMaxSettings, fn func()) {
	t.Helper()
	prev := *model.GetMiniMaxSettings()
	defer func() {
		*model.GetMiniMaxSettings() = prev
	}()
	*model.GetMiniMaxSettings() = cfg
	fn()
}

func newConvertAudioContext() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/audio/speech", nil)
	return c
}

func decodeTTSBody(t *testing.T, r io.Reader) MiniMaxTTSRequest {
	t.Helper()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var req MiniMaxTTSRequest
	if err := jsonx.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	return req
}

// TestConvertAudioRequest_MetadataCannotOverrideModel 验证 issue #107 修复：
// 当 metadata 包含 model 且 cfg.Enabled=true 配置了模型重定向时，管理员强制策略必须覆盖 metadata。
// （音色重定向已迁移到数据库，此处仅校验 model 字段。）
func TestConvertAudioRequest_MetadataCannotOverrideModel(t *testing.T) {
	cfg := model.MiniMaxSettings{
		Enabled:       true,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
	}
	metadata := []byte(`{"model":"evil-model","voice_setting":{"voice_id":"evil-voice"}}`)

	info := &relaycommon.RelayInfo{
		RelayMode:   constant.RelayModeAudioSpeech,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "tts-1-hd"},
	}
	request := dto.AudioRequest{
		Model:          "tts-1-hd",
		Input:          "hello",
		Voice:          "alloy",
		ResponseFormat: "mp3",
		Metadata:       metadata,
	}

	withMiniMaxSettings(t, cfg, func() {
		c := newConvertAudioContext()
		a := &Adaptor{}
		reader, err := a.ConvertAudioRequest(c, info, request)
		if err != nil {
			t.Fatalf("ConvertAudioRequest error: %v", err)
		}
		got := decodeTTSBody(t, reader)
		if got.Model != "speech-02-hd" {
			t.Errorf("Model = %q, want speech-02-hd (admin redirect must win over metadata)", got.Model)
		}
		if v, ok := c.Get("minimax_voice_id"); !ok || v != "alloy" {
			t.Errorf("minimax_voice_id = %v (ok=%v), want alloy", v, ok)
		}
	})
}

// TestConvertAudioRequest_DisabledKeepsMetadataValues 验证 cfg.Enabled=false 时保留 metadata。
func TestConvertAudioRequest_DisabledKeepsMetadataValues(t *testing.T) {
	cfg := model.MiniMaxSettings{
		Enabled:       false,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
	}
	metadata := []byte(`{"model":"user-model","voice_setting":{"voice_id":"user-voice"}}`)

	info := &relaycommon.RelayInfo{
		RelayMode:   constant.RelayModeAudioSpeech,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "tts-1-hd"},
	}
	request := dto.AudioRequest{
		Model:          "tts-1-hd",
		Input:          "hello",
		Voice:          "alloy",
		ResponseFormat: "mp3",
		Metadata:       metadata,
	}

	withMiniMaxSettings(t, cfg, func() {
		c := newConvertAudioContext()
		a := &Adaptor{}
		reader, err := a.ConvertAudioRequest(c, info, request)
		if err != nil {
			t.Fatalf("ConvertAudioRequest error: %v", err)
		}
		got := decodeTTSBody(t, reader)
		if got.Model != "user-model" {
			t.Errorf("Model = %q, want user-model (disabled: metadata should win)", got.Model)
		}
	})
}

// TestConvertAudioRequest_UserVoiceLogged 校验音色日志记录用户原始 voice。
func TestConvertAudioRequest_UserVoiceLogged(t *testing.T) {
	cases := []struct {
		name      string
		enabled   bool
		wantModel string
	}{
		{"enabled_applies_model_redirect", true, "speech-02-turbo"},
		{"disabled_keeps_user_values", false, "tts-1-turbo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := model.MiniMaxSettings{
				Enabled:       tc.enabled,
				ModelRedirect: map[string]string{"tts-1-turbo": "speech-02-turbo"},
			}
			info := &relaycommon.RelayInfo{
				RelayMode:   constant.RelayModeAudioSpeech,
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "tts-1-turbo"},
			}
			request := dto.AudioRequest{
				Model:          "tts-1-turbo",
				Input:          "hello",
				Voice:          "voice_2",
				ResponseFormat: "mp3",
			}

			withMiniMaxSettings(t, cfg, func() {
				c := newConvertAudioContext()
				a := &Adaptor{}
				reader, err := a.ConvertAudioRequest(c, info, request)
				if err != nil {
					t.Fatalf("ConvertAudioRequest error: %v", err)
				}
				got := decodeTTSBody(t, reader)
				if got.Model != tc.wantModel {
					t.Errorf("Model = %q, want %q", got.Model, tc.wantModel)
				}
				if v, ok := c.Get("minimax_voice_id"); !ok || v != "voice_2" {
					t.Errorf("minimax_voice_id = %v (ok=%v), want voice_2", v, ok)
				}
			})
		})
	}
}

// TestConvertAudioRequest_MetadataNonPolicyFieldsPreserved 验证非策略字段（audio_setting.sample_rate）
// 在 cfg.Enabled=true 时仍由 metadata 提供，管理员策略只覆盖 model/emotion/text/format。
func TestConvertAudioRequest_MetadataNonPolicyFieldsPreserved(t *testing.T) {
	cfg := model.MiniMaxSettings{
		Enabled:       true,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
	}
	metadata := []byte(`{"model":"evil","audio_setting":{"sample_rate":32000,"format":"wav"},"language_boost":"zh"}`)

	info := &relaycommon.RelayInfo{
		RelayMode:   constant.RelayModeAudioSpeech,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "tts-1-hd"},
	}
	request := dto.AudioRequest{
		Model:          "tts-1-hd",
		Input:          "hello",
		Voice:          "alloy",
		ResponseFormat: "mp3",
		Metadata:       metadata,
	}

	withMiniMaxSettings(t, cfg, func() {
		c := newConvertAudioContext()
		a := &Adaptor{}
		reader, err := a.ConvertAudioRequest(c, info, request)
		if err != nil {
			t.Fatalf("ConvertAudioRequest error: %v", err)
		}
		got := decodeTTSBody(t, reader)
		if got.Model != "speech-02-hd" {
			t.Errorf("Model = %q, want speech-02-hd", got.Model)
		}
		if got.AudioSetting.Format != "mp3" {
			t.Errorf("AudioSetting.Format = %q, want mp3 (admin forced)", got.AudioSetting.Format)
		}
		if got.AudioSetting.SampleRate != 32000 {
			t.Errorf("AudioSetting.SampleRate = %d, want 32000 (from metadata)", got.AudioSetting.SampleRate)
		}
		if got.LanguageBoost != "zh" {
			t.Errorf("LanguageBoost = %q, want zh (from metadata)", got.LanguageBoost)
		}
	})
}

func TestConvertAudioRequest_NullAudioSettingMetadataDoesNotBypassPolicy(t *testing.T) {
	cfg := model.MiniMaxSettings{
		Enabled:       true,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
	}
	metadata := []byte(`{"audio_setting":null}`)

	info := &relaycommon.RelayInfo{
		RelayMode:   constant.RelayModeAudioSpeech,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "tts-1-hd"},
	}
	request := dto.AudioRequest{
		Model:          "tts-1-hd",
		Input:          "hello",
		Voice:          "alloy",
		ResponseFormat: "mp3",
		Metadata:       metadata,
	}

	withMiniMaxSettings(t, cfg, func() {
		c := newConvertAudioContext()
		a := &Adaptor{}
		reader, err := a.ConvertAudioRequest(c, info, request)
		if err != nil {
			t.Fatalf("ConvertAudioRequest error: %v", err)
		}
		got := decodeTTSBody(t, reader)
		if got.AudioSetting == nil || got.AudioSetting.Format != "mp3" {
			t.Fatalf("AudioSetting.Format = %v, want mp3 (admin forced even when metadata null)", got.AudioSetting)
		}
	})
}

// TestConvertAudioRequest_NormalizesFormats 验证 OpenAI response_format 映射到
// MiniMax audio_setting.format，且 output_format 固定为 hex。
func TestConvertAudioRequest_NormalizesFormats(t *testing.T) {
	cases := []struct {
		name            string
		format          string
		wantAudioFormat string
	}{
		{"default_mp3", "", "mp3"},
		{"mp3", "mp3", "mp3"},
		{"opus", "opus", "opus"},
		{"flac", "flac", "flac"},
		{"wav", "wav", "wav"},
		{"pcm", "pcm", "pcm"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withMiniMaxSettings(t, model.MiniMaxSettings{Enabled: false}, func() {
				c := newConvertAudioContext()
				a := &Adaptor{}
				info := &relaycommon.RelayInfo{
					RelayMode:   constant.RelayModeAudioSpeech,
					ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "tts-1-hd"},
				}
				request := dto.AudioRequest{
					Input:          "hello",
					ResponseFormat: tc.format,
				}
				reader, err := a.ConvertAudioRequest(c, info, request)
				if err != nil {
					t.Fatalf("ConvertAudioRequest error: %v", err)
				}
				got := decodeTTSBody(t, reader)
				if got.OutputFormat != "hex" {
					t.Errorf("OutputFormat = %q, want hex", got.OutputFormat)
				}
				if got.AudioSetting == nil || got.AudioSetting.Format != tc.wantAudioFormat {
					t.Fatalf("AudioSetting.Format = %v, want %s", got.AudioSetting, tc.wantAudioFormat)
				}
				v, ok := c.Get("response_format")
				if !ok {
					t.Fatalf("response_format not set")
				}
				if v != "hex" {
					t.Errorf("response_format = %v, want hex", v)
				}
				if v, ok := c.Get("minimax_audio_format"); !ok || v != tc.wantAudioFormat {
					t.Errorf("minimax_audio_format = %v (ok=%v), want %s", v, ok, tc.wantAudioFormat)
				}
			})
		})
	}
}

func TestConvertAudioRequest_AACReturnsSupportedFormatList(t *testing.T) {
	withMiniMaxSettings(t, model.MiniMaxSettings{Enabled: false}, func() {
		c := newConvertAudioContext()
		a := &Adaptor{}
		info := &relaycommon.RelayInfo{
			RelayMode:   constant.RelayModeAudioSpeech,
			ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "tts-1-hd"},
		}
		request := dto.AudioRequest{
			Input:          "hello",
			ResponseFormat: "aac",
		}

		_, err := a.ConvertAudioRequest(c, info, request)
		if err == nil {
			t.Fatal("ConvertAudioRequest error = nil, want unsupported aac error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "aac") {
			t.Fatalf("error = %q, want mention aac", msg)
		}
		if !strings.Contains(msg, miniMaxTTSSupportedAudioFormats) {
			t.Fatalf("error = %q, want supported format list %q", msg, miniMaxTTSSupportedAudioFormats)
		}
	})
}

func TestGetRequestURLForImageGeneration(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RelayMode: constant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.minimaxi.com/v1",
		},
	}
	got, err := GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error: %v", err)
	}
	if got != "https://api.minimaxi.com/v1/image_generation" {
		t.Fatalf("GetRequestURL = %q, want MiniMax image endpoint", got)
	}
}

func TestConvertImageRequestBuildsMiniMaxPayload(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:       constant.RelayModeImagesGenerations,
		OriginModelName: "image-01",
	}
	request := dto.ImageRequest{
		Model:          "image-01",
		Prompt:         "a red fox in snowfall",
		Size:           "1536x1024",
		ResponseFormat: "b64_json",
		N:              2,
	}
	got, err := adaptor.ConvertImageRequest(newConvertAudioContext(), info, request)
	if err != nil {
		t.Fatalf("ConvertImageRequest error: %v", err)
	}
	payload, ok := got.(MiniMaxImageRequest)
	if !ok {
		t.Fatalf("converted request type = %T, want MiniMaxImageRequest", got)
	}
	if payload.Model != "image-01" {
		t.Fatalf("Model = %q, want image-01", payload.Model)
	}
	if payload.N != 2 {
		t.Fatalf("N = %d, want 2", payload.N)
	}
}

func TestDoResponseForImageGenerationReturnsOpenAIImageResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := &relaycommon.RelayInfo{
		RelayMode: constant.RelayModeImagesGenerations,
		StartTime: time.Unix(1700000000, 0),
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(`{
			"data": {
				"image_urls": ["https://example.com/minimax.png"]
			},
			"base_resp": {
				"status_code": 0
			}
		}`)),
	}

	usage, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
	if apiErr != nil {
		t.Fatalf("DoResponse error: %v", apiErr)
	}
	imageUsage, ok := usage.(*dto.Usage)
	if !ok {
		t.Fatalf("usage type = %T, want *dto.Usage", usage)
	}
	if imageUsage.TotalTokens != 0 {
		t.Fatalf("TotalTokens = %d, want 0 so ImageHelper can apply request N", imageUsage.TotalTokens)
	}
	if strings.Contains(recorder.Body.String(), "image_urls") {
		t.Fatalf("response body should not expose raw image_urls payload")
	}
}
