package siliconflow

import (
	"net/http"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"

	"github.com/NookMux/NookMux/pkg/jsonx"

	"github.com/NookMux/NookMux/internal/relay/helper"
	"github.com/gin-gonic/gin"
)

func siliconflowRerankHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*shared.Usage, *shared.NookMuxError) {
	defer helper.CloseResponseBodyGracefully(resp)

	responseBody, err := common.ReadResponseBody(resp.Body)
	if err != nil {
		return nil, shared.NewOpenAIError(err, shared.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	var siliconflowResp SFRerankResponse
	err = jsonx.Unmarshal(responseBody, &siliconflowResp)
	if err != nil {
		return nil, shared.NewOpenAIError(err, shared.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	usage := &shared.Usage{
		PromptTokens:     siliconflowResp.Meta.Tokens.InputTokens,
		CompletionTokens: siliconflowResp.Meta.Tokens.OutputTokens,
		TotalTokens:      siliconflowResp.Meta.Tokens.InputTokens + siliconflowResp.Meta.Tokens.OutputTokens,
	}
	rerankResp := &shared.RerankResponse{
		Results: siliconflowResp.Results,
		Usage:   *usage,
	}

	jsonResponse, err := jsonx.Marshal(rerankResp)
	if err != nil {
		return nil, shared.NewError(err, shared.ErrorCodeBadResponseBody)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	helper.IOCopyBytesGracefully(c, resp, jsonResponse)
	return usage, nil
}
