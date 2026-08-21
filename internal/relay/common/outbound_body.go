package common

import (
	"io"

	"github.com/NookMux/NookMux/internal/infra/cache"
)

// NewOutboundJSONBody wraps an already-marshaled upstream request body in
// BodyStorage. Large JSON payloads can be disk-backed, which lets callers drop
// the original []byte before waiting on the upstream response.
func NewOutboundJSONBody(data []byte) (body io.Reader, size int64, closer io.Closer, err error) {
	storage, err := cache.CreateBodyStorage(data)
	if err != nil {
		return nil, 0, nil, err
	}
	return cache.ReaderOnly(storage), storage.Size(), storage, nil
}
