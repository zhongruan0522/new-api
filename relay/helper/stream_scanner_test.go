package helper

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/constant"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func TestStreamScannerHandlerTrimsAndSkipsEmptyDataFrames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"data:   ",
			"data:\r",
			"data:  {\"ok\":true}\r",
			"data: [DONE]",
			"",
		}, "\n"))),
	}

	var got []string
	StreamScannerHandler(c, resp, &relaycommon.RelayInfo{}, func(data string) bool {
		got = append(got, data)
		return true
	})

	want := []string{`{"ok":true}`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("data frames = %#v, want %#v", got, want)
	}
}
