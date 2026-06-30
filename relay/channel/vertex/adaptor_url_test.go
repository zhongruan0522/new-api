package vertex

import (
	"strings"
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func vertexInfo(t *testing.T, baseURL string, apiKey string, apiVersion string, stream bool) *relaycommon.RelayInfo {
	t.Helper()
	return &relaycommon.RelayInfo{
		IsStream:        stream,
		OriginModelName: "gemini-2.5-pro",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       17,
			ChannelBaseUrl:    baseURL,
			ApiKey:            apiKey,
			ApiVersion:        apiVersion,
			UpstreamModelName: "gemini-2.5-pro",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				VertexKeyType: dto.VertexKeyTypeJSON,
			},
		},
	}
}

func serviceAccountJSON(t *testing.T, projectID string) string {
	t.Helper()
	raw, err := common.Marshal(Credentials{ProjectID: projectID})
	if err != nil {
		t.Fatalf("marshal credentials: %v", err)
	}
	return string(raw)
}

// TestAdaptorGetRequestURL_HonorsCustomBaseURL verifies the user-facing fix for
// #148: a Vertex channel configured with a custom base_url/gateway prefix must
// send requests to that gateway instead of the hardcoded Google host.
func TestAdaptorGetRequestURL_HonorsCustomBaseURL(t *testing.T) {
	a := &Adaptor{RequestMode: RequestModeGemini}
	info := vertexInfo(t, "https://gateway.example/vertex", serviceAccountJSON(t, "p1"), "us-central1", false)

	got, err := a.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if !strings.HasPrefix(got, "https://gateway.example/vertex/") {
		t.Fatalf("Gemini URL did not honor custom base_url: %s", got)
	}
	if !strings.Contains(got, "publishers/google/models/gemini-2.5-pro:generateContent") {
		t.Fatalf("Gemini URL lost model/action suffix: %s", got)
	}

	// Streaming must keep the alt=sse query on the same gateway host.
	info.IsStream = true
	got, err = a.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL stream error = %v", err)
	}
	if !strings.HasPrefix(got, "https://gateway.example/vertex/") {
		t.Fatalf("Gemini stream URL did not honor custom base_url: %s", got)
	}
	if !strings.Contains(got, "streamGenerateContent?alt=sse") {
		t.Fatalf("Gemini stream URL lost streaming suffix: %s", got)
	}
}

// TestAdaptorGetRequestURL_ClaudeHonorsCustomBaseURL ensures Claude-on-Vertex
// requests also route through a configured gateway prefix.
func TestAdaptorGetRequestURL_ClaudeHonorsCustomBaseURL(t *testing.T) {
	a := &Adaptor{RequestMode: RequestModeClaude}
	info := vertexInfo(t, "https://gateway.example/vertex", serviceAccountJSON(t, "p1"), "global", false)
	info.UpstreamModelName = "claude-sonnet-4-20250514"
	info.OriginModelName = "claude-sonnet-4-20250514"

	got, err := a.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if !strings.HasPrefix(got, "https://gateway.example/vertex/") {
		t.Fatalf("Claude URL did not honor custom base_url: %s", got)
	}
	if !strings.Contains(got, "publishers/anthropic/models/claude-sonnet-4@20250514:rawPredict") {
		t.Fatalf("Claude URL lost anthropic publisher/model/action: %s", got)
	}
}

// TestAdaptorGetRequestURL_DefaultHostUnchanged guards against regressing the
// default Google host when no base_url is configured.
func TestAdaptorGetRequestURL_DefaultHostUnchanged(t *testing.T) {
	a := &Adaptor{RequestMode: RequestModeGemini}
	info := vertexInfo(t, "", serviceAccountJSON(t, "p1"), "global", false)

	got, err := a.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if !strings.HasPrefix(got, "https://aiplatform.googleapis.com/") {
		t.Fatalf("default Gemini URL should keep Google host, got: %s", got)
	}
}
