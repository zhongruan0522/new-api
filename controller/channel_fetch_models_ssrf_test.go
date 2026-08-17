package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/setting/system_setting"

	"github.com/gin-gonic/gin"
)

var fetchModelsSSRFClientOnce sync.Once

func ensureFetchModelsSSRFHttpClient() {
	fetchModelsSSRFClientOnce.Do(service.InitHttpClient)
}

// overrideFetchModelsSSRFSetting 构造"初始 URL 合法、redirect 目标非法"的 SSRF 配置：
//   - 初始 URL 用 localhost（域名形式），ApplyIPFilterForDomain=false 使域名跳过
//     私有 IP 检查，仅走域名黑名单（空列表 = 全允许）；
//   - redirect 目标用 127.0.0.1（IP 形式），AllowPrivateIp=false 使其被私有 IP
//     规则拒绝；
//   - AllowedPorts 置空以允许 httptest 的随机端口。
func overrideFetchModelsSSRFSetting(t *testing.T) {
	t.Helper()

	setting := system_setting.GetFetchSetting()
	old := *setting
	t.Cleanup(func() {
		*setting = old
	})

	setting.EnableSSRFProtection = true
	setting.AllowPrivateIp = false
	setting.DomainFilterMode = false
	setting.IpFilterMode = false
	setting.DomainList = []string{}
	setting.IpList = []string{}
	setting.AllowedPorts = []string{}
	setting.ApplyIPFilterForDomain = false
}

// newFetchModelsRedirectServers 起一个 redirect 源（localhost 域名形式）和一个
// 被禁止的 redirect 目标（127.0.0.1 IP 形式）。源对任意请求 302 到目标。
func newFetchModelsRedirectServers(t *testing.T) (originURL string, blockedRedirectTarget string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"secret-model"}]}`))
	}))
	t.Cleanup(target.Close)

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(origin.Close)

	// httptest URL 是 127.0.0.1 形式；初始 URL 改写为 localhost 域名形式。
	originURL = strings.Replace(origin.URL, "127.0.0.1", "localhost", 1)
	return originURL, target.URL
}

// fetchModelsRedirectProbeURL 把 base_url 转成 FetchModels 内部构造的 /v1/models URL。
func fetchModelsRedirectProbeURL(baseURL string) string {
	return fmt.Sprintf("%s/v1/models", strings.TrimRight(baseURL, "/"))
}

// TestFetchModelsRedirectBlockedByControlledClient 证明修复生效：
// FetchModels 现在用 service.GetHttpClient()，其 CheckRedirect 会对 redirect
// 目标复查 SSRF 规则并拦截跳转。
func TestFetchModelsRedirectBlockedByControlledClient(t *testing.T) {
	overrideFetchModelsSSRFSetting(t)
	ensureFetchModelsSSRFHttpClient()

	originURL, blockedRedirectTarget := newFetchModelsRedirectServers(t)
	probeURL := fetchModelsRedirectProbeURL(originURL)

	// 前置断言 1：初始 URL 必须通过现有校验，否则请求到不了 redirect 环节。
	if err := validateFetchModelsURL(probeURL); err != nil {
		t.Fatalf("initial URL should pass validation, got: %v", err)
	}

	// 前置断言 2：redirect 目标必须被 SSRF 规则拒绝。
	if err := validateFetchModelsURL(blockedRedirectTarget); err == nil {
		t.Fatalf("redirect target should be rejected by SSRF rules, but passed: %s", blockedRedirectTarget)
	}

	// 核心断言：受控 client 在跟随 redirect 时被 CheckRedirect 拦截。
	req, err := http.NewRequest(http.MethodGet, probeURL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := service.GetHttpClient().Do(req)
	if err == nil {
		defer resp.Body.Close()
		t.Fatalf("expected redirect to be blocked by CheckRedirect, got status %d", resp.StatusCode)
	}
	if !strings.Contains(err.Error(), "redirect to") || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected SSRF redirect block error, got: %v", err)
	}
}

// TestFetchModelsBareClientFollowsRedirectUnblocked 固定"问题存在"的证据：
// 修复前的裸 &http.Client{} 无 CheckRedirect，同一场景下 redirect 不被拦截，
// 攻击者可达被 SSRF 规则禁止的目标。若有人把受控 client 换回裸 client，
// 该用例与上一用例的组合仍能暴露回归。
func TestFetchModelsBareClientFollowsRedirectUnblocked(t *testing.T) {
	overrideFetchModelsSSRFSetting(t)

	originURL, blockedRedirectTarget := newFetchModelsRedirectServers(t)
	probeURL := fetchModelsRedirectProbeURL(originURL)

	// 场景自检：redirect 目标本身在 SSRF 黑名单内。
	if err := validateFetchModelsURL(blockedRedirectTarget); err == nil {
		t.Fatalf("redirect target should be rejected by SSRF rules, but passed: %s", blockedRedirectTarget)
	}

	req, err := http.NewRequest(http.MethodGet, probeURL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("bare client should follow redirect without SSRF recheck: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bare client should reach blocked redirect target, got status %d", resp.StatusCode)
	}
}
