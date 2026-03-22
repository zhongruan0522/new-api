package service

import (
	"github.com/zhongruan0522/new-api/setting/operation_setting"
	"github.com/zhongruan0522/new-api/setting/system_setting"
)

func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return system_setting.ServerAddress
	}
	return operation_setting.CustomCallbackAddress
}
