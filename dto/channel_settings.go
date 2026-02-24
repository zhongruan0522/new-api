package dto

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

	// Convert image blocks (type: "image_url") into plain-text URLs and append them
	// to the last user message. Useful for text-only models to call external multimodal tools (e.g. MCP).
	//
	// Note: despite the legacy field name, it also applies to "video_url".
	ImageAutoConvertToURL bool `json:"image_auto_convert_to_url,omitempty"`
}

func (s *ChannelOtherSettings) IsOpenRouterEnterprise() bool {
	if s == nil || s.OpenRouterEnterprise == nil {
		return false
	}
	return *s.OpenRouterEnterprise
}
