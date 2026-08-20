package channelcontroller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/NookMux/NookMux/internal/config/system"
	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/pkg/jsonx"

	httpclient "github.com/NookMux/NookMux/internal/infra/httpclient"
	"github.com/gin-gonic/gin"
)

var fetchModelsSSRFClientOnce sync.Once

func ensureFetchModelsSSRFHttpClient() {
	fetchModelsSSRFClientOnce.Do(httpclient.InitHttpClient)
}

// overrideFetchModelsSSRFSetting 构造"初始 URL 合法、redirect 目标非法"的 SSRF 配置：
//   - 初始 URL 用 localhost（域名形式），ApplyIPFilterForDomain=false 使域名跳过
//     私有 IP 检查，仅走域名黑名单（空列表 = 全允许）；
//   - redirect 目标用 127.0.0.1（IP 形式），AllowPrivateIp=false 使其被私有 IP
//     规则拒绝；
//   - AllowedPorts 置空以允许 httptest 的随机端口。
func overrideFetchModelsSSRFSetting(t *testing.T) {
	t.Helper()

	setting := system.GetFetchSetting()
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
// FetchModels 用 httpclient.GetHttpClient()，其 CheckRedirect 会对 redirect
// 目标复查 SSRF 规则并拦截跳转。测试通过 gin 上下文实际调用 FetchModels
// 处理器，覆盖完整链路（初始校验 → 请求 → redirect 复查）；若生产代码
// 换回裸 http.Client，本用例会因跳转未被拦截而失败。
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

	// 核心断言：实际调用 FetchModels，受控 client 拦截 redirect 后返回 500。
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := fmt.Sprintf(`{"base_url":%q,"type":%d,"key":"sk-test"}`, originURL, constant.ChannelTypeOpenAI)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/channel/fetch_models", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	FetchModels(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from FetchModels when redirect is blocked, got %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := jsonx.DecodeJson(bytes.NewReader(w.Body.Bytes()), &resp); err != nil {
		t.Fatalf("decode response: %v, body: %s", err, w.Body.String())
	}
	if resp.Success {
		t.Fatalf("expected success=false, body: %s", w.Body.String())
	}
	if !strings.Contains(resp.Message, "redirect to") || !strings.Contains(resp.Message, "blocked") {
		t.Fatalf("expected SSRF redirect block error message, got: %s", resp.Message)
	}
}

// TestFetchModelsBareClientRegression 固定回归证据：同一场景下裸
// &http.Client{} 会跟随 redirect 到达被禁止的目标。这证明上一用例失败时
// （即 FetchModels 被改回裸 client）暴露的是真实攻击路径，而非误报。
func TestFetchModelsBareClientRegression(t *testing.T) {
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
