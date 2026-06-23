package minimax

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	"github.com/zhongruan0522/new-api/relay/channel"
	"github.com/zhongruan0522/new-api/relay/channel/claude"
	"github.com/zhongruan0522/new-api/relay/channel/openai"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/relay/constant"
	"github.com/zhongruan0522/new-api/setting/model_setting"
	"github.com/zhongruan0522/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	adaptor := claude.Adaptor{}
	return adaptor.ConvertClaudeRequest(c, info, req)
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	if info.RelayMode != constant.RelayModeAudioSpeech {
		return nil, errors.New("unsupported audio relay mode")
	}

	// <editor-fold desc="debug H2/H3 minimax tts adapter inputs">
	relaycommon.DebugMiniMaxTTS("relay/channel/minimax/adaptor.go:37", "convert-audio-request-start", map[string]any{
		"hypothesisId":      "H2-H3",
		"relayMode":         info.RelayMode,
		"channelType":       info.ChannelType,
		"apiType":           info.ApiType,
		"requestModel":      request.Model,
		"upstreamModelName": info.UpstreamModelName,
		"requestVoice":      request.Voice,
		"responseFormat":    request.ResponseFormat,
		"metadataLen":       len(request.Metadata),
	})
	// </editor-fold>

	outputFormat := request.ResponseFormat

	// 1) 先用用户请求原始值构造基础请求（不应用管理员重定向）。
	//    使用 UpstreamModelName（渠道级 model_mapping 已由 ModelMappedHelper 处理），
	//    作为 TTS 专用重定向的输入，支持 alias -> tts-1-hd -> speech-02-hd 链式映射。
	minimaxRequest := MiniMaxTTSRequest{
		Model: info.UpstreamModelName,
		Text:  request.Input,
		VoiceSetting: VoiceSetting{
			VoiceID: request.Voice,
			Speed:   request.Speed,
		},
		AudioSetting: &AudioSetting{
			Format: outputFormat,
		},
		OutputFormat: outputFormat,
	}

	// 2) 合并用户 metadata（扩展字段的厂商自定义值）。
	//    必须在管理员强制策略之前合并，使管理员映射具有最高优先级，
	//    防止用户通过 metadata 覆盖 model/voice_id 等策略字段 (issue #107)。
	if len(request.Metadata) > 0 {
		if err := common.Unmarshal(request.Metadata, &minimaxRequest); err != nil {
			return nil, fmt.Errorf("error unmarshalling metadata to minimax request: %w", err)
		}
	}

	// <editor-fold desc="debug H2/H3 minimax tts after metadata merge">
	relaycommon.DebugMiniMaxTTS("relay/channel/minimax/adaptor.go:63", "after-metadata-merge", map[string]any{
		"hypothesisId":       "H2-H3",
		"model":              minimaxRequest.Model,
		"voiceId":            minimaxRequest.VoiceSetting.VoiceID,
		"emotion":            minimaxRequest.VoiceSetting.Emotion,
		"text":               minimaxRequest.Text,
		"audioFormat":        minimaxRequest.AudioSetting.Format,
		"outputFormat":       minimaxRequest.OutputFormat,
		"cfgEnabled":         model_setting.GetMiniMaxSettings().Enabled,
		"redirectCountModel": len(model_setting.GetMiniMaxSettings().ModelRedirect),
		"redirectCountVoice": len(model_setting.GetMiniMaxSettings().VoiceRedirect),
	})
	// </editor-fold>

	// 3) 应用管理员强制策略：在 metadata 合并之后，用映射结果覆盖策略字段。
	//    仅当 cfg.Enabled 时生效；关闭时保留用户原始值（含 metadata）。
	cfg := model_setting.GetMiniMaxSettings()
	if cfg.Enabled {
		minimaxRequest.Model = applyModelRedirect(info.UpstreamModelName, cfg)
		minimaxRequest.VoiceSetting.VoiceID = applyVoiceRedirect(request.Voice, cfg)
		emotion, inputText := extractEmotion(request.Input, cfg.EmotionPattern, cfg.EmotionRedirect)
		inputText = replaceToneWords(inputText, cfg.ToneWordPattern, cfg.ToneWordRedirect)
		minimaxRequest.VoiceSetting.Emotion = emotion
		minimaxRequest.Text = inputText
		minimaxRequest.AudioSetting.Format = outputFormat
		minimaxRequest.OutputFormat = outputFormat
	}

	normalizedFormat := outputFormat
	if normalizedFormat != "hex" {
		normalizedFormat = "url"
	}
	c.Set("response_format", normalizedFormat)

	// <editor-fold desc="debug H1/H2/H3 minimax tts final outbound request">
	relaycommon.DebugMiniMaxTTS("relay/channel/minimax/adaptor.go:81", "final-outbound-request", map[string]any{
		"hypothesisId":   "H1-H2-H3",
		"cfgEnabled":     cfg.Enabled,
		"finalModel":     minimaxRequest.Model,
		"finalVoiceId":   minimaxRequest.VoiceSetting.VoiceID,
		"finalEmotion":   minimaxRequest.VoiceSetting.Emotion,
		"finalText":      minimaxRequest.Text,
		"finalAudioFmt":  minimaxRequest.AudioSetting.Format,
		"finalOutFormat": minimaxRequest.OutputFormat,
		"responseFormat": normalizedFormat,
	})
	// </editor-fold>

	jsonData, err := common.Marshal(minimaxRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshalling minimax request: %w", err)
	}

	// 音色日志：记录 MiniMax TTS 实际使用的 voice_id（管理员策略最终覆盖后的值）。
	// 由上层 audio_handler.go 通过 extraContent 传给 PostAudioConsumeQuota。
	if cfg.Enabled {
		c.Set("minimax_voice_id", minimaxRequest.VoiceSetting.VoiceID)
	}

	return bytes.NewReader(jsonData), nil
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return GetRequestURL(info)
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		adaptor := claude.Adaptor{}
		return adaptor.DoResponse(c, resp, info)
	default:
	}

	if info.RelayMode == constant.RelayModeAudioSpeech {
		return handleTTSResponse(c, resp, info)
	}

	adaptor := openai.Adaptor{}
	return adaptor.DoResponse(c, resp, info)
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
