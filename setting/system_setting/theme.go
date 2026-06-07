package system_setting

import (
	"github.com/zhongruan0522/new-api/setting/config"
)

type ThemeSettings struct {
	Frontend string `json:"frontend"`
}

var themeSettings = ThemeSettings{
	Frontend: "default",
}

func init() {
	config.GlobalConfig.Register("theme", &themeSettings)
}

func GetThemeSettings() *ThemeSettings {
	return &themeSettings
}

func UpdateAndSyncTheme() {
	// Theme is now fixed to "default"; kept for config registry compatibility.
}
