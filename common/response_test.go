package common

import (
	"strings"
	"testing"

	"github.com/zhongruan0522/new-api/constant"
)

func TestTypedResponseBodyLimits(t *testing.T) {
	oldText := constant.MaxTextResponseBodyMB
	oldError := constant.MaxErrorResponseBodyMB
	oldMedia := constant.MaxMediaResponseBodyMB
	constant.MaxTextResponseBodyMB = 1
	constant.MaxErrorResponseBodyMB = 1
	constant.MaxMediaResponseBodyMB = 2
	t.Cleanup(func() {
		constant.MaxTextResponseBodyMB = oldText
		constant.MaxErrorResponseBodyMB = oldError
		constant.MaxMediaResponseBodyMB = oldMedia
	})

	body := strings.Repeat("x", (1<<20)+1)
	if _, err := ReadResponseBody(strings.NewReader(body)); err == nil {
		t.Fatal("ReadResponseBody accepted a body above the text limit")
	}
	if _, err := ReadErrorResponseBody(strings.NewReader(body)); err == nil {
		t.Fatal("ReadErrorResponseBody accepted a body above the error limit")
	}
	if _, err := ReadMediaResponseBody(strings.NewReader(body)); err != nil {
		t.Fatalf("ReadMediaResponseBody rejected a body below the media limit: %v", err)
	}
}
