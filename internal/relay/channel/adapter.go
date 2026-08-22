package channel

import (
	"io"
	"net/http"

	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"

	"github.com/gin-gonic/gin"
)

type Adaptor interface {
	// Init IsStream bool
	Init(info *relaycommon.RelayInfo)
	GetRequestURL(info *relaycommon.RelayInfo) (string, error)
	SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error
	ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *shared.GeneralOpenAIRequest) (any, error)
	ConvertRerankRequest(c *gin.Context, relayMode int, request shared.RerankRequest) (any, error)
	ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request shared.EmbeddingRequest) (any, error)
	ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request shared.AudioRequest) (io.Reader, error)
	ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request shared.ImageRequest) (any, error)
	ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request shared.OpenAIResponsesRequest) (any, error)
	DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error)
	DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *shared.NookMuxError)
	GetModelList() []string
	GetChannelName() string
	ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *shared.ClaudeRequest) (any, error)
	ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *shared.GeminiChatRequest) (any, error)
}
