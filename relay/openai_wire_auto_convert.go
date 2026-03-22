package relay

import (
	"fmt"
	"net/http"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	relayconstant "github.com/zhongruan0522/new-api/relay/constant"
	"github.com/zhongruan0522/new-api/types"

	"github.com/gin-gonic/gin"
)

// OpenAIWireHelper auto-converts between ChatCompletions and Responses based on channel setting openai_wire_api.
// It only applies to endpoints:
// - /v1/chat/completions
// - /v1/responses
// - /v1/responses/compact (conversion not supported when chat-only)
func OpenAIWireHelper(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	info.InitChannelMeta(c)

	wire, ok := info.ChannelSetting.OpenAIWireAPI.Normalize()
	if !ok {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("invalid channel setting openai_wire_api: %q", info.ChannelSetting.OpenAIWireAPI),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	switch info.RelayMode {
	case relayconstant.RelayModeChatCompletions:
		if wire == dto.OpenAIWireAPIResponses {
			return relayChatDownstreamToResponsesUpstream(c, info)
		}
		return TextHelper(c, info)
	case relayconstant.RelayModeResponses:
		if wire == dto.OpenAIWireAPIChat {
			return relayResponsesDownstreamToChatUpstream(c, info)
		}
		return ResponsesHelper(c, info)
	case relayconstant.RelayModeResponsesCompact:
		if wire == dto.OpenAIWireAPIChat {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("endpoint %q is not supported when channel openai_wire_api=%q", "/v1/responses/compact", wire),
				types.ErrorCodeInvalidRequest,
				http.StatusBadRequest,
				types.ErrOptionWithSkipRetry(),
			)
		}
		return ResponsesHelper(c, info)
	default:
		return types.NewErrorWithStatusCode(
			fmt.Errorf("unsupported relay mode for openai wire conversion: %d", info.RelayMode),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}
}

func relayChatDownstreamToResponsesUpstream(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	chatReq, ok := info.Request.(*dto.GeneralOpenAIRequest)
	if !ok {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected dto.GeneralOpenAIRequest, got %T", info.Request),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	responsesReq, err := relaycommon.ConvertChatCompletionsRequestToResponsesRequest(chatReq)
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, responsesReq)

	includeUsage := true
	if chatReq.StreamOptions != nil {
		includeUsage = chatReq.StreamOptions.IncludeUsage
	}

	snapshot := takeRelayInfoSnapshot(info)
	defer snapshot.restore(info)

	bodySnap, err := takeRequestBodySnapshot(c)
	if err != nil {
		return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
	}
	defer bodySnap.restore(c)

	info.Request = responsesReq
	info.RelayMode = relayconstant.RelayModeResponses
	info.RelayFormat = types.RelayFormatOpenAIResponses
	info.RequestURLPath = "/v1/responses"
	info.IsStream = responsesReq.Stream

	bodyBytes, err := common.Marshal(responsesReq)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	setTemporaryRequestBody(c, bodyBytes)

	if responsesReq.Stream {
		return streamUpstreamWithWireConversion(c, info, dto.OpenAIWireAPIResponses, dto.OpenAIWireAPIChat, includeUsage, ResponsesHelper)
	}
	return nonStreamUpstreamWithWireConversion(c, info, dto.OpenAIWireAPIResponses, dto.OpenAIWireAPIChat, ResponsesHelper)
}

func relayResponsesDownstreamToChatUpstream(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	responsesReq, ok := info.Request.(*dto.OpenAIResponsesRequest)
	if !ok {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected dto.OpenAIResponsesRequest, got %T", info.Request),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	chatReq, err := relaycommon.ConvertResponsesRequestToChatCompletionsRequest(responsesReq)
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, chatReq)
	if responsesReq.Stream && info.SupportStreamOptions {
		chatReq.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
	}

	snapshot := takeRelayInfoSnapshot(info)
	defer snapshot.restore(info)

	bodySnap, err := takeRequestBodySnapshot(c)
	if err != nil {
		return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
	}
	defer bodySnap.restore(c)

	info.Request = chatReq
	info.RelayMode = relayconstant.RelayModeChatCompletions
	info.RelayFormat = types.RelayFormatOpenAI
	info.RequestURLPath = "/v1/chat/completions"
	info.IsStream = chatReq.Stream

	bodyBytes, err := common.Marshal(chatReq)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	setTemporaryRequestBody(c, bodyBytes)

	if chatReq.Stream {
		return streamUpstreamWithWireConversion(c, info, dto.OpenAIWireAPIChat, dto.OpenAIWireAPIResponses, false, TextHelper)
	}
	return nonStreamUpstreamWithWireConversion(c, info, dto.OpenAIWireAPIChat, dto.OpenAIWireAPIResponses, TextHelper)
}

type upstreamHelperFn func(*gin.Context, *relaycommon.RelayInfo) *types.NewAPIError

func streamUpstreamWithWireConversion(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	upstream dto.OpenAIWireAPI,
	downstream dto.OpenAIWireAPI,
	includeUsage bool,
	fn upstreamHelperFn,
) *types.NewAPIError {
	base := c.Writer
	writer, err := newOpenAIWireStreamWriter(base, upstream, downstream, openAIWireStreamOptions{ChatIncludeUsage: includeUsage})
	if err != nil {
		return types.NewError(err, types.ErrorCodeBadResponse, types.ErrOptionWithSkipRetry())
	}
	c.Writer = writer
	defer func() { c.Writer = base }()

	newAPIError := fn(c, info)
	if newAPIError != nil {
		return newAPIError
	}
	if convErr := writer.ConversionErr(); convErr != nil {
		return types.NewError(convErr, types.ErrorCodeBadResponseBody)
	}
	return nil
}

func nonStreamUpstreamWithWireConversion(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	upstream dto.OpenAIWireAPI,
	downstream dto.OpenAIWireAPI,
	fn upstreamHelperFn,
) *types.NewAPIError {
	base := c.Writer
	capture := newOpenAIWireCaptureWriter(base)
	c.Writer = capture

	newAPIError := fn(c, info)
	c.Writer = base
	if newAPIError != nil {
		return newAPIError
	}

	if err := writeConvertedNonStreamResponse(c, capture, upstream, downstream); err != nil {
		return types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	return nil
}
