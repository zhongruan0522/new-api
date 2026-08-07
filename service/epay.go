package service

import (
	"github.com/NookMux/NookMux/setting/operation_setting"
	"github.com/NookMux/NookMux/setting/system_setting"
)

func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return system_setting.ServerAddress
	}
	return operation_setting.CustomCallbackAddress
}
