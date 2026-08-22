package ollama

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	channelconstant "github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/relay/channel"
	"github.com/NookMux/NookMux/internal/relay/channel/claude"
	"github.com/NookMux/NookMux/internal/relay/channel/openai"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"

	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *shared.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *shared.ClaudeRequest) (any, error) {
	// Ollama 已经原生支持 Anthropic Messages API，这里直接透传 Claude 请求。
	return request, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request shared.AudioRequest) (io.Reader, error) {
	// 音频请求体仍沿用 OpenAI 兼容格式，复用现有构造逻辑即可。
	delegate := &openai.Adaptor{}
	return delegate.ConvertAudioRequest(c, info, request)
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request shared.ImageRequest) (any, error) {
	// 图片请求体仍沿用 OpenAI 兼容格式，复用现有构造逻辑即可。
	delegate := &openai.Adaptor{}
	return delegate.ConvertImageRequest(c, info, request)
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	// 复用 OpenAI adaptor 的通用初始化。
	delegate := &openai.Adaptor{}
	delegate.Init(info)
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimSpace(info.ChannelBaseUrl)
	if baseURL == "" {
		baseURL = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeOllama]
	}
	specialPlan, hasSpecialPlan := channelconstant.ChannelSpecialBases[baseURL]
	if hasSpecialPlan {
		switch info.RelayFormat {
		case relayconstant.RelayFormatClaude:
			if specialPlan.ClaudeBaseURL != "" {
				return fmt.Sprintf("%s/v1/messages", specialPlan.ClaudeBaseURL), nil
			}
		default:
			if specialPlan.OpenAIBaseURL != "" {
				return fmt.Sprintf("%s%s", specialPlan.OpenAIBaseURL, info.RequestURLPath), nil
			}
		}
	}
	// Ollama 现已支持标准兼容端点，直接转发到客户端请求的原始规范路径。
	return relaycommon.GetFullRequestURL(baseURL, info.RequestURLPath, info.ChannelType), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	if info.RelayFormat == relayconstant.RelayFormatClaude {
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
	if info.ApiKey != "" {
		req.Set("Authorization", "Bearer "+info.ApiKey)
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *shared.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	// OpenAI 兼容请求已经可以被 Ollama 原生识别，直接透传。
	return request, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request shared.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request shared.EmbeddingRequest) (any, error) {
	// Embeddings 也已支持 OpenAI 标准请求体，直接透传。
	return request, nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request shared.OpenAIResponsesRequest) (any, error) {
	// Responses API 直接走 Ollama 的 /v1/responses，不再降级到旧 /api/chat。
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *shared.NookMuxError) {
	if info.RelayFormat == relayconstant.RelayFormatClaude {
		delegate := &claude.Adaptor{}
		delegate.Init(info)
		return delegate.DoResponse(c, resp, info)
	}
	delegate := &openai.Adaptor{}
	delegate.Init(info)
	return delegate.DoResponse(c, resp, info)
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
