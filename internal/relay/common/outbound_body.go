package common

import (
	"io"

	rootcommon "github.com/NookMux/NookMux/internal/common"
)

// NewOutboundJSONBody wraps an already-marshaled upstream request body in
// BodyStorage. Large JSON payloads can be disk-backed, which lets callers drop
// the original []byte before waiting on the upstream response.
func NewOutboundJSONBody(data []byte) (body io.Reader, size int64, closer io.Closer, err error) {
	storage, err := rootcommon.CreateBodyStorage(data)
	if err != nil {
		return nil, 0, nil, err
	}
	return rootcommon.ReaderOnly(storage), storage.Size(), storage, nil
}
