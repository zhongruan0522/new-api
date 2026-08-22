package httpapi

import (
	"github.com/NookMux/NookMux/internal/domain/shared"
)

const defaultAnonymousRequestBodyLimitKB = 512

func GetAnonymousRequestBodyLimitBytes() int64 {
	limitKB := shared.AnonymousRequestBodyLimitKB
	if limitKB < 0 {
		limitKB = defaultAnonymousRequestBodyLimitKB
	}
	return int64(limitKB) << 10
}
