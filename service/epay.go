package service

import (
	"one-api/setting"
	"one-api/setting/operation_setting"
)

func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return setting.ServerAddress
	}
	return operation_setting.CustomCallbackAddress
}
