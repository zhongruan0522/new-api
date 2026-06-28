package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/constant"
)

func TestAnonymousRequestBodyLimitRejectsOversizedBody(t *testing.T) {
	oldLimit := constant.AnonymousRequestBodyLimitKB
	constant.AnonymousRequestBodyLimitKB = 1
	t.Cleanup(func() {
		constant.AnonymousRequestBodyLimitKB = oldLimit
	})

	router := gin.New()
	router.POST("/login", AnonymousRequestBodyLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(strings.Repeat("x", 1025)))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestAnonymousRequestBodyLimitPreservesAcceptedBody(t *testing.T) {
	oldLimit := constant.AnonymousRequestBodyLimitKB
	constant.AnonymousRequestBodyLimitKB = 1
	t.Cleanup(func() {
		constant.AnonymousRequestBodyLimitKB = oldLimit
	})

	router := gin.New()
	router.POST("/login", AnonymousRequestBodyLimit(), func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		c.String(http.StatusOK, string(body))
	})

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"alice"}`))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != `{"username":"alice"}` {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestAnonymousRequestBodyLimitCanBeDisabled(t *testing.T) {
	oldLimit := constant.AnonymousRequestBodyLimitKB
	constant.AnonymousRequestBodyLimitKB = 0
	t.Cleanup(func() {
		constant.AnonymousRequestBodyLimitKB = oldLimit
	})

	router := gin.New()
	router.POST("/webhook", AnonymousRequestBodyLimit(), func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		c.String(http.StatusOK, "%d", len(body))
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(strings.Repeat("x", 2048)))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "2048" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}
