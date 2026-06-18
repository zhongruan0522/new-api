package common

import (
	"fmt"
	"io"

	"github.com/zhongruan0522/new-api/constant"
)

// MaxResponseBodyExceededError is returned by ReadResponseBody when the
// upstream response body exceeds the configured MaxResponseBodyMB limit.
type MaxResponseBodyExceededError struct {
	LimitMB int
}

func (e *MaxResponseBodyExceededError) Error() string {
	return fmt.Sprintf("upstream response body exceeds %d MB limit", e.LimitMB)
}

// ReadResponseBody reads an upstream HTTP response body into memory with a
// size cap derived from constant.MaxResponseBodyMB.
//
// io.ReadAll has no upper bound, so a misbehaving or hostile upstream that
// returns an arbitrarily large body can exhaust server memory under
// concurrency. Every relay channel handler that loads non-streaming responses
// should use this instead of io.ReadAll.
func ReadResponseBody(r io.Reader) ([]byte, error) {
	maxMB := constant.MaxResponseBodyMB
	if maxMB <= 0 {
		maxMB = 128
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
