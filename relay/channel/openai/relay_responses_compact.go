package openai

import (
	"net/http"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/dto"
	"github.com/NookMux/NookMux/pkg/jsonx"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/relay/helper"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/types"

	"github.com/gin-gonic/gin"
)

func OaiResponsesCompactionHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NookMuxError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := common.ReadResponseBody(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var compactResp dto.OpenAIResponsesCompactionResponse
	if err := jsonx.Unmarshal(responseBody, &compactResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := compactResp.GetOpenAIError(); oaiError != nil && oaiError.Message != "" {
		return nil, types.WithOpenAIError(*oaiError, upstreamErrorStatusCode(resp.StatusCode, oaiError))
	}

	responseBody = helper.MaskTopLevelModelJSON(responseBody, info)

	service.IOCopyBytesGracefully(c, resp, responseBody)

	usage := dto.Usage{}
	relaycommon.ApplyResponsesUsageToChatUsage(&usage, compactResp.Usage)

	return &usage, nil
}
