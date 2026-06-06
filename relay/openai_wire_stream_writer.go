package relay

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

type openAIWireStreamOptions struct {
	ChatIncludeUsage bool
	ToolContext      *relaycommon.OpenAIWireToolContext
}

type openAIWireStreamWriter struct {
	gin.ResponseWriter

	converter openAIWireStreamConverter
	pending   []byte
	lastErr   error
}

type openAIWireStreamConverter interface {
	ConvertFrame(event string, data string, rawFrame string) (string, error)
	Err() error
}

func newOpenAIWireStreamWriter(
	base gin.ResponseWriter,
	upstream dto.OpenAIWireAPI,
	downstream dto.OpenAIWireAPI,
	opts openAIWireStreamOptions,
) (*openAIWireStreamWriter, error) {
	var converter openAIWireStreamConverter
	switch {
	case upstream == dto.OpenAIWireAPIResponses && downstream == dto.OpenAIWireAPIChat:
		converter = newResponsesToChatStreamConverter(opts.ChatIncludeUsage)
	case upstream == dto.OpenAIWireAPIChat && downstream == dto.OpenAIWireAPIResponses:
		converter = relaycommon.NewChatToResponsesStreamConverter(opts.ToolContext)
	default:
		return nil, fmt.Errorf("unsupported stream conversion: %s -> %s", upstream, downstream)
	}

	return &openAIWireStreamWriter{
		ResponseWriter: base,
		converter:      converter,
	}, nil
}

func (w *openAIWireStreamWriter) Write(p []byte) (int, error) {
	if w.lastErr != nil {
		return 0, w.lastErr
	}
	if len(p) == 0 {
		return 0, nil
	}

	w.pending = append(w.pending, p...)
	for {
		frame, rest, ok := splitSSEFrame(w.pending)
		if !ok {
			break
		}
		w.pending = rest

		event, data, raw, err := parseSSEFrame(frame)
		if err != nil {
			w.lastErr = err
			return 0, err
		}

		out, err := w.converter.ConvertFrame(event, data, raw)
		if err != nil {
			w.lastErr = err
			return 0, err
		}
		if out != "" {
			if _, err := w.ResponseWriter.Write([]byte(out)); err != nil {
				w.lastErr = err
				return 0, err
			}
		}
	}

	return len(p), nil
}

func (w *openAIWireStreamWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *openAIWireStreamWriter) ConversionErr() error {
	if w.lastErr != nil {
		return w.lastErr
	}
	return w.converter.Err()
}

func splitSSEFrame(buf []byte) (frame string, rest []byte, ok bool) {
	idx, delimiterLen := firstSSEDelimiter(buf)
	if idx < 0 {
		return "", buf, false
	}
	end := idx + delimiterLen
	return string(buf[:end]), buf[end:], true
}

func firstSSEDelimiter(buf []byte) (int, int) {
	lf := bytes.Index(buf, []byte("\n\n"))
	crlf := bytes.Index(buf, []byte("\r\n\r\n"))
	switch {
	case lf < 0 && crlf < 0:
		return -1, 0
	case crlf < 0 || (lf >= 0 && lf < crlf):
		return lf, len("\n\n")
	default:
		return crlf, len("\r\n\r\n")
	}
}

func parseSSEFrame(frame string) (event string, data string, raw string, err error) {
	raw = frame
	trimmed := strings.TrimSuffix(strings.TrimSuffix(frame, "\r\n\r\n"), "\n\n")
	if strings.HasPrefix(trimmed, ":") {
		return "", "", raw, nil
	}

	lines := strings.Split(strings.ReplaceAll(trimmed, "\r\n", "\n"), "\n")
	dataLines := make([]string, 0, 1)
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	data = strings.Join(dataLines, "\n")
	return event, data, raw, nil
}
