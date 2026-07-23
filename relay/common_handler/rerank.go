package common_handler

import (
	"net/http"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/dto"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/types"

	"github.com/gin-gonic/gin"
)

func RerankHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := common.ReadResponseBody(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	if common.DebugEnabled {
		println("reranker response body: ", string(responseBody))
	}
	var rerankResp dto.RerankResponse
	err = common.Unmarshal(responseBody, &rerankResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	rerankResp.Usage.PromptTokens = rerankResp.Usage.TotalTokens

	c.Writer.Header().Set("Content-Type", "application/json")
	c.JSON(http.StatusOK, rerankResp)
	return &rerankResp.Usage, nil
}
