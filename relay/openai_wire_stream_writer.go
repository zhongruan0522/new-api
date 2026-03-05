package relay

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

type openAIWireStreamOptions struct {
	ChatIncludeUsage bool
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
		converter = newChatToResponsesStreamConverter()
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
	idx := bytes.Index(buf, []byte("\n\n"))
	if idx < 0 {
		return "", buf, false
	}
	end := idx + len("\n\n")
	return string(buf[:end]), buf[end:], true
}

func parseSSEFrame(frame string) (event string, data string, raw string, err error) {
	raw = frame
	trimmed := strings.TrimSuffix(frame, "\n\n")
	if strings.HasPrefix(trimmed, ":") {
		return "", "", raw, nil
	}

	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
	return event, data, raw, nil
}
