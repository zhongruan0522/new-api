package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	channelconstant "github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/gin-gonic/gin"
)

func newOpenRouterClaudeInfo(baseURL string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayFormat:       relayconstant.RelayFormatClaude,
		ResponseModelName: "claude-sonnet-4",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       channelconstant.ChannelTypeOpenRouter,
			ChannelBaseUrl:    baseURL,
			ApiKey:            "sk-or-test",
			UpstreamModelName: "anthropic/claude-sonnet-4",
		},
	}
}

// OpenRouter 原生提供 Anthropic Messages 兼容端点，Claude 请求（含 thinking 等
// 原生字段）必须原样透传，不得转换为 OpenAI Chat 格式。
func TestConvertClaudeRequestOpenRouterNativePassthrough(t *testing.T) {
	budgetTokens := 2048
	request := &shared.ClaudeRequest{
		Model:     "anthropic/claude-sonnet-4",
		MaxTokens: 1024,
		Messages: []shared.ClaudeMessage{
			{Role: "user", Content: "你好"},
		},
		Thinking: &shared.Thinking{
			Type:         "enabled",
			BudgetTokens: &budgetTokens,
		},
	}
	info := newOpenRouterClaudeInfo("https://openrouter.ai/api")

	converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertClaudeRequest error = %v", err)
	}
	if converted != any(request) {
		t.Fatalf("ConvertClaudeRequest 应原样返回同一个 Claude 请求（原生透传），got %T", converted)
	}
}

// Claude 请求在 OpenRouter 渠道直达原生 /v1/messages 端点。
func TestGetRequestURLOpenRouterClaudeNative(t *testing.T) {
	a := &Adaptor{}

	info := newOpenRouterClaudeInfo("")
	url, err := a.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if url != "https://openrouter.ai/api/v1/messages" {
		t.Fatalf("默认 base URL 兜底: url = %q, want https://openrouter.ai/api/v1/messages", url)
	}

	info = newOpenRouterClaudeInfo("https://example.com/api/")
	url, err = a.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if url != "https://example.com/api/v1/messages" {
		t.Fatalf("自定义 base URL: url = %q, want https://example.com/api/v1/messages", url)
	}
}

// Responses / Chat 格式维持既有 URL 行为（原生直达，无回归）。
func TestGetRequestURLOpenRouterResponsesAndChatUnchanged(t *testing.T) {
	a := &Adaptor{}

	responsesInfo := &relaycommon.RelayInfo{
		RelayFormat:    relayconstant.RelayFormatOpenAIResponses,
		RelayMode:      relayconstant.RelayModeResponses,
		RequestURLPath: "/v1/responses",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       channelconstant.ChannelTypeOpenRouter,
			ChannelBaseUrl:    "https://openrouter.ai/api",
			UpstreamModelName: "openai/gpt-5",
		},
	}
	url, err := a.GetRequestURL(responsesInfo)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if url != "https://openrouter.ai/api/v1/responses" {
		t.Fatalf("responses url = %q, want https://openrouter.ai/api/v1/responses", url)
	}

	chatInfo := &relaycommon.RelayInfo{
		RelayFormat:    relayconstant.RelayFormatOpenAI,
		RelayMode:      relayconstant.RelayModeChatCompletions,
		RequestURLPath: "/v1/chat/completions",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       channelconstant.ChannelTypeOpenRouter,
			ChannelBaseUrl:    "https://openrouter.ai/api",
			UpstreamModelName: "openai/gpt-5",
		},
	}
	url, err = a.GetRequestURL(chatInfo)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if url != "https://openrouter.ai/api/v1/chat/completions" {
		t.Fatalf("chat url = %q, want https://openrouter.ai/api/v1/chat/completions", url)
	}
}

// 回归保护：OpenRouter 无 Gemini 原生端点，Gemini 格式仍转换为 chat/completions。
func TestGetRequestURLOpenRouterGeminiStillConverted(t *testing.T) {
	a := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayFormat:    relayconstant.RelayFormatGemini,
		RequestURLPath: "/v1beta/models/gemini-2.5-pro:generateContent",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       channelconstant.ChannelTypeOpenRouter,
			ChannelBaseUrl:    "https://openrouter.ai/api",
			UpstreamModelName: "google/gemini-2.5-pro",
		},
	}
	url, err := a.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if url != "https://openrouter.ai/api/v1/chat/completions" {
		t.Fatalf("gemini url = %q, want https://openrouter.ai/api/v1/chat/completions", url)
	}
}

// OpenRouter 的 Anthropic 兼容端点使用 Bearer 认证；x-api-key 会被 OpenRouter
// 解释为直连 Anthropic 凭据，必须避免携带。
func TestSetupRequestHeaderOpenRouterClaudeBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	c.Request.Header.Set("anthropic-version", "2023-06-01-custom")
	c.Request.Header.Set("anthropic-beta", "context-management-2025-06-27")

	info := newOpenRouterClaudeInfo("https://openrouter.ai/api")
	header := http.Header{}
	if err := (&Adaptor{}).SetupRequestHeader(c, &header, info); err != nil {
		t.Fatalf("SetupRequestHeader error = %v", err)
	}

	if got := header.Get("Authorization"); got != "Bearer sk-or-test" {
		t.Fatalf("Authorization = %q, want Bearer sk-or-test", got)
	}
	if got := header.Get("x-api-key"); got != "" {
		t.Fatalf("x-api-key = %q, want 空（OpenRouter 将其解释为直连 Anthropic 凭据）", got)
	}
	if got := header.Get("anthropic-version"); got != "2023-06-01-custom" {
		t.Fatalf("anthropic-version = %q, want 透传客户端值 2023-06-01-custom", got)
	}
	if got := header.Get("anthropic-beta"); got != "context-management-2025-06-27" {
		t.Fatalf("anthropic-beta = %q, want 透传客户端值", got)
	}
}

// 客户端未传 anthropic-version 时使用 Anthropic 默认版本。
func TestSetupRequestHeaderOpenRouterClaudeDefaultVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)

	info := newOpenRouterClaudeInfo("https://openrouter.ai/api")
	header := http.Header{}
	if err := (&Adaptor{}).SetupRequestHeader(c, &header, info); err != nil {
		t.Fatalf("SetupRequestHeader error = %v", err)
	}
	if got := header.Get("anthropic-version"); got != "2023-06-01" {
		t.Fatalf("anthropic-version = %q, want 默认 2023-06-01", got)
	}
}

// OpenRouter 原生 Claude 响应由 claude adaptor 处理：响应体透传（模型名掩码），
// usage 按 Claude 语义归一化（PromptTokens 含 cache 读取/写入，与计费约定一致）。
func TestDoResponseOpenRouterClaudeNative(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)

	info := newOpenRouterClaudeInfo("https://openrouter.ai/api")
	body := `{"id":"msg_1","type":"message","role":"assistant","model":"anthropic/claude-sonnet-4","content":[{"type":"text","text":"你好"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"cache_read_input_tokens":5,"cache_creation_input_tokens":3,"output_tokens":7}}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
	if apiErr != nil {
		t.Fatalf("DoResponse error = %v", apiErr)
	}
	u, ok := usage.(*shared.Usage)
	if !ok {
		t.Fatalf("usage type = %T, want *shared.Usage", usage)
	}
	if u.PromptTokens != 18 {
		t.Fatalf("PromptTokens = %d, want 18（input 10 + cache_read 5 + cache_creation 3，含缓存归一化）", u.PromptTokens)
	}
	if u.PromptTokensDetails.CachedTokens != 5 {
		t.Fatalf("CachedTokens = %d, want 5", u.PromptTokensDetails.CachedTokens)
	}
	if u.PromptTokensDetails.CachedCreationTokens != 3 {
		t.Fatalf("CachedCreationTokens = %d, want 3", u.PromptTokensDetails.CachedCreationTokens)
	}
	if u.CompletionTokens != 7 {
		t.Fatalf("CompletionTokens = %d, want 7", u.CompletionTokens)
	}

	out := w.Body.String()
	if !strings.Contains(out, `"role":"assistant"`) {
		t.Fatalf("响应体应原生透传, body = %q", out)
	}
	if strings.Contains(out, "anthropic/claude-sonnet-4") {
		t.Fatalf("上游模型名应被掩码为客户端请求名, body = %q", out)
	}
	if !strings.Contains(out, `"claude-sonnet-4"`) {
		t.Fatalf("模型名应掩码为 claude-sonnet-4, body = %q", out)
	}
}
