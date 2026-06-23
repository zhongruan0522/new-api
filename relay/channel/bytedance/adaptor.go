package bytedance

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	channelconstant "github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/dto"
	"github.com/zhongruan0522/new-api/relay/channel"
	"github.com/zhongruan0522/new-api/relay/channel/openai"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	relayconstant "github.com/zhongruan0522/new-api/relay/constant"
	"github.com/zhongruan0522/new-api/types"
)

// Adaptor 对接字节跳动（火山方舟 Ark）OpenAI 兼容接口。
// Ark 基础 URL 形如 https://ark.cn-beijing.volces.com/api/v3，
// 上游 chat / responses / embeddings 路径直接挂在 base URL 之下。
type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	adaptor := openai.Adaptor{}
	return adaptor.ConvertClaudeRequest(c, info, req)
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("not supported")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	adaptor := openai.Adaptor{}
	return adaptor.ConvertImageRequest(c, info, request)
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimRight(info.ChannelBaseUrl, "/")
	if baseURL == "" {
		baseURL = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeByteDance]
	}

	switch info.RelayFormat {
	case types.RelayFormatClaude:
		// Ark 暂未提供 Anthropic 原生协议入口
		return "", errors.New("claude format is not supported by ByteDance channel")
	default:
		switch info.RelayMode {
		case relayconstant.RelayModeResponses, relayconstant.RelayModeResponsesCompact:
			return fmt.Sprintf("%s/responses", baseURL), nil
		case relayconstant.RelayModeEmbeddings:
			return fmt.Sprintf("%s/embeddings", baseURL), nil
		case relayconstant.RelayModeChatCompletions:
			return fmt.Sprintf("%s/chat/completions", baseURL), nil
		case relayconstant.RelayModeCompletions:
			return fmt.Sprintf("%s/completions", baseURL), nil
		case relayconstant.RelayModeRerank:
			return fmt.Sprintf("%s/rerank", baseURL), nil
		default:
			return fmt.Sprintf("%s/chat/completions", baseURL), nil
		}
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", fmt.Sprintf("Bearer %s", info.ApiKey))
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	adaptor := openai.Adaptor{}
	return adaptor.DoResponse(c, resp, info)
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
