package common

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// SSRFProtection SSRF防护配置
type SSRFProtection struct {
	AllowPrivateIp   bool
	WhitelistDomains []string // domain format, e.g. example.com, *.example.com
	WhitelistIps     []string // CIDR format
	AllowedPorts     []int    // 允许的端口范围
}

// DefaultSSRFProtection 默认SSRF防护配置
var DefaultSSRFProtection = &SSRFProtection{
	AllowPrivateIp:   false,
	WhitelistDomains: []string{},
	WhitelistIps:     []string{},
	AllowedPorts:     []int{},
}

// isPrivateIP 检查IP是否为私有地址
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// 检查私有网段
	private := []net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},     // 10.0.0.0/8
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},  // 172.16.0.0/12
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)}, // 192.168.0.0/16
		{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},    // 127.0.0.0/8
		{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}, // 169.254.0.0/16 (链路本地)
		{IP: net.IPv4(224, 0, 0, 0), Mask: net.CIDRMask(4, 32)},    // 224.0.0.0/4 (组播)
		{IP: net.IPv4(240, 0, 0, 0), Mask: net.CIDRMask(4, 32)},    // 240.0.0.0/4 (保留)
	}

	for _, privateNet := range private {
		if privateNet.Contains(ip) {
			return true
		}
	}

	// 检查IPv6私有地址
	if ip.To4() == nil {
		// IPv6 loopback
		if ip.Equal(net.IPv6loopback) {
			return true
		}
		// IPv6 link-local
		if strings.HasPrefix(ip.String(), "fe80:") {
			return true
		}
		// IPv6 unique local
		if strings.HasPrefix(ip.String(), "fc") || strings.HasPrefix(ip.String(), "fd") {
			return true
		}
	}

	return false
}

// parsePortRanges 解析端口范围配置
// 支持格式: "80", "443", "8000-9000"
func parsePortRanges(portConfigs []string) ([]int, error) {
	var ports []int

	for _, config := range portConfigs {
		config = strings.TrimSpace(config)
		if config == "" {
			continue
		}

		if strings.Contains(config, "-") {
			// 处理端口范围 "8000-9000"
			parts := strings.Split(config, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid port range format: %s", config)
			}

			startPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port in range %s: %v", config, err)
			}

			endPort, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port in range %s: %v", config, err)
			}

			if startPort > endPort {
				return nil, fmt.Errorf("invalid port range %s: start port cannot be greater than end port", config)
			}

			if startPort < 1 || startPort > 65535 || endPort < 1 || endPort > 65535 {
				return nil, fmt.Errorf("port range %s contains invalid port numbers (must be 1-65535)", config)
			}

			// 添加范围内的所有端口
			for port := startPort; port <= endPort; port++ {
				ports = append(ports, port)
			}
		} else {
			// 处理单个端口 "80"
			port, err := strconv.Atoi(config)
			if err != nil {
				return nil, fmt.Errorf("invalid port number: %s", config)
			}

			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("invalid port number %d (must be 1-65535)", port)
			}

			ports = append(ports, port)
		}
	}

	return ports, nil
}

// isAllowedPort 检查端口是否被允许
func (p *SSRFProtection) isAllowedPort(port int) bool {
	if len(p.AllowedPorts) == 0 {
		return true // 如果没有配置端口限制，则允许所有端口
	}

	for _, allowedPort := range p.AllowedPorts {
		if port == allowedPort {
			return true
		}
	}
	return false
}

// isAllowedPortFromRanges 从端口范围字符串检查端口是否被允许
func isAllowedPortFromRanges(port int, portRanges []string) bool {
	if len(portRanges) == 0 {
		return true // 如果没有配置端口限制，则允许所有端口
	}

	allowedPorts, err := parsePortRanges(portRanges)
	if err != nil {
		// 如果解析失败，为安全起见拒绝访问
		return false
	}

	for _, allowedPort := range allowedPorts {
		if port == allowedPort {
			return true
		}
	}
	return false
}

// isDomainWhitelisted 检查域名是否在白名单中
func (p *SSRFProtection) isDomainWhitelisted(domain string) bool {
	if len(p.WhitelistDomains) == 0 {
		return false
	}

	domain = strings.ToLower(domain)
	for _, whitelistDomain := range p.WhitelistDomains {
		whitelistDomain = strings.ToLower(whitelistDomain)

		// 精确匹配
		if domain == whitelistDomain {
			return true
		}

		// 通配符匹配 (*.example.com)
		if strings.HasPrefix(whitelistDomain, "*.") {
			suffix := strings.TrimPrefix(whitelistDomain, "*.")
			if strings.HasSuffix(domain, "."+suffix) || domain == suffix {
				return true
			}
		}
	}
	return false
}

// isIPWhitelisted 检查IP是否在白名单中
func (p *SSRFProtection) isIPWhitelisted(ip net.IP) bool {
	if len(p.WhitelistIps) == 0 {
		return false
	}

	for _, whitelistCIDR := range p.WhitelistIps {
		_, network, err := net.ParseCIDR(whitelistCIDR)
		if err != nil {
			// 尝试作为单个IP处理
			if whitelistIP := net.ParseIP(whitelistCIDR); whitelistIP != nil {
				if ip.Equal(whitelistIP) {
					return true
				}
			}
			continue
		}

		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// IsIPAccessAllowed 检查IP是否允许访问
func (p *SSRFProtection) IsIPAccessAllowed(ip net.IP) bool {
	// 如果IP在白名单中，直接允许访问（绕过私有IP检查）
	if p.isIPWhitelisted(ip) {
		return true
	}

	// 如果IP白名单为空，允许所有IP（但仍需通过私有IP检查）
	if len(p.WhitelistIps) == 0 {
		// 检查私有IP限制
		if isPrivateIP(ip) && !p.AllowPrivateIp {
			return false
		}
		return true
	}

	// 如果IP白名单不为空且IP不在白名单中，拒绝访问
	return false
}

// ValidateURL 验证URL是否安全
func (p *SSRFProtection) ValidateURL(urlStr string) error {
	// 解析URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// 只允许HTTP/HTTPS协议
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported protocol: %s (only http/https allowed)", u.Scheme)
	}

	// 解析主机和端口
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		// 没有端口，使用默认端口
		host = u.Host
		if u.Scheme == "https" {
			portStr = "443"
		} else {
			portStr = "80"
		}
	}

	// 验证端口
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %s", portStr)
	}

	if !p.isAllowedPort(port) {
		return fmt.Errorf("port %d is not allowed", port)
	}

	// 检查域名白名单
	if p.isDomainWhitelisted(host) {
		return nil // 白名单域名直接通过
	}

	// DNS解析获取IP地址
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %v", host, err)
	}

	// 检查所有解析的IP地址
	for _, ip := range ips {
		if !p.IsIPAccessAllowed(ip) {
			if isPrivateIP(ip) {
				return fmt.Errorf("private IP address not allowed: %s resolves to %s", host, ip.String())
			} else {
				return fmt.Errorf("IP address not in whitelist: %s resolves to %s", host, ip.String())
			}
		}
	}

	return nil
}

// ValidateURLWithDefaults 使用默认配置验证URL
func ValidateURLWithDefaults(urlStr string) error {
	return DefaultSSRFProtection.ValidateURL(urlStr)
}

// ValidateURLWithFetchSetting 使用FetchSetting配置验证URL
func ValidateURLWithFetchSetting(urlStr string, enableSSRFProtection, allowPrivateIp bool, whitelistDomains, whitelistIps, allowedPorts []string) error {
	// 如果SSRF防护被禁用，直接返回成功
	if !enableSSRFProtection {
		return nil
	}

	// 解析端口范围配置
	allowedPortInts, err := parsePortRanges(allowedPorts)
	if err != nil {
		return fmt.Errorf("request reject - invalid port configuration: %v", err)
	}

	protection := &SSRFProtection{
		AllowPrivateIp:   allowPrivateIp,
		WhitelistDomains: whitelistDomains,
		WhitelistIps:     whitelistIps,
		AllowedPorts:     allowedPortInts,
	}
	return protection.ValidateURL(urlStr)
}

// ValidateURLWithPortRanges 直接使用端口范围字符串验证URL（更高效的版本）
func ValidateURLWithPortRanges(urlStr string, allowPrivateIp bool, whitelistDomains, whitelistIps, allowedPorts []string) error {
	// 解析URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// 只允许HTTP/HTTPS协议
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported protocol: %s (only http/https allowed)", u.Scheme)
	}

	// 解析主机和端口
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		// 没有端口，使用默认端口
		host = u.Host
		if u.Scheme == "https" {
			portStr = "443"
		} else {
			portStr = "80"
		}
	}

	// 验证端口
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %s", portStr)
	}

	if !isAllowedPortFromRanges(port, allowedPorts) {
		return fmt.Errorf("port %d is not allowed", port)
	}

	// 创建临时的SSRFProtection来复用域名和IP检查逻辑
	protection := &SSRFProtection{
		AllowPrivateIp:   allowPrivateIp,
		WhitelistDomains: whitelistDomains,
		WhitelistIps:     whitelistIps,
	}

	// 检查域名白名单
	if protection.isDomainWhitelisted(host) {
		return nil // 白名单域名直接通过
	}

	// DNS解析获取IP地址
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %v", host, err)
	}

	// 检查所有解析的IP地址
	for _, ip := range ips {
		if !protection.IsIPAccessAllowed(ip) {
			if isPrivateIP(ip) {
				return fmt.Errorf("private IP address not allowed: %s resolves to %s", host, ip.String())
			} else {
				return fmt.Errorf("IP address not in whitelist: %s resolves to %s", host, ip.String())
			}
		}
	}

	return nil
}
