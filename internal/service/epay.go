package service

import (
	"github.com/NookMux/NookMux/internal/config/operation"
	"github.com/NookMux/NookMux/internal/config/system"
)

func GetCallbackAddress() string {
	if operation.CustomCallbackAddress == "" {
		return system.ServerAddress
	}
	return operation.CustomCallbackAddress
}
