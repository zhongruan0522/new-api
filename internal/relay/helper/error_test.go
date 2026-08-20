package helper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/constant"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/gin-gonic/gin"
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

func TestRelayErrorHandlerCapsAndClosesOversizedBody(t *testing.T) {
	oldMaxErrorResponseBodyMB := constant.MaxErrorResponseBodyMB
	constant.MaxErrorResponseBodyMB = 1
	t.Cleanup(func() {
		constant.MaxErrorResponseBodyMB = oldMaxErrorResponseBodyMB
	})

	body := strings.NewReader(strings.Repeat("x", (1<<20)+1))
	tracker := &closeTrackingReadCloser{Reader: body}
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       tracker,
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	if newAPIError == nil {
		t.Fatal("RelayErrorHandler returned nil error")
	}
	if !tracker.closed {
		t.Fatal("oversized upstream error response body was not closed")
	}
	if !strings.Contains(newAPIError.Error(), "upstream response body exceeds 1 MB limit") {
		t.Fatalf("error = %q, want oversized body limit error", newAPIError.Error())
	}
}

func TestResetStatusCodeAcceptsNumericMapping(t *testing.T) {
	newAPIError := &shared.NookMuxError{StatusCode: http.StatusTooManyRequests}

	ResetStatusCode(newAPIError, `{"429":200}`)

	if newAPIError.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", newAPIError.StatusCode, http.StatusOK)
	}
	if newAPIError.OriginalStatusCode != http.StatusTooManyRequests {
		t.Fatalf("OriginalStatusCode = %d, want %d", newAPIError.OriginalStatusCode, http.StatusTooManyRequests)
	}
}

func TestResetStatusCodeKeepsStringMapping(t *testing.T) {
	newAPIError := &shared.NookMuxError{StatusCode: http.StatusTooManyRequests}

	ResetStatusCode(newAPIError, `{"429":"200"}`)

	if newAPIError.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", newAPIError.StatusCode, http.StatusOK)
	}
	if newAPIError.OriginalStatusCode != http.StatusTooManyRequests {
		t.Fatalf("OriginalStatusCode = %d, want %d", newAPIError.OriginalStatusCode, http.StatusTooManyRequests)
	}
}

func TestResetStatusCodeRejectsInvalidMappingValues(t *testing.T) {
	tests := []struct {
		name    string
		mapping string
	}{
		{name: "invalid string", mapping: `{"429":"not-a-status"}`},
		{name: "fractional number", mapping: `{"429":200.5}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newAPIError := &shared.NookMuxError{StatusCode: http.StatusTooManyRequests}

			ResetStatusCode(newAPIError, tt.mapping)

			if newAPIError.StatusCode != http.StatusTooManyRequests {
				t.Fatalf("StatusCode = %d, want unchanged %d", newAPIError.StatusCode, http.StatusTooManyRequests)
			}
			if newAPIError.OriginalStatusCode != 0 {
				t.Fatalf("OriginalStatusCode = %d, want 0 when mapping is rejected", newAPIError.OriginalStatusCode)
			}
		})
	}
}

func TestResetStatusCodeDoesNotRemapOK(t *testing.T) {
	newAPIError := &shared.NookMuxError{StatusCode: http.StatusOK}

	ResetStatusCode(newAPIError, `{"200":500}`)

	if newAPIError.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want unchanged %d", newAPIError.StatusCode, http.StatusOK)
	}
	if newAPIError.OriginalStatusCode != 0 {
		t.Fatalf("OriginalStatusCode = %d, want 0", newAPIError.OriginalStatusCode)
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

type closeTrackingReadCloser struct {
	*strings.Reader
	closed bool
}

func (r *closeTrackingReadCloser) Close() error {
	r.closed = true
	return nil
}
