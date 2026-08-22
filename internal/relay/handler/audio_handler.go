package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/internal/relay/core"
	"github.com/NookMux/NookMux/internal/relay/helper"

	billing "github.com/NookMux/NookMux/internal/domain/billing"
	"github.com/gin-gonic/gin"
)

func AudioHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *shared.NookMuxError) {
	info.InitChannelMeta(c)

	audioReq, ok := info.Request.(*shared.AudioRequest)
	if !ok {
		return shared.NewError(errors.New("invalid request type"), shared.ErrorCodeInvalidRequest, shared.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(audioReq)
	if err != nil {
		return shared.NewError(fmt.Errorf("failed to copy request to AudioRequest: %w", err), shared.ErrorCodeInvalidRequest, shared.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeChannelModelMappedError, shared.ErrOptionWithSkipRetry())
	}

	adaptor := core.GetAdaptor(info.ApiType)
	if adaptor == nil {
		return shared.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), shared.ErrorCodeInvalidApiType, shared.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	ioReader, err := adaptor.ConvertAudioRequest(c, info, *request)
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeConvertRequestFailed, shared.ErrOptionWithSkipRetry())
	}

	resp, err := adaptor.DoRequest(c, info, ioReader)
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeDoRequestFailed)
	}
	statusCodeMappingStr := c.GetString("status_code_mapping")

	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK {
			newAPIError = helper.RelayErrorHandler(c.Request.Context(), httpResp, false)
			// reset status code 重置状态码
			helper.ResetStatusCode(newAPIError, statusCodeMappingStr)
			return newAPIError
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		helper.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}
	// 音色日志：记录 MiniMax TTS 实际使用的 voice_id（经重定向后的）
	var extraContent string
	if voiceID := c.GetString("minimax_voice_id"); voiceID != "" {
		extraContent = "voice_id: " + voiceID
	}

	if usage.(*shared.Usage).CompletionTokenDetails.AudioTokens > 0 || usage.(*shared.Usage).PromptTokensDetails.AudioTokens > 0 {
		if apiErr := billing.PostAudioConsumeQuota(c, info, usage.(*shared.Usage), extraContent); apiErr != nil {
			return apiErr
		}
	} else {
		settlement, apiErr := billing.CalculateUsage(c, info, usage.(*shared.Usage))
		if apiErr != nil {
			return apiErr
		}
		if apiErr := billing.ApplyQuota(c, info, settlement); apiErr != nil {
			return apiErr
		}
	}

	return nil
}
