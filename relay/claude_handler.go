package relay

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/dto"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/relay/helper"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/setting/model_setting"
	"github.com/NookMux/NookMux/types"

	"github.com/gin-gonic/gin"
)

func ClaudeHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {

	info.InitChannelMeta(c)

	claudeReq, ok := info.Request.(*dto.ClaudeRequest)

	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected *dto.ClaudeRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(claudeReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ClaudeRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	if request.MaxTokens == 0 {
		request.MaxTokens = uint(model_setting.GetClaudeSettings().GetDefaultMaxTokens(request.Model))
	}

	var requestBody io.Reader
	if info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		jsonData, err := storage.Bytes()
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		jsonData, err = relaycommon.RemoveClaudeDisabledFields(jsonData, info.ChannelOtherSettings)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		defer closer.Close()
		info.UpstreamRequestBodySize = size
		requestBody = body
	} else {
		convertedRequest, err := adaptor.ConvertClaudeRequest(c, info, request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		// remove disabled fields for Claude API
		jsonData, err = relaycommon.RemoveClaudeDisabledFields(jsonData, info.ChannelOtherSettings)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		// apply param override
		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			if err != nil {
				return types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
			}
		}

		if common.DebugEnabled {
			println("requestBody: ", string(jsonData))
		}
		body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		defer closer.Close()
		jsonData = nil
		info.UpstreamRequestBodySize = size
		requestBody = body
	}

	// 通用思维强度解析：adaptor 未设置时从原始请求兜底提取
	relaycommon.EnsureReasoningEffort(info, info.Request)

	statusCodeMappingStr := c.GetString("status_code_mapping")
	var httpResp *http.Response
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	if resp != nil {
		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode != http.StatusOK {
			newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			// reset status code 重置状态码
			service.ResetStatusCode(newAPIError, statusCodeMappingStr)
			return newAPIError
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	//log.Printf("usage: %v", usage)
	if newAPIError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	if apiErr := service.PostClaudeConsumeQuota(c, info, usage.(*dto.Usage)); apiErr != nil {
		return apiErr
	}
	return nil
}
