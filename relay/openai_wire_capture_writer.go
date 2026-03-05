package relay

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

type openAIWireCaptureWriter struct {
	gin.ResponseWriter
	header  http.Header
	status  int
	written bool
	body    bytes.Buffer
}

func newOpenAIWireCaptureWriter(base gin.ResponseWriter) *openAIWireCaptureWriter {
	return &openAIWireCaptureWriter{
		ResponseWriter: base,
		header:         make(http.Header),
		status:         http.StatusOK,
	}
}

func (w *openAIWireCaptureWriter) Header() http.Header {
	return w.header
}

func (w *openAIWireCaptureWriter) WriteHeader(code int) {
	if code <= 0 {
		return
	}
	w.status = code
}

func (w *openAIWireCaptureWriter) WriteHeaderNow() {}

func (w *openAIWireCaptureWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.written = true
	return w.body.Write(p)
}

func (w *openAIWireCaptureWriter) WriteString(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	w.written = true
	return w.body.WriteString(s)
}

func (w *openAIWireCaptureWriter) ReadFrom(r io.Reader) (int64, error) {
	if r == nil {
		return 0, nil
	}
	w.written = true
	return w.body.ReadFrom(r)
}

func (w *openAIWireCaptureWriter) Flush() {}

func (w *openAIWireCaptureWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *openAIWireCaptureWriter) Size() int {
	return w.body.Len()
}

func (w *openAIWireCaptureWriter) Written() bool {
	return w.written
}

func (w *openAIWireCaptureWriter) Pusher() http.Pusher {
	if p := w.ResponseWriter.Pusher(); p != nil {
		return p
	}
	return nil
}

func (w *openAIWireCaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.Hijack()
}

func (w *openAIWireCaptureWriter) BodyBytes() []byte {
	return w.body.Bytes()
}
