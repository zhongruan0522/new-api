package core

import (
	"fmt"

	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"

	billing "github.com/NookMux/NookMux/internal/domain/billing"
	"github.com/NookMux/NookMux/internal/relay/helper"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func WssHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *shared.NookMuxError) {
	info.InitChannelMeta(c)

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return shared.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), shared.ErrorCodeInvalidApiType, shared.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)
	//var requestBody io.Reader
	//firstWssRequest, _ := c.Get("first_wss_request")
	//requestBody = bytes.NewBuffer(firstWssRequest.([]byte))

	statusCodeMappingStr := c.GetString("status_code_mapping")
	resp, err := adaptor.DoRequest(c, info, nil)
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeDoRequestFailed)
	}

	if resp != nil {
		info.TargetWs = resp.(*websocket.Conn)
		defer info.TargetWs.Close()
	}

	usage, newAPIError := adaptor.DoResponse(c, nil, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		helper.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}
	if apiErr := billing.PostWssConsumeQuota(c, info, info.UpstreamModelName, usage.(*shared.RealtimeUsage), ""); apiErr != nil {
		return apiErr
	}
	return nil
}
