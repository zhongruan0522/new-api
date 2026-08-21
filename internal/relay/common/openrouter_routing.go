package common

import (
	"fmt"

	channelconstant "github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ApplyProviderRouting reconciles the client-supplied `provider` object (an
// OpenRouter-specific routing preference) with the channel's configured
// OpenRouter routing preferences.
//
// For OpenRouter channels: channel-configured fields override same-named
// client fields, client fields the channel leaves unset are preserved, and a
// body without a client `provider` object receives the configured object
// verbatim. A channel without routing preferences leaves the client object
// untouched.
//
// For every other channel type the client `provider` object is stripped:
// the typed request DTOs capture it for the merge above, but non-OpenRouter
// upstreams must not receive OpenRouter-only fields.
func ApplyProviderRouting(jsonData []byte, info *RelayInfo) ([]byte, error) {
	if info == nil || info.ChannelMeta == nil {
		return jsonData, nil
	}
	if info.ChannelType != channelconstant.ChannelTypeOpenRouter {
		if gjson.GetBytes(jsonData, "provider").Exists() {
			return sjson.DeleteBytes(jsonData, "provider")
		}
		return jsonData, nil
	}
	routing := info.ChannelOtherSettings.OpenRouterRouting
	if routing.IsEmpty() {
		return jsonData, nil
	}
	routingJSON, err := jsonx.Marshal(routing)
	if err != nil {
		return nil, fmt.Errorf("marshal openrouter routing: %w", err)
	}
	clientProvider := gjson.GetBytes(jsonData, "provider")
	if !clientProvider.Exists() || clientProvider.Type == gjson.Null {
		return sjson.SetRawBytes(jsonData, "provider", routingJSON)
	}
	if !clientProvider.IsObject() {
		return nil, fmt.Errorf(`request field "provider" must be a JSON object`)
	}
	var merged map[string]interface{}
	if err := jsonx.Unmarshal([]byte(clientProvider.Raw), &merged); err != nil {
		return nil, fmt.Errorf(`parse request field "provider": %w`, err)
	}
	var overlay map[string]interface{}
	if err := jsonx.Unmarshal(routingJSON, &overlay); err != nil {
		return nil, fmt.Errorf("parse channel openrouter routing: %w", err)
	}
	for key, value := range overlay {
		merged[key] = value
	}
	mergedJSON, err := jsonx.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("merge openrouter routing: %w", err)
	}
	return sjson.SetRawBytes(jsonData, "provider", mergedJSON)
}
