package minimax

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/relay/constant"
	"github.com/zhongruan0522/new-api/setting/model_setting"
)

// withMiniMaxSettings 临时替换全局 MiniMaxSettings，测试结束自动恢复。
// GetMiniMaxSettings 返回包级全局指针，需通过快照-恢复来隔离用例。
func withMiniMaxSettings(t *testing.T, cfg model_setting.MiniMaxSettings, fn func()) {
	t.Helper()
	prev := *model_setting.GetMiniMaxSettings()
	defer func() {
		*model_setting.GetMiniMaxSettings() = prev
	}()
	*model_setting.GetMiniMaxSettings() = cfg
	fn()
}

// newConvertAudioContext 构造调用 ConvertAudioRequest 所需的最小 gin.Context。
func newConvertAudioContext() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/audio/speech", nil)
	return c
}

// decodeTTSBody 将 ConvertAudioRequest 返回的 io.Reader 反序列化为 MiniMaxTTSRequest。
func decodeTTSBody(t *testing.T, r io.Reader) MiniMaxTTSRequest {
	t.Helper()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var req MiniMaxTTSRequest
	if err := common.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	return req
}

// TestConvertAudioRequest_MetadataCannotOverrideModelAndVoice 验证 issue #107 修复：
// 当 metadata 包含 model / voice_setting.voice_id 且 cfg.Enabled=true 配置了重定向时，
// 管理员强制策略必须最终覆盖 metadata 中的值。
//
// 修复前：metadata 在管理员策略之后反序列化到整个 minimaxRequest，
// 用户可通过 metadata 覆盖 model、voice_setting.voice_id 等策略字段。
func TestConvertAudioRequest_MetadataCannotOverrideModelAndVoice(t *testing.T) {
	cfg := model_setting.MiniMaxSettings{
		Enabled:       true,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
		VoiceRedirect: map[string]string{"alloy": "female-shaonv"},
	}

	// 用户 metadata 试图覆盖管理员策略字段
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

		// 管理员映射结果必须胜出
		if got.Model != "speech-02-hd" {
			t.Errorf("Model = %q, want speech-02-hd (admin redirect must win over metadata)", got.Model)
		}
		if got.VoiceSetting.VoiceID != "female-shaonv" {
			t.Errorf("VoiceSetting.VoiceID = %q, want female-shaonv (admin redirect must win over metadata)", got.VoiceSetting.VoiceID)
		}

		// 音色日志记录用户请求中的 OpenAI voice，不暴露重定向后的真实上游 voice_id。
		if v, ok := c.Get("minimax_voice_id"); !ok || v != "alloy" {
			t.Errorf("minimax_voice_id = %v (ok=%v), want alloy", v, ok)
		}
	})
}

// TestConvertAudioRequest_DisabledKeepsMetadataValues 验证 cfg.Enabled=false 时不做强制覆盖：
// 用户 metadata 的值应被保留（行为与修复前一致）。
func TestConvertAudioRequest_DisabledKeepsMetadataValues(t *testing.T) {
	cfg := model_setting.MiniMaxSettings{
		Enabled:       false,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
		VoiceRedirect: map[string]string{"alloy": "female-shaonv"},
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

		// 关闭增强时 metadata 中的值应保留
		if got.Model != "user-model" {
			t.Errorf("Model = %q, want user-model (disabled: metadata should win)", got.Model)
		}
		if got.VoiceSetting.VoiceID != "user-voice" {
			t.Errorf("VoiceSetting.VoiceID = %q, want user-voice (disabled: metadata should win)", got.VoiceSetting.VoiceID)
		}

		// 关闭增强时仍记录用户请求 voice，便于从用户侧排查 TTS 请求。
		if v, ok := c.Get("minimax_voice_id"); !ok || v != "alloy" {
			t.Errorf("minimax_voice_id = %v (ok=%v), want alloy", v, ok)
		}
	})
}

func TestConvertAudioRequest_UserVoiceLogDoesNotExposeRedirectedVoice(t *testing.T) {
	cases := []struct {
		name              string
		enabled           bool
		wantModel         string
		wantUpstreamVoice string
	}{
		{
			name:              "enabled_applies_redirect_but_logs_user_voice",
			enabled:           true,
			wantModel:         "speech-02-turbo",
			wantUpstreamVoice: "Chinese (Mandarin)_Warm_Girl",
		},
		{
			name:              "disabled_keeps_user_values_and_logs_user_voice",
			enabled:           false,
			wantModel:         "tts-1-turbo",
			wantUpstreamVoice: "voice_2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := model_setting.MiniMaxSettings{
				Enabled:       tc.enabled,
				ModelRedirect: map[string]string{"tts-1-turbo": "speech-02-turbo"},
				VoiceRedirect: map[string]string{"voice_2": "Chinese (Mandarin)_Warm_Girl"},
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
				if got.VoiceSetting.VoiceID != tc.wantUpstreamVoice {
					t.Errorf("VoiceSetting.VoiceID = %q, want %q", got.VoiceSetting.VoiceID, tc.wantUpstreamVoice)
				}
				if v, ok := c.Get("minimax_voice_id"); !ok || v != "voice_2" {
					t.Errorf("minimax_voice_id = %v (ok=%v), want voice_2", v, ok)
				}
			})
		})
	}
}

// TestConvertAudioRequest_MetadataNonPolicyFieldsPreserved 验证非策略字段（如 audio_setting.sample_rate）
// 在 cfg.Enabled=true 时仍由 metadata 提供，管理员策略只覆盖 model/voice/emotion/text/format。
func TestConvertAudioRequest_MetadataNonPolicyFieldsPreserved(t *testing.T) {
	cfg := model_setting.MiniMaxSettings{
		Enabled:       true,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
		VoiceRedirect: map[string]string{"alloy": "female-shaonv"},
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

		// 策略字段由管理员覆盖
		if got.Model != "speech-02-hd" {
			t.Errorf("Model = %q, want speech-02-hd", got.Model)
		}
		// format 是策略字段，管理员用 request.ResponseFormat 覆盖
		if got.AudioSetting.Format != "mp3" {
			t.Errorf("AudioSetting.Format = %q, want mp3 (admin forced)", got.AudioSetting.Format)
		}
		if got.OutputFormat != "mp3" {
			t.Errorf("OutputFormat = %q, want mp3 (admin forced)", got.OutputFormat)
		}
		// 非策略字段由 metadata 提供
		if got.AudioSetting.SampleRate != 32000 {
			t.Errorf("AudioSetting.SampleRate = %d, want 32000 (from metadata)", got.AudioSetting.SampleRate)
		}
		if got.LanguageBoost != "zh" {
			t.Errorf("LanguageBoost = %q, want zh (from metadata)", got.LanguageBoost)
		}
	})
}

func TestConvertAudioRequest_NullAudioSettingMetadataDoesNotBypassPolicy(t *testing.T) {
	cfg := model_setting.MiniMaxSettings{
		Enabled:       true,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
		VoiceRedirect: map[string]string{"alloy": "female-shaonv"},
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
		if got.AudioSetting == nil {
			t.Fatal("AudioSetting is nil, want initialized setting")
		}
		if got.AudioSetting.Format != "mp3" {
			t.Errorf("AudioSetting.Format = %q, want mp3", got.AudioSetting.Format)
		}
		if got.Model != "speech-02-hd" {
			t.Errorf("Model = %q, want speech-02-hd", got.Model)
		}
		if got.VoiceSetting.VoiceID != "female-shaonv" {
			t.Errorf("VoiceSetting.VoiceID = %q, want female-shaonv", got.VoiceSetting.VoiceID)
		}
	})
}

// TestConvertAudioRequest_EmptyMetadataUsesAdminRedirect 验证无 metadata 时仍正常应用管理员重定向。
func TestConvertAudioRequest_EmptyMetadataUsesAdminRedirect(t *testing.T) {
	cfg := model_setting.MiniMaxSettings{
		Enabled:       true,
		ModelRedirect: map[string]string{"tts-1-hd": "speech-02-hd"},
		VoiceRedirect: map[string]string{"alloy": "female-shaonv"},
	}

	info := &relaycommon.RelayInfo{
		RelayMode:   constant.RelayModeAudioSpeech,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "tts-1-hd"},
	}
	request := dto.AudioRequest{
		Model:          "tts-1-hd",
		Input:          "hello",
		Voice:          "alloy",
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

		if got.Model != "speech-02-hd" {
			t.Errorf("Model = %q, want speech-02-hd", got.Model)
		}
		if got.VoiceSetting.VoiceID != "female-shaonv" {
			t.Errorf("VoiceSetting.VoiceID = %q, want female-shaonv", got.VoiceSetting.VoiceID)
		}
	})
}

// TestConvertAudioRequest_ResponseFormatNormalization 验证 c.Set("response_format", ...) 规范化逻辑未回归。
func TestConvertAudioRequest_ResponseFormatNormalization(t *testing.T) {
	cases := []struct {
		name    string
		format  string
		wantSet string
	}{
		{"hex_passthrough", "hex", "hex"},
		{"non_hex_normalized", "mp3", "url"},
		{"empty_normalized", "", "url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withMiniMaxSettings(t, model_setting.MiniMaxSettings{Enabled: false}, func() {
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
				_, err := a.ConvertAudioRequest(c, info, request)
				if err != nil {
					t.Fatalf("ConvertAudioRequest error: %v", err)
				}
				v, ok := c.Get("response_format")
				if !ok {
					t.Fatalf("response_format not set")
				}
				if v != tc.wantSet {
					t.Errorf("response_format = %v, want %v", v, tc.wantSet)
				}
			})
		})
	}
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
	if payload.Prompt != request.Prompt {
		t.Fatalf("Prompt = %q, want %q", payload.Prompt, request.Prompt)
	}
	if payload.N != 2 {
		t.Fatalf("N = %d, want 2", payload.N)
	}
	if payload.AspectRatio != "3:2" {
		t.Fatalf("AspectRatio = %q, want 3:2", payload.AspectRatio)
	}
	if payload.ResponseFormat != "base64" {
		t.Fatalf("ResponseFormat = %q, want base64", payload.ResponseFormat)
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

	var body dto.ImageResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v, body=%s", err, recorder.Body.String())
	}
	if len(body.Data) != 1 || body.Data[0].Url != "https://example.com/minimax.png" {
		t.Fatalf("response data = %+v, want OpenAI image URL", body.Data)
	}
	if strings.Contains(recorder.Body.String(), "image_urls") {
		t.Fatalf("response body = %s, should not expose raw MiniMax image_urls payload", recorder.Body.String())
	}
}
