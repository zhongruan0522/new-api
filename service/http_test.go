package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
)

func TestShouldCopyUpstreamHeaderSkipsLocalRequestIdAndCapturesUpstreamId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if ShouldCopyUpstreamHeader(c, common.RequestIdKey, []string{"upstream-id"}) {
		t.Fatalf("expected %s to be skipped", common.RequestIdKey)
	}
	if got := c.GetString(common.UpstreamRequestIdKey); got != "upstream-id" {
		t.Fatalf("captured upstream request id = %q, want upstream-id", got)
	}
	if !ShouldCopyUpstreamHeader(c, "Content-Type", []string{"application/json"}) {
		t.Fatal("expected Content-Type to be copied")
	}
	if ShouldCopyUpstreamHeader(c, "Content-Length", []string{"123"}) {
		t.Fatal("expected Content-Length to be skipped")
	}
}

func TestIOCopyBytesGracefullyDoesNotOverrideLocalRequestId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	recorder.Header().Set(common.RequestIdKey, "local-id")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			common.RequestIdKey: []string{"upstream-id"},
			"Content-Type":      []string{"application/json"},
		},
	}
	IOCopyBytesGracefully(c, resp, []byte(`{"ok":true}`))

	if got := recorder.Header().Get(common.RequestIdKey); got != "local-id" {
		t.Fatalf("response request id = %q, want local-id", got)
	}
	if got := c.GetString(common.UpstreamRequestIdKey); got != "upstream-id" {
		t.Fatalf("captured upstream request id = %q, want upstream-id", got)
	}
}
