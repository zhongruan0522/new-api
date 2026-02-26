package common

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

const trustedProxiesEnvKey = "TRUSTED_PROXIES"

var cloudflareTrustedProxyCIDRs = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

var privateNetworkTrustedProxyCIDRs = []string{
	"127.0.0.1",
	"::1",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"100.64.0.0/10",  // CGNAT (commonly used in container/k8s networking)
	"169.254.0.0/16", // IPv4 link-local
	"fc00::/7",       // IPv6 ULA
	"fe80::/10",      // IPv6 link-local
}

func SetupGinTrustedProxies(engine *gin.Engine) {
	if engine == nil {
		return
	}

	// Ensure Gin can extract the real client IP from common reverse-proxy / CDN headers.
	// Order matters: prefer Cloudflare headers first when present.
	engine.ForwardedByClientIP = true
	engine.RemoteIPHeaders = []string{
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Forwarded-For",
		"X-Real-IP",
	}

	env := strings.TrimSpace(os.Getenv(trustedProxiesEnvKey))
	proxies, desc := buildTrustedProxiesFromEnv(env)

	if proxies == nil {
		if err := engine.SetTrustedProxies(nil); err != nil {
			SysError(fmt.Sprintf("failed to disable trusted proxies: %v", err))
			return
		}
		SysLog("trusted proxies disabled (TRUSTED_PROXIES=none)")
		return
	}

	if err := engine.SetTrustedProxies(proxies); err != nil {
		SysError(fmt.Sprintf("failed to set trusted proxies (%s): %v", desc, err))
		return
	}
	SysLog(fmt.Sprintf("trusted proxies configured (%s)", desc))
}

func buildTrustedProxiesFromEnv(env string) ([]string, string) {
	if env == "" {
		return defaultTrustedProxies(), "default=private,cloudflare"
	}

	switch strings.ToLower(env) {
	case "none", "disable", "off":
		return nil, "none"
	case "all", "unsafe":
		// Trust all proxies (NOT recommended). This matches Gin's historical default.
		return []string{"0.0.0.0/0", "::/0"}, "all (unsafe)"
	}

	tokens := splitAndTrim(env, ",")
	if len(tokens) == 0 {
		return defaultTrustedProxies(), "default=private,cloudflare (empty)"
	}

	proxies := make([]string, 0, len(tokens))
	descTokens := make([]string, 0, len(tokens))

	for _, token := range tokens {
		if token == "" {
			continue
		}
		switch strings.ToLower(token) {
		case "cloudflare", "cf":
			proxies = append(proxies, cloudflareTrustedProxyCIDRs...)
			descTokens = append(descTokens, "cloudflare")
		case "private", "local":
			proxies = append(proxies, privateNetworkTrustedProxyCIDRs...)
			descTokens = append(descTokens, "private")
		default:
			proxies = append(proxies, token)
			descTokens = append(descTokens, token)
		}
	}

	proxies = dedupeStrings(proxies)
	if len(proxies) == 0 {
		return defaultTrustedProxies(), "default=private,cloudflare (no valid entries)"
	}

	sort.Strings(proxies)
	return proxies, strings.Join(dedupeStrings(descTokens), ",")
}

func defaultTrustedProxies() []string {
	proxies := make([]string, 0, len(privateNetworkTrustedProxyCIDRs)+len(cloudflareTrustedProxyCIDRs))
	proxies = append(proxies, privateNetworkTrustedProxyCIDRs...)
	proxies = append(proxies, cloudflareTrustedProxyCIDRs...)
	proxies = dedupeStrings(proxies)
	sort.Strings(proxies)
	return proxies
}

func splitAndTrim(s string, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

