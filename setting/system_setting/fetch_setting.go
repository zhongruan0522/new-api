package system_setting

import "one-api/setting/config"

type FetchSetting struct {
	EnableSSRFProtection bool     `json:"enable_ssrf_protection"` // 是否启用SSRF防护
	AllowPrivateIp       bool     `json:"allow_private_ip"`
	WhitelistDomains     []string `json:"whitelist_domains"` // domain format, e.g. example.com, *.example.com
	WhitelistIps         []string `json:"whitelist_ips"`     // CIDR format
	AllowedPorts         []string `json:"allowed_ports"`     // port range format, e.g. 80, 443, 8000-9000
}

var defaultFetchSetting = FetchSetting{
	EnableSSRFProtection: true, // 默认开启SSRF防护
	AllowPrivateIp:       false,
	WhitelistDomains:     []string{},
	WhitelistIps:         []string{},
	AllowedPorts:         []string{"80", "443", "8080", "8443"},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("fetch_setting", &defaultFetchSetting)
}

func GetFetchSetting() *FetchSetting {
	return &defaultFetchSetting
}
