package channelcontroller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/system"
	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/store/channel"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupFetchProvidersTestDB 为 fetch_providers 端点测试准备内存 SQLite，
// 关闭内存缓存以免触发全局渠道缓存同步。
func setupFetchProvidersTestDB(t *testing.T) {
	t.Helper()

	oldDB := dbstore.DB
	oldMemoryCacheEnabled := common.MemoryCacheEnabled

	common.MemoryCacheEnabled = false

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&channelstore.Channel{}, &channelstore.Ability{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	dbstore.DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		dbstore.DB = oldDB
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func newFetchProvidersUpstream(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"data":[
			{"slug":"cerebras","name":"Cerebras","headquarters":"US","privacy_policy_url":"https://cerebras.ai/privacy"},
			{"slug":"deepinfra","name":"DeepInfra","headquarters":"US"},
			{"slug":"","name":"Empty Slug"},
			{"slug":"  openai  ","name":"OpenAI"}
		]}`))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func decodeFetchProvidersResponse(t *testing.T, body string) (bool, []OpenRouterProviderEntry) {
	t.Helper()
	var resp struct {
		Success bool                      `json:"success"`
		Data    []OpenRouterProviderEntry `json:"data"`
	}
	if err := jsonx.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode response: %v, body: %s", err, body)
	}
	return resp.Success, resp.Data
}

func TestFetchUpstreamProvidersReturnsProviderList(t *testing.T) {
	setupFetchProvidersTestDB(t)

	upstreamURL := newFetchProvidersUpstream(t)
	baseURL := upstreamURL
	row := &channelstore.Channel{
		Name:    "fetch-providers-openrouter",
		Type:    constant.ChannelTypeOpenRouter,
		Status:  common.ChannelStatusEnabled,
		Key:     "sk-test",
		BaseURL: &baseURL,
		Models:  "openai/gpt-4o",
		Group:   "default",
	}
	if err := dbstore.DB.Create(row).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", row.Id)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/channel/fetch_providers/"+fmt.Sprintf("%d", row.Id), nil)

	FetchUpstreamProviders(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	success, providers := decodeFetchProvidersResponse(t, w.Body.String())
	if !success {
		t.Fatalf("expected success=true, body: %s", w.Body.String())
	}
	// 空_slug 条目被丢弃，其余条目保留且 slug 被规整。
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers after dropping empty slug, got %d: %+v", len(providers), providers)
	}
	if providers[0].Slug != "cerebras" || providers[0].Name != "Cerebras" || providers[0].Headquarters != "US" {
		t.Fatalf("unexpected first provider: %+v", providers[0])
	}
	if providers[2].Slug != "openai" {
		t.Fatalf("expected trimmed slug, got %q", providers[2].Slug)
	}
}

func TestFetchUpstreamProvidersRejectsNonOpenRouterChannel(t *testing.T) {
	setupFetchProvidersTestDB(t)

	row := &channelstore.Channel{
		Name:   "fetch-providers-openai",
		Type:   constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusEnabled,
		Key:    "sk-test",
		Models: "gpt-4o",
		Group:  "default",
	}
	if err := dbstore.DB.Create(row).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", row.Id)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/channel/fetch_providers/"+fmt.Sprintf("%d", row.Id), nil)

	FetchUpstreamProviders(c)

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := jsonx.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body: %s", err, w.Body.String())
	}
	if resp.Success || resp.Message == "" {
		t.Fatalf("expected openrouter-only rejection with message, got success=%v message=%q", resp.Success, resp.Message)
	}
}

// overrideFetchProvidersSSRFSetting 放行 httptest 的 127.0.0.1 随机端口，
// 让 POST 版 fetch_providers 的 URL 校验可以通过（保持 SSRF 校验链路开启）。
func overrideFetchProvidersSSRFSetting(t *testing.T) {
	t.Helper()

	setting := system.GetFetchSetting()
	old := *setting
	t.Cleanup(func() {
		*setting = old
	})

	setting.EnableSSRFProtection = true
	setting.AllowPrivateIp = true
	setting.DomainFilterMode = false
	setting.IpFilterMode = false
	setting.DomainList = []string{}
	setting.IpList = []string{}
	setting.AllowedPorts = []string{}
	setting.ApplyIPFilterForDomain = false
}

func TestFetchProvidersPostReturnsProviderList(t *testing.T) {
	overrideFetchProvidersSSRFSetting(t)
	ensureFetchModelsSSRFHttpClient()
	upstreamURL := newFetchProvidersUpstream(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := fmt.Sprintf(`{"base_url":%q,"type":%d}`, upstreamURL, constant.ChannelTypeOpenRouter)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/channel/fetch_providers", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	FetchProviders(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	success, providers := decodeFetchProvidersResponse(t, w.Body.String())
	if !success || len(providers) != 3 {
		t.Fatalf("expected success with 3 providers, got success=%v count=%d body=%s", success, len(providers), w.Body.String())
	}
}

func TestFetchProvidersPostRejectsNonOpenRouterType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/channel/fetch_providers", strings.NewReader(`{"type":1}`))
	c.Request.Header.Set("Content-Type", "application/json")

	FetchProviders(c)

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := jsonx.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body: %s", err, w.Body.String())
	}
	if resp.Success || resp.Message == "" {
		t.Fatalf("expected openrouter-only rejection with message, got success=%v message=%q", resp.Success, resp.Message)
	}
}
