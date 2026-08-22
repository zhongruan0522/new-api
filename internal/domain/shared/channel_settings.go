package shared

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/NookMux/NookMux/pkg/jsonx"
)

type ChannelSettings struct {
	ForceFormat bool `json:"force_format,omitempty"`
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
	// OpenRouterRouting carries per-channel OpenRouter provider routing
	// preferences; nil means the channel does not intervene in routing and
	// client-supplied `provider` objects pass through untouched.
	OpenRouterRouting *OpenRouterRouting `json:"openrouter_routing,omitempty"`

	ClaudeBetaQuery       bool       `json:"claude_beta_query,omitempty"`
	AllowCacheControl     bool       `json:"allow_cache_control,omitempty"`
	AllowSpeed            bool       `json:"allow_speed,omitempty"`
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

// OpenRouterRouting mirrors OpenRouter's `provider` routing preference object
// (order/only/ignore/allow_fallbacks/data_collection/quantizations/sort/
// preferred_min_throughput/preferred_max_latency/max_price). Field names match
// the upstream wire format so the configured object can be merged into request
// bodies without renaming. Pointer and omitempty semantics keep explicit
// zero values (e.g. allow_fallbacks=false) distinguishable from "unset".
type OpenRouterRouting struct {
	Order                  []string               `json:"order,omitempty"`
	Only                   []string               `json:"only,omitempty"`
	Ignore                 []string               `json:"ignore,omitempty"`
	AllowFallbacks         *bool                  `json:"allow_fallbacks,omitempty"`
	RequireParameters      *bool                  `json:"require_parameters,omitempty"`
	DataCollection         string                 `json:"data_collection,omitempty"` // "" | "allow" | "deny"
	Zdr                    *bool                  `json:"zdr,omitempty"`
	EnforceDistillableText *bool                  `json:"enforce_distillable_text,omitempty"`
	Quantizations          []string               `json:"quantizations,omitempty"`
	Sort                   *OpenRouterRoutingSort `json:"sort,omitempty"`
	// PreferredMinThroughput / PreferredMaxLatency accept the upstream dual
	// shape: a bare number (applies to p50) or a single-percentile object.
	PreferredMinThroughput *OpenRouterThreshold `json:"preferred_min_throughput,omitempty"`
	PreferredMaxLatency    *OpenRouterThreshold `json:"preferred_max_latency,omitempty"`
	MaxPrice               *OpenRouterMaxPrice  `json:"max_price,omitempty"`
}

type OpenRouterRoutingSort struct {
	By        string `json:"by"`                  // "price" | "throughput" | "latency"
	Partition string `json:"partition,omitempty"` // "" (defaults to model) | "model" | "none"
}

// KnownOpenRouterRoutingKeys is the closed set of fields accepted inside the
// openrouter_routing settings object; store-side validation rejects anything
// else so typos fail loudly instead of silently not taking effect.
var KnownOpenRouterRoutingKeys = map[string]bool{
	"order":                    true,
	"only":                     true,
	"ignore":                   true,
	"allow_fallbacks":          true,
	"require_parameters":       true,
	"data_collection":          true,
	"zdr":                      true,
	"enforce_distillable_text": true,
	"quantizations":            true,
	"sort":                     true,
	"preferred_min_throughput": true,
	"preferred_max_latency":    true,
	"max_price":                true,
}

// OpenRouterThreshold marshals as a number when Percentile is empty and as
// {"<percentile>": value} otherwise, matching the upstream provider object.
type OpenRouterThreshold struct {
	Value      float64
	Percentile string // "" | "p50" | "p75" | "p90" | "p99"
}

var openRouterPercentiles = map[string]bool{"p50": true, "p75": true, "p90": true, "p99": true}

func (t OpenRouterThreshold) MarshalJSON() ([]byte, error) {
	if t.Percentile == "" {
		return jsonx.Marshal(t.Value)
	}
	if !openRouterPercentiles[t.Percentile] {
		return nil, fmt.Errorf("invalid openrouter percentile %q", t.Percentile)
	}
	return jsonx.Marshal(map[string]float64{t.Percentile: t.Value})
}

func (t *OpenRouterThreshold) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		var raw map[string]json.RawMessage
		if err := jsonx.Unmarshal(data, &raw); err != nil {
			return err
		}
		if len(raw) != 1 {
			return fmt.Errorf("openrouter threshold object must contain exactly one percentile key")
		}
		for key, rawValue := range raw {
			if !openRouterPercentiles[key] {
				return fmt.Errorf("invalid openrouter percentile %q", key)
			}
			// encoding/json treats null as a no-op when decoding into float64,
			// which would silently turn "unset" into 0; reject it explicitly.
			if strings.EqualFold(strings.TrimSpace(string(rawValue)), "null") {
				return fmt.Errorf("openrouter percentile %q value must not be null", key)
			}
			var value float64
			if err := jsonx.Unmarshal(rawValue, &value); err != nil {
				return fmt.Errorf("openrouter percentile %q value must be a number: %w", key, err)
			}
			t.Percentile = key
			t.Value = value
			return nil
		}
	}
	if err := jsonx.Unmarshal(data, &t.Value); err != nil {
		return err
	}
	t.Percentile = ""
	return nil
}

type OpenRouterMaxPrice struct {
	Prompt     *float64 `json:"prompt,omitempty"`
	Completion *float64 `json:"completion,omitempty"`
	Request    *float64 `json:"request,omitempty"`
	Image      *float64 `json:"image,omitempty"`
}

func (r *OpenRouterRouting) IsEmpty() bool {
	if r == nil {
		return true
	}
	return len(r.Order) == 0 &&
		len(r.Only) == 0 &&
		len(r.Ignore) == 0 &&
		r.AllowFallbacks == nil &&
		r.RequireParameters == nil &&
		r.DataCollection == "" &&
		r.Zdr == nil &&
		r.EnforceDistillableText == nil &&
		len(r.Quantizations) == 0 &&
		r.Sort == nil &&
		r.PreferredMinThroughput == nil &&
		r.PreferredMaxLatency == nil &&
		r.MaxPrice == nil
}

// Validate checks closed-set enums and numeric bounds. Slug lists
// (order/only/ignore) and quantizations are validated for shape only, not
// membership: OpenRouter adds providers and quantization levels over time and
// strict allow-lists would reject values that are valid upstream.
func (r *OpenRouterRouting) Validate() error {
	if r == nil {
		return nil
	}
	for field, slugs := range map[string][]string{
		"order":         r.Order,
		"only":          r.Only,
		"ignore":        r.Ignore,
		"quantizations": r.Quantizations,
	} {
		for _, slug := range slugs {
			if strings.TrimSpace(slug) == "" {
				return fmt.Errorf("openrouter_routing.%s contains an empty entry", field)
			}
		}
	}
	switch r.DataCollection {
	case "", "allow", "deny":
	default:
		return fmt.Errorf("openrouter_routing.data_collection must be \"allow\" or \"deny\"")
	}
	if r.Sort != nil {
		switch r.Sort.By {
		case "price", "throughput", "latency":
		default:
			return fmt.Errorf("openrouter_routing.sort.by must be \"price\", \"throughput\" or \"latency\"")
		}
		switch r.Sort.Partition {
		case "", "model", "none":
		default:
			return fmt.Errorf("openrouter_routing.sort.partition must be \"model\" or \"none\"")
		}
	}
	for name, threshold := range map[string]*OpenRouterThreshold{
		"preferred_min_throughput": r.PreferredMinThroughput,
		"preferred_max_latency":    r.PreferredMaxLatency,
	} {
		if threshold == nil {
			continue
		}
		if threshold.Value < 0 {
			return fmt.Errorf("openrouter_routing.%s must be >= 0", name)
		}
		if threshold.Percentile != "" && !openRouterPercentiles[threshold.Percentile] {
			return fmt.Errorf("openrouter_routing.%s has invalid percentile %q", name, threshold.Percentile)
		}
	}
	if r.MaxPrice != nil {
		for name, price := range map[string]*float64{
			"prompt":     r.MaxPrice.Prompt,
			"completion": r.MaxPrice.Completion,
			"request":    r.MaxPrice.Request,
			"image":      r.MaxPrice.Image,
		} {
			if price != nil && *price < 0 {
				return fmt.Errorf("openrouter_routing.max_price.%s must be >= 0", name)
			}
		}
	}
	return nil
}
