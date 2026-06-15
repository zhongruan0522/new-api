package minimax

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

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

	voiceID := request.Voice
	speed := request.Speed
	outputFormat := request.ResponseFormat
	// 使用 UpstreamModelName（渠道级 model_mapping 已由 ModelMappedHelper 处理），
	// 作为 TTS 专用重定向的输入，支持 alias -> tts-1-hd -> speech-02-hd 链式映射。
	modelName := info.UpstreamModelName
	inputText := request.Input
	emotion := ""

	// MiniMax TTS 增强配置（仅 MiniMax 渠道生效，因为只有 MiniMax adaptor 会走到这里）
	cfg := model_setting.GetMiniMaxSettings()
	if cfg.Enabled {
		modelName = applyModelRedirect(info.UpstreamModelName, cfg)
		voiceID = applyVoiceRedirect(request.Voice, cfg)
		emotion, inputText = extractEmotion(inputText, cfg.EmotionPattern, cfg.EmotionRedirect)
		inputText = replaceToneWords(inputText, cfg.ToneWordPattern, cfg.ToneWordRedirect)
	}

	minimaxRequest := MiniMaxTTSRequest{
		Model: modelName,
		Text:  inputText,
		VoiceSetting: VoiceSetting{
			VoiceID: voiceID,
			Speed:   speed,
			Emotion: emotion,
		},
		AudioSetting: &AudioSetting{
			Format: outputFormat,
		},
		OutputFormat: outputFormat,
	}

	// 同步扩展字段的厂商自定义metadata
	if len(request.Metadata) > 0 {
		if err := json.Unmarshal(request.Metadata, &minimaxRequest); err != nil {
			return nil, fmt.Errorf("error unmarshalling metadata to minimax request: %w", err)
		}
	}

	jsonData, err := json.Marshal(minimaxRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshalling minimax request: %w", err)
	}
	if outputFormat != "hex" {
		outputFormat = "url"
	}

	c.Set("response_format", outputFormat)

	// 音色日志：记录 MiniMax TTS 实际使用的 voice_id。
	// 使用 metadata 合并后的最终 voice_id（用户通过 metadata 覆盖时也能反映真实值）。
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
