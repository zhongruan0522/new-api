package gemini

import (
	"fmt"
	"net/http"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/constant"
	"github.com/NookMux/NookMux/dto"
	"github.com/NookMux/NookMux/logger"
	"github.com/NookMux/NookMux/pkg/jsonx"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/relay/helper"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/types"

	"github.com/gin-gonic/gin"
)

func GeminiTextGenerationHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NookMuxError) {
	defer service.CloseResponseBodyGracefully(resp)

	// 读取响应体
	responseBody, err := common.ReadResponseBody(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	if common.DebugEnabled {
		println(string(responseBody))
	}

	// 解析为 Gemini 原生响应格式
	var geminiResponse dto.GeminiChatResponse
	err = jsonx.Unmarshal(responseBody, &geminiResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	// 上游在 HTTP 200 中携带错误载荷（Gemini/中间网关会把 429/5xx 转成
	// 200 + {"error":{...}} 下发）：保留真实上游错误，避免计费阶段因
	// totalTokens=0 被误记为「502 上游没有返回计费信息」。
	if geminiResponse.Error != nil && geminiResponse.Error.Message != "" {
		return nil, types.WithOpenAIError(types.OpenAIError{
			Message: geminiResponse.Error.Message,
			Type:    "upstream_error",
			Code:    geminiResponse.Error.Status,
		}, service.UpstreamErrorStatusCode(resp.StatusCode, geminiResponse.Error.Code))
	}

	if len(geminiResponse.Candidates) == 0 && geminiResponse.PromptFeedback != nil && geminiResponse.PromptFeedback.BlockReason != nil {
		common.SetContextKey(c, constant.ContextKeyAdminRejectReason, fmt.Sprintf("gemini_block_reason=%s", *geminiResponse.PromptFeedback.BlockReason))
	}

	// 计算使用量（基于 UsageMetadata）
	usage := service.GeminiUsageMetadataToOpenAIUsage(geminiResponse.UsageMetadata)

	service.IOCopyBytesGracefully(c, resp, responseBody)

	return &usage, nil
}

func NativeGeminiEmbeddingHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NookMuxError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := common.ReadEmbeddingResponseBody(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	if common.DebugEnabled {
		println(string(responseBody))
	}

	usage := service.ResponseText2Usage(c, "", info.UpstreamModelName, info.GetEstimatePromptTokens())

	if info.IsGeminiBatchEmbedding {
		var geminiResponse dto.GeminiBatchEmbeddingResponse
		err = jsonx.Unmarshal(responseBody, &geminiResponse)
		if err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
	} else {
		var geminiResponse dto.GeminiEmbeddingResponse
		err = jsonx.Unmarshal(responseBody, &geminiResponse)
		if err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
	}

	service.IOCopyBytesGracefully(c, resp, responseBody)

	return usage, nil
}

func GeminiTextGenerationStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NookMuxError) {
	helper.SetEventStreamHeaders(c)

	return geminiStreamHandler(c, info, resp, func(data string, geminiResponse *dto.GeminiChatResponse) bool {
		err := helper.StringData(c, data)
		if err != nil {
			logger.LogError(c, "failed to write stream data: "+err.Error())
			return false
		}
		info.SendResponseCount++
		return true
	})
}
