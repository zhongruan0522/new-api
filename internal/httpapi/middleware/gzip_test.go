package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
)

func TestDecompressRequestMiddlewareSupportsRequestEncodings(t *testing.T) {
	const payload = `{"model":"test-model"}`

	tests := []struct {
		name    string
		header  string
		encoder func([]byte) ([]byte, error)
	}{
		{
			name:   "gzip",
			header: "gzip",
			encoder: func(body []byte) ([]byte, error) {
				var compressed bytes.Buffer
				writer := gzip.NewWriter(&compressed)
				if _, err := writer.Write(body); err != nil {
					return nil, err
				}
				if err := writer.Close(); err != nil {
					return nil, err
				}
				return compressed.Bytes(), nil
			},
		},
		{
			name:   "brotli",
			header: "br",
			encoder: func(body []byte) ([]byte, error) {
				var compressed bytes.Buffer
				writer := brotli.NewWriter(&compressed)
				if _, err := writer.Write(body); err != nil {
					return nil, err
				}
				if err := writer.Close(); err != nil {
					return nil, err
				}
				return compressed.Bytes(), nil
			},
		},
		{
			name:   "zstd",
			header: "zstd",
			encoder: func(body []byte) ([]byte, error) {
				var compressed bytes.Buffer
				writer, err := zstd.NewWriter(&compressed)
				if err != nil {
					return nil, err
				}
				if _, err := writer.Write(body); err != nil {
					_ = writer.Close()
					return nil, err
				}
				if err := writer.Close(); err != nil {
					return nil, err
				}
				return compressed.Bytes(), nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/resource", DecompressRequestMiddleware(), func(c *gin.Context) {
				body, err := io.ReadAll(c.Request.Body)
				if err != nil {
					c.String(http.StatusBadRequest, "read body: %v", err)
					return
				}
				if got := c.GetHeader("Content-Encoding"); got != "" {
					c.String(http.StatusExpectationFailed, "content encoding was not removed: %q", got)
					return
				}
				c.Data(http.StatusOK, http.DetectContentType(body), body)
			})

			compressed, err := tt.encoder([]byte(payload))
			if err != nil {
				t.Fatalf("compress body: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/resource", bytes.NewReader(compressed))
			req.Header.Set("Content-Encoding", tt.header)
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			if recorder.Body.String() != payload {
				t.Fatalf("body = %q, want %q", recorder.Body.String(), payload)
			}
		})
	}
}

func TestDecompressRequestMiddlewareRejectsInvalidZstdBody(t *testing.T) {
	oldLimit := shared.MaxRequestBodyMB
	shared.MaxRequestBodyMB = 1
	t.Cleanup(func() {
		shared.MaxRequestBodyMB = oldLimit
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/resource", DecompressRequestMiddleware(), func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.String(http.StatusBadRequest, "read body: %v", err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/resource", bytes.NewReader([]byte("not-zstd")))
	req.Header.Set("Content-Encoding", "zstd")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
