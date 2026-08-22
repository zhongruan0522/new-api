package openai

import (
	"net/http"

	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/internal/relay/helper"
	"github.com/NookMux/NookMux/internal/relay/wire/convert"

	"github.com/NookMux/NookMux/pkg/jsonx"

	"github.com/gin-gonic/gin"
)

func OaiResponsesCompactionHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*shared.Usage, *shared.NookMuxError) {
	defer helper.CloseResponseBodyGracefully(resp)

	responseBody, err := helper.ReadResponseBody(resp.Body)
	if err != nil {
		return nil, shared.NewOpenAIError(err, shared.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var compactResp shared.OpenAIResponsesCompactionResponse
	if err := jsonx.Unmarshal(responseBody, &compactResp); err != nil {
		return nil, shared.NewOpenAIError(err, shared.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := compactResp.GetOpenAIError(); oaiError != nil && oaiError.Message != "" {
		return nil, shared.WithOpenAIError(*oaiError, upstreamErrorStatusCode(resp.StatusCode, oaiError))
	}

	responseBody = helper.MaskTopLevelModelJSON(responseBody, info)

	helper.IOCopyBytesGracefully(c, resp, responseBody)

	usage := shared.Usage{}
	convert.ApplyResponsesUsageToChatUsage(&usage, compactResp.Usage)

	return &usage, nil
}
