package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"

	"github.com/NookMux/NookMux/pkg/jsonx"

	"github.com/NookMux/NookMux/internal/httpapi"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/internal/relay/helper"
	"github.com/gin-gonic/gin"
)

type relayInfoSnapshot struct {
	request        shared.Request
	relayMode      int
	relayFormat    relayconstant.RelayFormat
	requestURLPath string
	isStream       bool
}

func takeRelayInfoSnapshot(info *relaycommon.RelayInfo) relayInfoSnapshot {
	return relayInfoSnapshot{
		request:        info.Request,
		relayMode:      info.RelayMode,
		relayFormat:    info.RelayFormat,
		requestURLPath: info.RequestURLPath,
		isStream:       info.IsStream,
	}
}

func (s relayInfoSnapshot) restore(info *relaycommon.RelayInfo) {
	info.Request = s.request
	info.RelayMode = s.relayMode
	info.RelayFormat = s.relayFormat
	info.RequestURLPath = s.requestURLPath
	info.IsStream = s.isStream
}

type requestBodySnapshot struct {
	body    []byte
	storage any
}

type openAIWireConversionOptions struct {
	ChatIncludeUsage bool
	ToolContext      *relaycommon.OpenAIWireToolContext
}

func takeRequestBodySnapshot(c *gin.Context) (requestBodySnapshot, error) {
	body, err := httpapi.GetRequestBody(c)
	if err != nil {
		return requestBodySnapshot{}, err
	}
	storage, _ := c.Get(httpapi.KeyBodyStorage)
	return requestBodySnapshot{body: body, storage: storage}, nil
}

func (s requestBodySnapshot) restore(c *gin.Context) {
	c.Set(httpapi.KeyRequestBody, s.body)
	c.Set(httpapi.KeyBodyStorage, s.storage)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(s.body))
	c.Request.ContentLength = int64(len(s.body))
}

func setTemporaryRequestBody(c *gin.Context, body []byte) {
	c.Set(httpapi.KeyBodyStorage, nil)
	c.Set(httpapi.KeyRequestBody, body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	c.Request.ContentLength = int64(len(body))
}

func writeConvertedNonStreamResponse(c *gin.Context, captured *openAIWireCaptureWriter, upstream shared.OpenAIWireAPI, downstream shared.OpenAIWireAPI, opts openAIWireConversionOptions) error {
	if captured == nil {
		return fmt.Errorf("captured writer is nil")
	}
	body := captured.BodyBytes()
	if len(body) == 0 {
		return fmt.Errorf("empty upstream response body")
	}

	converted, err := convertNonStreamBody(body, upstream, downstream, opts)
	if err != nil {
		return err
	}

	copyHeaders(c.Writer.Header(), captured.Header())
	c.Writer.WriteHeader(captured.Status())
	_, err = c.Writer.Write(converted)
	return err
}

func convertNonStreamBody(body []byte, upstream shared.OpenAIWireAPI, downstream shared.OpenAIWireAPI, opts openAIWireConversionOptions) ([]byte, error) {
	switch {
	case upstream == shared.OpenAIWireAPIResponses && downstream == shared.OpenAIWireAPIChat:
		var resp shared.OpenAIResponsesResponse
		if err := jsonx.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal responses response failed: %w", err)
		}
		chatResp, err := relaycommon.ConvertResponsesResponseToChatCompletionResponse(&resp)
		if err != nil {
			return nil, err
		}
		return jsonx.Marshal(chatResp)
	case upstream == shared.OpenAIWireAPIChat && downstream == shared.OpenAIWireAPIResponses:
		var chatResp shared.OpenAITextResponse
		if err := jsonx.Unmarshal(body, &chatResp); err != nil {
			return nil, fmt.Errorf("unmarshal chat completion response failed: %w", err)
		}
		resp, err := relaycommon.ConvertChatCompletionResponseToResponsesResponseWithToolContext(&chatResp, opts.ToolContext)
		if err != nil {
			return nil, err
		}
		return jsonx.Marshal(resp)
	default:
		return nil, fmt.Errorf("unsupported non-stream conversion: %s -> %s", upstream, downstream)
	}
}

func copyHeaders(dst http.Header, src http.Header) {
	for k, vals := range src {
		if strings.TrimSpace(k) == "" || !helper.ShouldCopyUpstreamHeader(nil, k, vals) {
			continue
		}
		dst[k] = append([]string(nil), vals...)
	}
}
