package common

import (
	"fmt"
	"io"

	"github.com/NookMux/NookMux/constant"
)

// MaxResponseBodyExceededError is returned by ReadResponseBody when the
// upstream response body exceeds the configured MaxResponseBodyMB limit.
type MaxResponseBodyExceededError struct {
	LimitMB int
}

func (e *MaxResponseBodyExceededError) Error() string {
	return fmt.Sprintf("upstream response body exceeds %d MB limit", e.LimitMB)
}

// ReadResponseBody reads a regular upstream text/JSON response body into
// memory with a size cap.
//
// io.ReadAll has no upper bound, so a misbehaving or hostile upstream that
// returns an arbitrarily large body can exhaust server memory under
// concurrency. Relay channel handlers should use one of the typed helpers
// below instead of io.ReadAll so each business path has an explicit budget.
func ReadResponseBody(r io.Reader) ([]byte, error) {
	return readResponseBodyWithLimit(r, constant.MaxTextResponseBodyMB, 16)
}

func ReadEmbeddingResponseBody(r io.Reader) ([]byte, error) {
	return readResponseBodyWithLimit(r, constant.MaxEmbeddingResponseBodyMB, 64)
}

func ReadMediaResponseBody(r io.Reader) ([]byte, error) {
	return readResponseBodyWithLimit(r, constant.MaxMediaResponseBodyMB, 64)
}

func ReadErrorResponseBody(r io.Reader) ([]byte, error) {
	return readResponseBodyWithLimit(r, constant.MaxErrorResponseBodyMB, 4)
}

func ReadModelListResponseBody(r io.Reader) ([]byte, error) {
	return readResponseBodyWithLimit(r, constant.MaxModelListResponseBodyMB, 8)
}

func readResponseBodyWithLimit(r io.Reader, maxMB int, fallbackMB int) ([]byte, error) {
	if maxMB <= 0 {
		maxMB = fallbackMB
	}
	maxBytes := int64(maxMB) << 20

	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, &MaxResponseBodyExceededError{LimitMB: maxMB}
	}
	return data, nil
}
