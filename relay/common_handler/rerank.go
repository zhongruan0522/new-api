package common_handler

import (
	"net/http"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/dto"
	"github.com/NookMux/NookMux/pkg/jsonx"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/types"

	"github.com/gin-gonic/gin"
)

func RerankHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NookMuxError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := common.ReadResponseBody(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	if common.DebugEnabled {
		println("reranker response body: ", string(responseBody))
	}
	var rerankResp dto.RerankResponse
	err = jsonx.Unmarshal(responseBody, &rerankResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	// 部分上游/中间网关会把 429/5xx 错误转成 HTTP 200 + error body 下发。
	// 识别后向上暴露真实上游错误，避免计费阶段因 usage 全零被误记为
	// 「502 上游没有返回计费信息」。
	var errProbe dto.SimpleResponse
	if probeErr := jsonx.Unmarshal(responseBody, &errProbe); probeErr == nil {
		if oaiError := errProbe.GetOpenAIError(); oaiError != nil && oaiError.Message != "" {
			return nil, types.WithOpenAIError(*oaiError, service.UpstreamErrorStatusCode(resp.StatusCode, oaiError.Code))
		}
	}

	rerankResp.Usage.PromptTokens = rerankResp.Usage.TotalTokens

	c.Writer.Header().Set("Content-Type", "application/json")
	c.JSON(http.StatusOK, rerankResp)
	return &rerankResp.Usage, nil
}
