package dto

import "strings"

type ChannelSettings struct {
	ForceFormat       bool `json:"force_format,omitempty"`
	ThinkingToContent bool `json:"thinking_to_content,omitempty"`
	// OpenAIWireAPI controls which OpenAI wire API this channel should treat as its default upstream spec.
	// Supported values:
	//   - "both"      : channel is compatible with both /v1/chat/completions and /v1/responses (no auto conversion)
	//   - "chat"      : treat ChatCompletions as default; auto-convert Responses -> ChatCompletions when needed
	//   - "responses" : treat Responses as default; auto-convert ChatCompletions -> Responses when needed
	//
	// Empty value is treated as "both" for backward compatibility.
	OpenAIWireAPI             OpenAIWireAPI `json:"openai_wire_api,omitempty"`
	Proxy                     string        `json:"proxy"`
	PassThroughBodyEnabled    bool          `json:"pass_through_body_enabled,omitempty"`
	PassThroughHeadersEnabled bool          `json:"pass_through_headers_enabled"`
}

type OpenAIWireAPI string

const (
	OpenAIWireAPIBoth      OpenAIWireAPI = "both"
	OpenAIWireAPIChat      OpenAIWireAPI = "chat"
	OpenAIWireAPIResponses OpenAIWireAPI = "responses"
)

func (api OpenAIWireAPI) Normalize() (OpenAIWireAPI, bool) {
	raw := strings.TrimSpace(strings.ToLower(string(api)))
	if raw == "" {
		return OpenAIWireAPIBoth, true
	}
	switch OpenAIWireAPI(raw) {
	case OpenAIWireAPIBoth, OpenAIWireAPIChat, OpenAIWireAPIResponses:
		return OpenAIWireAPI(raw), true
	default:
		return OpenAIWireAPIBoth, false
	}
}

type VertexKeyType string

const (
	VertexKeyTypeJSON   VertexKeyType = "json"
	VertexKeyTypeAPIKey VertexKeyType = "api_key"
)

type AwsKeyType string

const (
	AwsKeyTypeAKSK   AwsKeyType = "ak_sk" // default
	AwsKeyTypeApiKey AwsKeyType = "api_key"
)

type ChannelOtherSettings struct {
	AzureResponsesVersion string        `json:"azure_responses_version,omitempty"`
	VertexKeyType         VertexKeyType `json:"vertex_key_type,omitempty"` // "json" or "api_key"
	OpenRouterEnterprise  *bool         `json:"openrouter_enterprise,omitempty"`

	ClaudeBetaQuery       bool       `json:"claude_beta_query,omitempty"`
	AllowServiceTier      bool       `json:"allow_service_tier,omitempty"`
	DisableStore          bool       `json:"disable_store,omitempty"`
	AllowSafetyIdentifier bool       `json:"allow_safety_identifier,omitempty"`
	AwsKeyType            AwsKeyType `json:"aws_key_type,omitempty"`

	// ImageAutoConvertToURLMode selects how to handle multimodal media blocks
	// (image_url/video_url) when the upstream model is text-only.
	//
	// Supported values:
	//   - "off" : disable rewriting
	//   - "mcp" : append media URLs as text and instruct the model to use MCP/tools
	//
	// Legacy compatibility:
	// If this field is empty, ImageAutoConvertToURL will still be read so old
	// channel records can be migrated cleanly on startup.
	//   - true  => "mcp"
	//   - false => "off"
	ImageAutoConvertToURLMode string `json:"image_auto_convert_to_url_mode,omitempty"`

	// ImageAutoConvertToURL is a removed legacy field that is kept read-only for
	// compatibility with existing rows before migration cleanup runs.
	ImageAutoConvertToURL bool `json:"image_auto_convert_to_url,omitempty"`
}

type ImageAutoConvertToURLMode string

const (
	ImageAutoConvertToURLModeOff ImageAutoConvertToURLMode = "off"
	ImageAutoConvertToURLModeMCP ImageAutoConvertToURLMode = "mcp"
)

func (s ChannelOtherSettings) ParseImageAutoConvertToURLMode() (mode ImageAutoConvertToURLMode, ok bool) {
	raw := strings.TrimSpace(strings.ToLower(s.ImageAutoConvertToURLMode))
	if raw == "" {
		if s.ImageAutoConvertToURL {
			return ImageAutoConvertToURLModeMCP, true
		}
		return ImageAutoConvertToURLModeOff, true
	}

	switch ImageAutoConvertToURLMode(raw) {
	case ImageAutoConvertToURLModeOff, ImageAutoConvertToURLModeMCP:
		return ImageAutoConvertToURLMode(raw), true
	case "third_party_model":
		// Keep old rows readable until the startup migration rewrites them to "mcp".
		return ImageAutoConvertToURLModeMCP, true
	default:
		return ImageAutoConvertToURLModeOff, false
	}
}

func (s *ChannelOtherSettings) IsOpenRouterEnterprise() bool {
	if s == nil || s.OpenRouterEnterprise == nil {
		return false
	}
	return *s.OpenRouterEnterprise
}
