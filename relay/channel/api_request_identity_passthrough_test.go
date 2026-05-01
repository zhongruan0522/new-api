package channel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"sync"

	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/types"

	"github.com/gin-gonic/gin"
)

var initHTTPClientOnce sync.Once

func ensureHTTPClient() {
	initHTTPClientOnce.Do(service.InitHttpClient)
}

const upstreamRequestWaitTimeout = 2 * time.Second

type testNoopAdaptor struct {
	url string
}

func (a *testNoopAdaptor) Init(info *relaycommon.RelayInfo) {}

func (a *testNoopAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return a.url, nil
}

func (a *testNoopAdaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	return nil
}

func (a *testNoopAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	panic("unexpected call: ConvertOpenAIRequest")
}

func (a *testNoopAdaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	panic("unexpected call: ConvertRerankRequest")
}

func (a *testNoopAdaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	panic("unexpected call: ConvertEmbeddingRequest")
}

func (a *testNoopAdaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	panic("unexpected call: ConvertAudioRequest")
}

func (a *testNoopAdaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	panic("unexpected call: ConvertImageRequest")
}

func (a *testNoopAdaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	panic("unexpected call: ConvertOpenAIResponsesRequest")
}

func (a *testNoopAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	panic("unexpected call: DoRequest")
}

func (a *testNoopAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	panic("unexpected call: DoResponse")
}

func (a *testNoopAdaptor) GetModelList() []string {
	return nil
}

func (a *testNoopAdaptor) GetChannelName() string {
	return "test_noop"
}

func (a *testNoopAdaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	panic("unexpected call: ConvertClaudeRequest")
}

func (a *testNoopAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	panic("unexpected call: ConvertGeminiRequest")
}

func newTestGinContext(t *testing.T, headers map[string]string) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/chat/completions", strings.NewReader("ok"))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

func newTestRelayInfo(passThrough bool, headersOverride map[string]interface{}) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{
				PassThroughHeadersEnabled: passThrough,
			},
			HeadersOverride: headersOverride,
		},
	}
}

func defaultClientHeaders() map[string]string {
	return map[string]string{
		"User-Agent":    "ClientUA",
		"X-Title":       "ClientTitle",
		"HTTP-Referer":  "https://client.example",
		"Referer":       "https://client.example/ref",
		"X-Other-Alive": "ok",
	}
}

func doApiRequestToTestServer(t *testing.T, info *relaycommon.RelayInfo, clientHeaders map[string]string) http.Header {
	t.Helper()
	ensureHTTPClient()

	gotCh := make(chan http.Header, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCh <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	c := newTestGinContext(t, clientHeaders)
	adaptor := &testNoopAdaptor{url: upstream.URL}
	resp, err := DoApiRequest(adaptor, c, info, strings.NewReader("body"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = resp.Body.Close()

	select {
	case got := <-gotCh:
		return got
	case <-time.After(upstreamRequestWaitTimeout):
		t.Fatalf("timeout waiting for upstream request")
		return nil
	}
}

func TestDoApiRequest_PassthroughIncludesIdentityHeaders(t *testing.T) {
	info := newTestRelayInfo(true, map[string]interface{}{})
	got := doApiRequestToTestServer(t, info, defaultClientHeaders())

	if got.Get("X-Other-Alive") != "ok" {
		t.Fatalf("expected X-Other-Alive to be passthrough, got %q", got.Get("X-Other-Alive"))
	}
	if got.Get("User-Agent") != "ClientUA" {
		t.Fatalf("expected User-Agent to be passthrough, got %q", got.Get("User-Agent"))
	}
	if got.Get("X-Title") != "ClientTitle" {
		t.Fatalf("expected X-Title to be passthrough, got %q", got.Get("X-Title"))
	}
	if got.Get("HTTP-Referer") != "https://client.example" {
		t.Fatalf("expected HTTP-Referer to be passthrough, got %q", got.Get("HTTP-Referer"))
	}
	if got.Get("Referer") != "https://client.example/ref" {
		t.Fatalf("expected Referer to be passthrough, got %q", got.Get("Referer"))
	}
}

func TestDoApiRequest_AllowsIdentityHeadersViaExplicitClientHeaderOverrideEvenWhenPassthroughDisabled(t *testing.T) {
	info := newTestRelayInfo(false, map[string]interface{}{
		"User-Agent":   "{client_header:User-Agent}",
		"X-Title":      "{client_header:X-Title}",
		"HTTP-Referer": "{client_header:HTTP-Referer}",
		"Referer":      "{client_header:Referer}",
	})
	got := doApiRequestToTestServer(t, info, defaultClientHeaders())

	if got.Get("X-Other-Alive") != "" {
		t.Fatalf("expected X-Other-Alive to not passthrough, got %q", got.Get("X-Other-Alive"))
	}
	if got.Get("User-Agent") != "ClientUA" {
		t.Fatalf("expected User-Agent from explicit override, got %q", got.Get("User-Agent"))
	}
	if got.Get("X-Title") != "ClientTitle" {
		t.Fatalf("expected X-Title from explicit override, got %q", got.Get("X-Title"))
	}
	if got.Get("HTTP-Referer") != "https://client.example" {
		t.Fatalf("expected HTTP-Referer from explicit override, got %q", got.Get("HTTP-Referer"))
	}
	if got.Get("Referer") != "https://client.example/ref" {
		t.Fatalf("expected Referer from explicit override, got %q", got.Get("Referer"))
	}
}

func TestDoApiRequest_ExplicitCookieInHeaderOverride(t *testing.T) {
	info := newTestRelayInfo(false, map[string]interface{}{
		"Cookie": "session=abc123; token=xyz",
	})
	got := doApiRequestToTestServer(t, info, nil)

	if got.Get("Cookie") != "session=abc123; token=xyz" {
		t.Fatalf("expected Cookie from explicit override, got %q", got.Get("Cookie"))
	}
}

func TestDoApiRequest_CookieNotPassthroughByWildcard(t *testing.T) {
	info := newTestRelayInfo(true, map[string]interface{}{
		"*": true,
	})
	clientHeaders := map[string]string{
		"Cookie":    "client_session=should_not_pass",
		"X-Custom":  "should_pass",
	}
	got := doApiRequestToTestServer(t, info, clientHeaders)

	if got.Get("Cookie") != "" {
		t.Fatalf("expected Cookie to NOT be passthrough by wildcard, got %q", got.Get("Cookie"))
	}
	if got.Get("X-Custom") != "should_pass" {
		t.Fatalf("expected X-Custom to be passthrough, got %q", got.Get("X-Custom"))
	}
}

func TestDoApiRequest_ExplicitCookieOverridesPassthrough(t *testing.T) {
	info := newTestRelayInfo(true, map[string]interface{}{
		"*":       true,
		"Cookie":  "explicit_cookie=value",
	})
	clientHeaders := map[string]string{
		"Cookie":   "client_cookie=should_not_pass",
		"X-Custom": "should_pass",
	}
	got := doApiRequestToTestServer(t, info, clientHeaders)

	if got.Get("Cookie") != "explicit_cookie=value" {
		t.Fatalf("expected explicit Cookie override to win, got %q", got.Get("Cookie"))
	}
	if got.Get("X-Custom") != "should_pass" {
		t.Fatalf("expected X-Custom to be passthrough, got %q", got.Get("X-Custom"))
	}
}
