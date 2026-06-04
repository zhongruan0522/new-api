package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
)

func TestRelayErrorHandlerTruncatesInvalidJSONBodyInLog(t *testing.T) {
	withDebugEnabled(t, false)

	body := strings.Repeat("b", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	t.Cleanup(func() {
		gin.DefaultErrorWriter = oldWriter
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	if newAPIError == nil {
		t.Fatal("RelayErrorHandler returned nil error")
	}
	if got := newAPIError.Error(); got != "bad response status code 500" {
		t.Fatalf("error = %q, want bad response status code 500", got)
	}
	logged := logBuffer.String()
	if !strings.Contains(logged, "[truncated") {
		t.Fatalf("log did not include truncation marker: %s", logged)
	}
	if !strings.Contains(logged, fmt.Sprintf("original_length=%d", len(body))) {
		t.Fatalf("log did not include original length: %s", logged)
	}
	if strings.Contains(logged, strings.Repeat("b", common.LocalLogContentLimit+1)) {
		t.Fatal("log contained more than the allowed preview of the upstream body")
	}
}

func TestRelayErrorHandlerKeepsInvalidJSONBodyInDebugLog(t *testing.T) {
	withDebugEnabled(t, true)

	body := strings.Repeat("e", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	t.Cleanup(func() {
		gin.DefaultErrorWriter = oldWriter
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	if newAPIError == nil {
		t.Fatal("RelayErrorHandler returned nil error")
	}
	if strings.Contains(logBuffer.String(), "[truncated") {
		t.Fatalf("debug log unexpectedly truncated body: %s", logBuffer.String())
	}
	if !strings.Contains(logBuffer.String(), body) {
		t.Fatal("debug log did not include full upstream body")
	}
}

func TestRelayErrorHandlerKeepsStructuredErrorMessage(t *testing.T) {
	message := strings.Repeat("c", common.LocalLogContentLimit+256)
	body := `{"message":"` + message + `"}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	if newAPIError == nil {
		t.Fatal("RelayErrorHandler returned nil error")
	}
	if got := newAPIError.Error(); got != message {
		t.Fatalf("error = %q, want structured message", got)
	}
}

func withDebugEnabled(t *testing.T, enabled bool) {
	t.Helper()

	oldDebug := common.DebugEnabled
	common.DebugEnabled = enabled
	t.Cleanup(func() {
		common.DebugEnabled = oldDebug
	})
}
