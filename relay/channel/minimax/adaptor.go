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
			return nil, fmt.Errorf("error unmarshalling metadata to TTS request: %w", err)
		}
	}
	clientVoice := minimaxRequest.VoiceSetting.VoiceID
	if clientVoice == "" {
		clientVoice = request.Voice
	}
	if err := model_setting.ValidateMiniMaxVoiceAllowed(clientVoice); err != nil {
		return nil, err
	}

	// 3) 应用管理员强制策略：在 metadata 合并之后，用映射结果覆盖策略字段。
	//    仅当 cfg.Enabled 时生效；关闭时保留用户原始值（含 metadata）。
	policy := model_setting.ApplyMiniMaxTTSPolicy(info.UpstreamModelName, request.Voice, request.Input, outputFormat)
	if policy.Enabled {
		minimaxRequest.Model = policy.Model
		minimaxRequest.VoiceSetting.VoiceID = policy.Voice
		minimaxRequest.VoiceSetting.Emotion = policy.Emotion
		minimaxRequest.Text = policy.Text
		if minimaxRequest.AudioSetting == nil {
			minimaxRequest.AudioSetting = &AudioSetting{}
		}
		minimaxRequest.AudioSetting.Format = outputFormat
		minimaxRequest.OutputFormat = outputFormat
	}

	normalizedFormat := outputFormat
	if normalizedFormat != "hex" {
		normalizedFormat = "url"
	}
	c.Set("response_format", normalizedFormat)

	jsonData, err := common.Marshal(minimaxRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshalling TTS request: %w", err)
	}

	// 音色日志：只记录用户请求中的 OpenAI TTS voice，避免暴露重定向后的真实上游 voice_id。
	// 由上层 audio_handler.go 通过 extraContent 传给 PostAudioConsumeQuota。
	if request.Voice != "" {
		c.Set("minimax_voice_id", request.Voice)
	}

	return bytes.NewReader(jsonData), nil
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if info.RelayMode != constant.RelayModeImagesGenerations {
		return nil, fmt.Errorf("unsupported image relay mode: %d", info.RelayMode)
	}
	return oaiImage2MiniMaxImageRequest(request), nil
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
	if info.RelayMode == constant.RelayModeImagesGenerations {
		return miniMaxImageHandler(c, resp, info)
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
