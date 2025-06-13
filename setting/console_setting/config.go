package console_setting

import "one-api/setting/config"

type ConsoleSetting struct {
    ApiInfo        string `json:"api_info"`         // 控制台 API 信息 (JSON 数组字符串)
    UptimeKumaUrl  string `json:"uptime_kuma_url"`  // Uptime Kuma 服务地址（如 https://status.example.com ）
    UptimeKumaSlug string `json:"uptime_kuma_slug"` // Uptime Kuma Status Page Slug
    Announcements  string `json:"announcements"`    // 系统公告 (JSON 数组字符串)
    FAQ            string `json:"faq"`              // 常见问题 (JSON 数组字符串)
}

// 默认配置
var defaultConsoleSetting = ConsoleSetting{
    ApiInfo:        "",
    UptimeKumaUrl:  "",
    UptimeKumaSlug: "",
    Announcements:  "",
    FAQ:            "",
}

// 全局实例
var consoleSetting = defaultConsoleSetting

func init() {
    // 注册到全局配置管理器，键名为 console_setting
    config.GlobalConfig.Register("console_setting", &consoleSetting)
}

// GetConsoleSetting 获取 ConsoleSetting 配置实例
func GetConsoleSetting() *ConsoleSetting {
    return &consoleSetting
} 