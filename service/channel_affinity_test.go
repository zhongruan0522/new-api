package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/setting/operation_setting"
)

func TestExtractChannelAffinityValueFromRequestHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		headerName string
		headerValue string
		key        string
		want       string
	}{
		{
			name:        "extracts canonical header value",
			headerName:  "X-Conversation-Id",
			headerValue: "conv-123",
			key:         "X-Conversation-Id",
			want:        "conv-123",
		},
		{
			name:        "header lookup is case-insensitive",
			headerName:  "X-Conversation-Id",
			headerValue: "conv-456",
			key:         "x-conversation-id",
			want:        "conv-456",
		},
		{
			name:        "trims whitespace around value",
			headerName:  "X-Trace-Id",
			headerValue: "  trace-789  ",
			key:         "X-Trace-Id",
			want:        "trace-789",
		},
		{
			name:        "empty key returns empty",
			headerName:  "X-Trace-Id",
			headerValue: "value",
			key:         "",
			want:        "",
		},
		{
			name:        "missing header returns empty",
			headerName:  "X-Other-Id",
			headerValue: "value",
			key:         "X-Missing-Id",
			want:        "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
			if tc.headerValue != "" {
				req.Header.Set(tc.headerName, tc.headerValue)
			}
			c, _ := gin.CreateTestContext(recorder)
			c.Request = req

			src := operation_setting.ChannelAffinityKeySource{
				Type: "request_header",
				Key:  tc.key,
			}
			got := extractChannelAffinityValue(c, src)
			if got != tc.want {
				t.Fatalf("extractChannelAffinityValue() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractChannelAffinityValueRequestHeaderWithoutRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = nil

	src := operation_setting.ChannelAffinityKeySource{
		Type: "request_header",
		Key:  "X-Conversation-Id",
	}
	got := extractChannelAffinityValue(c, src)
	if got != "" {
		t.Fatalf("expected empty value when request is nil, got %q", got)
	}
}

func TestGetPreferredChannelByAffinityFromRequestHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setting := operation_setting.GetChannelAffinitySetting()
	origEnabled := setting.Enabled
	origRules := make([]operation_setting.ChannelAffinityRule, len(setting.Rules))
	copy(origRules, setting.Rules)

	setting.Enabled = true
	setting.Rules = []operation_setting.ChannelAffinityRule{
		{
			Name:        "trace-by-header",
			ModelRegex:  []string{"^gpt-.*$"},
			PathRegex:   []string{"/v1/chat/completions"},
			KeySources: []operation_setting.ChannelAffinityKeySource{
				{Type: "request_header", Key: "X-Conversation-Id"},
			},
			ValueRegex:         "^[-0-9A-Za-z]+$",
			TTLSeconds:         60,
			SkipRetryOnFailure: false,
			IncludeUsingGroup:  false,
			IncludeRuleName:    true,
			IncludeModelName:   false,
		},
	}

	t.Cleanup(func() {
		setting.Enabled = origEnabled
		setting.Rules = origRules
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	req.Header.Set("X-Conversation-Id", "conv-abc-123")
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req

	_, matched := GetPreferredChannelByAffinity(c, "gpt-4o", "default")
	if matched {
		t.Fatal("expected no cached channel, but matched was true")
	}

	meta, ok := getChannelAffinityMeta(c)
	if !ok {
		t.Fatal("expected channel affinity meta to be set after rule matched")
	}
	if meta.KeySourceType != "request_header" {
		t.Fatalf("expected key source type request_header, got %q", meta.KeySourceType)
	}
	if meta.KeySourceKey != "X-Conversation-Id" {
		t.Fatalf("expected key source key X-Conversation-Id, got %q", meta.KeySourceKey)
	}
	if meta.KeyHint != "conv-abc-123" {
		t.Fatalf("expected key hint conv-abc-123, got %q", meta.KeyHint)
	}
}

func TestGetPreferredChannelByAffinityRequestHeaderValueRegexFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setting := operation_setting.GetChannelAffinitySetting()
	origEnabled := setting.Enabled
	origRules := make([]operation_setting.ChannelAffinityRule, len(setting.Rules))
	copy(origRules, setting.Rules)

	setting.Enabled = true
	setting.Rules = []operation_setting.ChannelAffinityRule{
		{
			Name:        "trace-by-header",
			ModelRegex:  []string{"^gpt-.*$"},
			PathRegex:   []string{"/v1/chat/completions"},
			KeySources: []operation_setting.ChannelAffinityKeySource{
				{Type: "request_header", Key: "X-Conversation-Id"},
			},
			ValueRegex:         "^conv-.*$",
			TTLSeconds:         60,
			SkipRetryOnFailure: false,
			IncludeUsingGroup:  false,
			IncludeRuleName:    true,
			IncludeModelName:   false,
		},
	}

	t.Cleanup(func() {
		setting.Enabled = origEnabled
		setting.Rules = origRules
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	req.Header.Set("X-Conversation-Id", "not-matching")
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req

	_, matched := GetPreferredChannelByAffinity(c, "gpt-4o", "default")
	if matched {
		t.Fatal("expected rule to be rejected by value regex")
	}
}

func TestGetPreferredChannelByAffinityRequestHeaderMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setting := operation_setting.GetChannelAffinitySetting()
	origEnabled := setting.Enabled
	origRules := make([]operation_setting.ChannelAffinityRule, len(setting.Rules))
	copy(origRules, setting.Rules)

	setting.Enabled = true
	setting.Rules = []operation_setting.ChannelAffinityRule{
		{
			Name:        "trace-by-header",
			ModelRegex:  []string{"^gpt-.*$"},
			PathRegex:   []string{"/v1/chat/completions"},
			KeySources: []operation_setting.ChannelAffinityKeySource{
				{Type: "request_header", Key: "X-Conversation-Id"},
			},
			TTLSeconds:         60,
			SkipRetryOnFailure: false,
			IncludeUsingGroup:  false,
			IncludeRuleName:    true,
			IncludeModelName:   false,
		},
	}

	t.Cleanup(func() {
		setting.Enabled = origEnabled
		setting.Rules = origRules
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req

	_, matched := GetPreferredChannelByAffinity(c, "gpt-4o", "default")
	if matched {
		t.Fatal("expected rule not to match when header is missing")
	}
}
