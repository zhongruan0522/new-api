package model_setting

import (
	"github.com/zhongruan0522/new-api/setting/config"
)

type GlobalSettings struct {
}

// 默认配置
var defaultOpenaiSettings = GlobalSettings{}

// 全局实例
var globalSettings = defaultOpenaiSettings

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("global", &globalSettings)
}

func GetGlobalSettings() *GlobalSettings {
	return &globalSettings
}
