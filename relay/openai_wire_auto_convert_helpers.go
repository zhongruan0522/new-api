package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type relayInfoSnapshot struct {
	request        dto.Request
	relayMode      int
	relayFormat    types.RelayFormat
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

func takeRequestBodySnapshot(c *gin.Context) (requestBodySnapshot, error) {
	body, err := common.GetRequestBody(c)
	if err != nil {
		return requestBodySnapshot{}, err
	}
	storage, _ := c.Get(common.KeyBodyStorage)
	return requestBodySnapshot{body: body, storage: storage}, nil
}

func (s requestBodySnapshot) restore(c *gin.Context) {
	c.Set(common.KeyRequestBody, s.body)
	c.Set(common.KeyBodyStorage, s.storage)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(s.body))
	c.Request.ContentLength = int64(len(s.body))
}

func setTemporaryRequestBody(c *gin.Context, body []byte) {
	c.Set(common.KeyBodyStorage, nil)
	c.Set(common.KeyRequestBody, body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	c.Request.ContentLength = int64(len(body))
}

func writeConvertedNonStreamResponse(c *gin.Context, captured *openAIWireCaptureWriter, upstream dto.OpenAIWireAPI, downstream dto.OpenAIWireAPI) error {
	if captured == nil {
		return fmt.Errorf("captured writer is nil")
	}
	body := captured.BodyBytes()
	if len(body) == 0 {
		return fmt.Errorf("empty upstream response body")
	}

	converted, err := convertNonStreamBody(body, upstream, downstream)
	if err != nil {
		return err
	}

	copyHeaders(c.Writer.Header(), captured.Header())
	c.Writer.WriteHeader(captured.Status())
	_, err = c.Writer.Write(converted)
	return err
}

func convertNonStreamBody(body []byte, upstream dto.OpenAIWireAPI, downstream dto.OpenAIWireAPI) ([]byte, error) {
	switch {
	case upstream == dto.OpenAIWireAPIResponses && downstream == dto.OpenAIWireAPIChat:
		var resp dto.OpenAIResponsesResponse
		if err := common.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal responses response failed: %w", err)
		}
		chatResp, err := relaycommon.ConvertResponsesResponseToChatCompletionResponse(&resp)
		if err != nil {
			return nil, err
		}
		return common.Marshal(chatResp)
	case upstream == dto.OpenAIWireAPIChat && downstream == dto.OpenAIWireAPIResponses:
		var chatResp dto.OpenAITextResponse
		if err := common.Unmarshal(body, &chatResp); err != nil {
			return nil, fmt.Errorf("unmarshal chat completion response failed: %w", err)
		}
		resp, err := relaycommon.ConvertChatCompletionResponseToResponsesResponse(&chatResp)
		if err != nil {
			return nil, err
		}
		return common.Marshal(resp)
	default:
		return nil, fmt.Errorf("unsupported non-stream conversion: %s -> %s", upstream, downstream)
	}
}

func copyHeaders(dst http.Header, src http.Header) {
	for k, vals := range src {
		if strings.TrimSpace(k) == "" {
			continue
		}
		dst[k] = append([]string(nil), vals...)
	}
}
