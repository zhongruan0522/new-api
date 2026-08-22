package stream

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CaptureWriter struct {
	gin.ResponseWriter
	header  http.Header
	status  int
	written bool
	body    bytes.Buffer
}

func NewCaptureWriter(base gin.ResponseWriter) *CaptureWriter {
	return &CaptureWriter{
		ResponseWriter: base,
		header:         make(http.Header),
		status:         http.StatusOK,
	}
}

func (w *CaptureWriter) Header() http.Header {
	return w.header
}

func (w *CaptureWriter) WriteHeader(code int) {
	if code <= 0 {
		return
	}
	w.status = code
}

func (w *CaptureWriter) WriteHeaderNow() {}

func (w *CaptureWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.written = true
	return w.body.Write(p)
}

func (w *CaptureWriter) WriteString(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	w.written = true
	return w.body.WriteString(s)
}

func (w *CaptureWriter) ReadFrom(r io.Reader) (int64, error) {
	if r == nil {
		return 0, nil
	}
	w.written = true
	return w.body.ReadFrom(r)
}

func (w *CaptureWriter) Flush() {}

func (w *CaptureWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *CaptureWriter) Size() int {
	return w.body.Len()
}

func (w *CaptureWriter) Written() bool {
	return w.written
}

func (w *CaptureWriter) Pusher() http.Pusher {
	if p := w.ResponseWriter.Pusher(); p != nil {
		return p
	}
	return nil
}

func (w *CaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.Hijack()
}

func (w *CaptureWriter) BodyBytes() []byte {
	return w.body.Bytes()
}
