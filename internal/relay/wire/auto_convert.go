package wire

import (
	"fmt"
	"net/http"

	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/internal/relay/handler"
	"github.com/NookMux/NookMux/internal/relay/wire/convert"
	"github.com/NookMux/NookMux/internal/relay/wire/stream"

	"github.com/NookMux/NookMux/pkg/jsonx"

	"github.com/gin-gonic/gin"
)

// OpenAIWireHelper auto-converts between ChatCompletions and Responses based on channel setting openai_wire_api.
// It only applies to endpoints:
// - /v1/chat/completions
// - /v1/responses
// - /v1/responses/compact (conversion not supported when chat-only)
func OpenAIWireHelper(c *gin.Context, info *relaycommon.RelayInfo) *shared.NookMuxError {
	info.InitChannelMeta(c)

	wire, ok := info.ChannelSetting.OpenAIWireAPI.Normalize()
	if !ok {
		return shared.NewErrorWithStatusCode(
			fmt.Errorf("invalid channel setting openai_wire_api: %q", info.ChannelSetting.OpenAIWireAPI),
			shared.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			shared.ErrOptionWithSkipRetry(),
		)
	}

	switch info.RelayMode {
	case relayconstant.RelayModeChatCompletions:
		if wire == shared.OpenAIWireAPIResponses {
			return relayChatDownstreamToResponsesUpstream(c, info)
		}
		return handler.TextHelper(c, info)
	case relayconstant.RelayModeResponses:
		if wire == shared.OpenAIWireAPIChat {
			return relayResponsesDownstreamToChatUpstream(c, info)
		}
		return handler.ResponsesHelper(c, info)
	case relayconstant.RelayModeResponsesCompact:
		if wire == shared.OpenAIWireAPIChat {
			return shared.NewErrorWithStatusCode(
				fmt.Errorf("endpoint %q is not supported when channel openai_wire_api=%q", "/v1/responses/compact", wire),
				shared.ErrorCodeInvalidRequest,
				http.StatusBadRequest,
				shared.ErrOptionWithSkipRetry(),
			)
		}
		return handler.ResponsesHelper(c, info)
	default:
		return shared.NewErrorWithStatusCode(
			fmt.Errorf("unsupported relay mode for openai wire conversion: %d", info.RelayMode),
			shared.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			shared.ErrOptionWithSkipRetry(),
		)
	}
}

func relayChatDownstreamToResponsesUpstream(c *gin.Context, info *relaycommon.RelayInfo) *shared.NookMuxError {
	chatReq, ok := info.Request.(*shared.GeneralOpenAIRequest)
	if !ok {
		return shared.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected shared.GeneralOpenAIRequest, got %T", info.Request),
			shared.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			shared.ErrOptionWithSkipRetry(),
		)
	}

	responsesReq, err := convert.ConvertChatCompletionsRequestToResponsesRequest(chatReq)
	if err != nil {
		return shared.NewErrorWithStatusCode(err, shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
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
		return shared.NewError(err, shared.ErrorCodeReadRequestBodyFailed, shared.ErrOptionWithSkipRetry())
	}
	defer bodySnap.restore(c)

	info.Request = responsesReq
	info.RelayMode = relayconstant.RelayModeResponses
	info.RelayFormat = relayconstant.RelayFormatOpenAIResponses
	info.RequestURLPath = "/v1/responses"
	info.IsStream = responsesReq.Stream

	bodyBytes, err := jsonx.Marshal(responsesReq)
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeConvertRequestFailed, shared.ErrOptionWithSkipRetry())
	}
	setTemporaryRequestBody(c, bodyBytes)

	if responsesReq.Stream {
		return streamUpstreamWithWireConversion(c, info, shared.OpenAIWireAPIResponses, shared.OpenAIWireAPIChat, openAIWireConversionOptions{ChatIncludeUsage: includeUsage}, handler.ResponsesHelper)
	}
	return nonStreamUpstreamWithWireConversion(c, info, shared.OpenAIWireAPIResponses, shared.OpenAIWireAPIChat, openAIWireConversionOptions{}, handler.ResponsesHelper)
}

func relayResponsesDownstreamToChatUpstream(c *gin.Context, info *relaycommon.RelayInfo) *shared.NookMuxError {
	responsesReq, ok := info.Request.(*shared.OpenAIResponsesRequest)
	if !ok {
		return shared.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected shared.OpenAIResponsesRequest, got %T", info.Request),
			shared.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			shared.ErrOptionWithSkipRetry(),
		)
	}

	chatReq, toolContext, err := convert.ConvertResponsesRequestToChatCompletionsRequestWithToolContext(responsesReq)
	if err != nil {
		return shared.NewErrorWithStatusCode(err, shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, chatReq)
	if responsesReq.Stream && info.SupportStreamOptions {
		chatReq.StreamOptions = &shared.StreamOptions{IncludeUsage: true}
	}

	snapshot := takeRelayInfoSnapshot(info)
	defer snapshot.restore(info)

	bodySnap, err := takeRequestBodySnapshot(c)
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeReadRequestBodyFailed, shared.ErrOptionWithSkipRetry())
	}
	defer bodySnap.restore(c)

	info.Request = chatReq
	info.RelayMode = relayconstant.RelayModeChatCompletions
	info.RelayFormat = relayconstant.RelayFormatOpenAI
	info.RequestURLPath = "/v1/chat/completions"
	info.IsStream = chatReq.Stream

	bodyBytes, err := jsonx.Marshal(chatReq)
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeConvertRequestFailed, shared.ErrOptionWithSkipRetry())
	}
	setTemporaryRequestBody(c, bodyBytes)

	if chatReq.Stream {
		return streamUpstreamWithWireConversion(c, info, shared.OpenAIWireAPIChat, shared.OpenAIWireAPIResponses, openAIWireConversionOptions{ToolContext: toolContext}, handler.TextHelper)
	}
	return nonStreamUpstreamWithWireConversion(c, info, shared.OpenAIWireAPIChat, shared.OpenAIWireAPIResponses, openAIWireConversionOptions{ToolContext: toolContext}, handler.TextHelper)
}

type upstreamHelperFn func(*gin.Context, *relaycommon.RelayInfo) *shared.NookMuxError

func streamUpstreamWithWireConversion(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	upstream shared.OpenAIWireAPI,
	downstream shared.OpenAIWireAPI,
	opts openAIWireConversionOptions,
	fn upstreamHelperFn,
) *shared.NookMuxError {
	base := c.Writer
	writer, err := stream.NewStreamWriter(base, upstream, downstream, stream.StreamOptions(opts))
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeBadResponse, shared.ErrOptionWithSkipRetry())
	}
	c.Writer = writer
	defer func() { c.Writer = base }()

	newAPIError := fn(c, info)
	if newAPIError != nil {
		return newAPIError
	}
	if convErr := writer.ConversionErr(); convErr != nil {
		return shared.NewError(convErr, shared.ErrorCodeBadResponseBody)
	}
	return nil
}

func nonStreamUpstreamWithWireConversion(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	upstream shared.OpenAIWireAPI,
	downstream shared.OpenAIWireAPI,
	opts openAIWireConversionOptions,
	fn upstreamHelperFn,
) *shared.NookMuxError {
	base := c.Writer
	capture := stream.NewCaptureWriter(base)
	c.Writer = capture

	newAPIError := fn(c, info)
	c.Writer = base
	if newAPIError != nil {
		return newAPIError
	}

	if err := writeConvertedNonStreamResponse(c, capture, upstream, downstream, opts); err != nil {
		return shared.NewError(err, shared.ErrorCodeBadResponseBody)
	}
	return nil
}
