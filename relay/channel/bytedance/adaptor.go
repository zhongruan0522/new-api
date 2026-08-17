package bytedance

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	channelconstant "github.com/NookMux/NookMux/constant"
	"github.com/NookMux/NookMux/dto"
	"github.com/NookMux/NookMux/relay/channel"
	"github.com/NookMux/NookMux/relay/channel/claude"
	"github.com/NookMux/NookMux/relay/channel/openai"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	relayconstant "github.com/NookMux/NookMux/relay/constant"
	"github.com/NookMux/NookMux/types"
	"github.com/gin-gonic/gin"
)

// Adaptor 对接字节跳动（火山方舟 Ark）。
// Ark 同时提供 OpenAI 兼容入口与 Anthropic 兼容入口：
//   - OpenAI：基础 URL 形如 https://ark.cn-beijing.volces.com/api/v3，
//     chat/responses/embeddings 路径直接挂在 base URL 之下。
//   - Claude：入口为 https://ark.cn-beijing.volces.com/api/compatible/v1/messages，
//     由 base URL 中的 /api/v3 替换为 /api/compatible 推导得到。
type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	// Ark 已原生支持 Anthropic Messages API，直接透传 Claude 请求。
	return req, nil
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

// claudeBaseURL 将 Ark 的 OpenAI 兼容 base URL 转为 Anthropic 兼容 base URL。
// 默认 https://ark.cn-beijing.volces.com/api/v3 -> https://ark.cn-beijing.volces.com/api/compatible。
// 若 base URL 不含 /api/v3 后缀，则原样返回，由用户自行保证可拼接 /v1/messages。
func claudeBaseURL(baseURL string) string {
	if baseURL == "" {
		return channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeByteDance]
	}
	if strings.HasSuffix(baseURL, "/api/v3") {
		return strings.TrimSuffix(baseURL, "/api/v3") + "/api/compatible"
	}
	if strings.HasSuffix(baseURL, "/v3") {
		return strings.TrimSuffix(baseURL, "/v3") + "/compatible"
	}
	return baseURL
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimRight(info.ChannelBaseUrl, "/")
	if baseURL == "" {
		baseURL = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeByteDance]
	}

	switch info.RelayFormat {
	case types.RelayFormatClaude:
		return fmt.Sprintf("%s/v1/messages", claudeBaseURL(baseURL)), nil
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
	if info.RelayFormat == types.RelayFormatClaude {
		// Anthropic 兼容接口使用 x-api-key 与 anthropic-version 头。
		req.Del("Authorization")
		if info.ApiKey != "" {
			req.Set("x-api-key", info.ApiKey)
		}
		anthropicVersion := c.Request.Header.Get("anthropic-version")
		if anthropicVersion == "" {
			anthropicVersion = "2023-06-01"
		}
		req.Set("anthropic-version", anthropicVersion)
		claude.CommonClaudeHeadersOperation(c, req, info)
		return nil
	}
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

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NookMuxError) {
	if info.RelayFormat == types.RelayFormatClaude {
		delegate := &claude.Adaptor{}
		delegate.Init(info)
		return delegate.DoResponse(c, resp, info)
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
