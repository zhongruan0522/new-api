package stream

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/relay/wire/convert"
	"github.com/gin-gonic/gin"
)

type StreamOptions struct {
	ChatIncludeUsage bool
	ToolContext      *convert.OpenAIWireToolContext
}

type StreamWriter struct {
	gin.ResponseWriter

	converter OpenAIWireStreamConverter
	pending   []byte
	lastErr   error
}

func NewStreamWriter(
	base gin.ResponseWriter,
	upstream shared.OpenAIWireAPI,
	downstream shared.OpenAIWireAPI,
	opts StreamOptions,
) (*StreamWriter, error) {
	var converter OpenAIWireStreamConverter
	switch {
	case upstream == shared.OpenAIWireAPIResponses && downstream == shared.OpenAIWireAPIChat:
		converter = newResponsesToChatStreamConverter(opts.ChatIncludeUsage)
	case upstream == shared.OpenAIWireAPIChat && downstream == shared.OpenAIWireAPIResponses:
		converter = newChatToResponsesStreamConverter(opts.ToolContext)
	default:
		return nil, fmt.Errorf("unsupported stream conversion: %s -> %s", upstream, downstream)
	}

	return &StreamWriter{
		ResponseWriter: base,
		converter:      converter,
	}, nil
}

func (w *StreamWriter) Write(p []byte) (int, error) {
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

func (w *StreamWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *StreamWriter) ConversionErr() error {
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
