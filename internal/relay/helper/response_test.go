package helper

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/shared"
)

func TestTypedResponseBodyLimits(t *testing.T) {
	oldText := shared.MaxTextResponseBodyMB
	oldError := shared.MaxErrorResponseBodyMB
	oldMedia := shared.MaxMediaResponseBodyMB
	shared.MaxTextResponseBodyMB = 1
	shared.MaxErrorResponseBodyMB = 1
	shared.MaxMediaResponseBodyMB = 2
	t.Cleanup(func() {
		shared.MaxTextResponseBodyMB = oldText
		shared.MaxErrorResponseBodyMB = oldError
		shared.MaxMediaResponseBodyMB = oldMedia
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
