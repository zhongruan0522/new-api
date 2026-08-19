package model

import (
	"github.com/NookMux/NookMux/internal/config/manager"
)

type GlobalSettings struct {
}

// 默认配置
var defaultOpenaiSettings = GlobalSettings{}

// 全局实例
var globalSettings = defaultOpenaiSettings

func init() {
	// 注册到全局配置管理器
	manager.GlobalConfig.Register("global", &globalSettings)
}

func GetGlobalSettings() *GlobalSettings {
	return &globalSettings
}
