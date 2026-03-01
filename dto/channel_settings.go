package dto

import "strings"

type ChannelSettings struct {
	ForceFormat               bool   `json:"force_format,omitempty"`
	ThinkingToContent         bool   `json:"thinking_to_content,omitempty"`
	Proxy                     string `json:"proxy"`
	PassThroughBodyEnabled    bool   `json:"pass_through_body_enabled,omitempty"`
	PassThroughHeadersEnabled bool   `json:"pass_through_headers_enabled"`
	SystemPrompt              string `json:"system_prompt,omitempty"`
	SystemPromptOverride      bool   `json:"system_prompt_override,omitempty"`
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

	// ImageAutoConvertToURLMode selects how to handle multimodal media blocks (image_url/video_url)
	// when the upstream model is text-only.
	//
	// Supported values:
	//   - "off"               : disable rewriting
	//   - "mcp"               : legacy behavior, append media URLs as text and instruct MCP usage
	//   - "third_party_model" : call a configured multimodal model to convert media into text, then append
	//
	// Backward compatibility:
	// If this field is empty, ImageAutoConvertToURL (legacy boolean) will be used:
	//   - true  => "mcp"
	//   - false => "off"
	ImageAutoConvertToURLMode string `json:"image_auto_convert_to_url_mode,omitempty"`

	// Convert image blocks (type: "image_url") into plain-text URLs and append them
	// to the end of the corresponding user message. Useful for text-only models to call external multimodal tools (e.g. MCP).
	//
	// Note: despite the legacy field name, it also applies to "video_url".
	ImageAutoConvertToURL bool `json:"image_auto_convert_to_url,omitempty"`
}

type ImageAutoConvertToURLMode string

const (
	ImageAutoConvertToURLModeOff             ImageAutoConvertToURLMode = "off"
	ImageAutoConvertToURLModeMCP             ImageAutoConvertToURLMode = "mcp"
	ImageAutoConvertToURLModeThirdPartyModel ImageAutoConvertToURLMode = "third_party_model"
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
	case ImageAutoConvertToURLModeOff, ImageAutoConvertToURLModeMCP, ImageAutoConvertToURLModeThirdPartyModel:
		return ImageAutoConvertToURLMode(raw), true
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
