package common_handler

import (
	"io"
	"net/http"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/types"

	"github.com/gin-gonic/gin"
)

func RerankHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)
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
