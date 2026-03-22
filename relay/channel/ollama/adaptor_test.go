package ollama

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/types"

	"github.com/gin-gonic/gin"
)

func TestGetRequestURLUsesCompatiblePath(t *testing.T) {
	tests := []struct {
		name           string
		channelBaseURL string
		requestPath    string
		expectURL      string
	}{
		{
			name:           "chat completions",
			channelBaseURL: "http://ollama.local",
			requestPath:    "/v1/chat/completions",
			expectURL:      "http://ollama.local/v1/chat/completions",
		},
		{
			name:           "anthropic messages with query",
			channelBaseURL: "http://ollama.local",
			requestPath:    "/v1/messages?beta=true",
			expectURL:      "http://ollama.local/v1/messages?beta=true",
		},
	}

	adaptor := &Adaptor{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta:    &relaycommon.ChannelMeta{ChannelBaseUrl: tt.channelBaseURL},
				RequestURLPath: tt.requestPath,
			}

			got, err := adaptor.GetRequestURL(info)
			if err != nil {
				t.Fatalf("GetRequestURL returned error: %v", err)
			}
			if got != tt.expectURL {
				t.Fatalf("GetRequestURL = %q, want %q", got, tt.expectURL)
			}
		})
	}
}

func TestSetupRequestHeaderOpenAI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		IsStream:    true,
		ChannelMeta: &relaycommon.ChannelMeta{ApiKey: "secret-key"},
	}

	headers := http.Header{}
	adaptor := &Adaptor{}
	if err := adaptor.SetupRequestHeader(ctx, &headers, info); err != nil {
		t.Fatalf("SetupRequestHeader returned error: %v", err)
	}

	if got := headers.Get("Authorization"); got != "Bearer secret-key" {
		t.Fatalf("Authorization = %q, want %q", got, "Bearer secret-key")
	}
	if got := headers.Get("Accept"); got != "text/event-stream" {
		t.Fatalf("Accept = %q, want %q", got, "text/event-stream")
	}
}

func TestSetupRequestHeaderClaude(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("anthropic-beta", "tools-2025-01-01")

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{ApiKey: "secret-key"},
	}

	headers := http.Header{}
	adaptor := &Adaptor{}
	if err := adaptor.SetupRequestHeader(ctx, &headers, info); err != nil {
		t.Fatalf("SetupRequestHeader returned error: %v", err)
	}

	if got := headers.Get("x-api-key"); got != "secret-key" {
		t.Fatalf("x-api-key = %q, want %q", got, "secret-key")
	}
	if got := headers.Get("anthropic-version"); got != "2023-06-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2023-06-01")
	}
	if got := headers.Get("anthropic-beta"); got != "tools-2025-01-01" {
		t.Fatalf("anthropic-beta = %q, want %q", got, "tools-2025-01-01")
	}
	if got := headers.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want empty", got)
	}
}

func TestConvertOpenAIResponsesRequestPassThrough(t *testing.T) {
	adaptor := &Adaptor{}
	request := dto.OpenAIResponsesRequest{Model: "qwen3:8b", Stream: true}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, nil, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIResponsesRequest returned error: %v", err)
	}

	convertedRequest, ok := converted.(dto.OpenAIResponsesRequest)
	if !ok {
		t.Fatalf("converted request type = %T, want dto.OpenAIResponsesRequest", converted)
	}
	if convertedRequest.Model != request.Model || convertedRequest.Stream != request.Stream {
		t.Fatalf("converted request = %+v, want %+v", convertedRequest, request)
	}
}

func TestFetchOllamaModelsUsesOpenAICompatibleEndpoint(t *testing.T) {
	var requestPath string
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"qwen3:8b","created":1710000000,"owned_by":"library"}]}`))
	}))
	defer server.Close()

	models, err := FetchOllamaModels(server.URL, "ollama")
	if err != nil {
		t.Fatalf("FetchOllamaModels returned error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("model count = %d, want 1", len(models))
	}
	if requestPath != "/v1/models" {
		t.Fatalf("request path = %q, want %q", requestPath, "/v1/models")
	}
	if authHeader != "Bearer ollama" {
		t.Fatalf("Authorization = %q, want %q", authHeader, "Bearer ollama")
	}
	if models[0].Name != "qwen3:8b" {
		t.Fatalf("model name = %q, want %q", models[0].Name, "qwen3:8b")
	}
	if models[0].OwnedBy != "library" {
		t.Fatalf("owned_by = %q, want %q", models[0].OwnedBy, "library")
	}
	if models[0].Created != 1710000000 {
		t.Fatalf("created = %d, want %d", models[0].Created, int64(1710000000))
	}
	if _, err := time.Parse(time.RFC3339, models[0].ModifiedAt); err != nil {
		t.Fatalf("modified_at = %q is not RFC3339: %v", models[0].ModifiedAt, err)
	}
}
