package model

import (
	"encoding/json"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/pkg/jsonx"

	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
)

func preferChannelsByRequestFormat(channels []*Channel, preferredAPIType int, relayFormat relayconstant.RelayFormat) []*Channel {
	if preferredAPIType < 0 && !isOpenAIWireRelayFormat(relayFormat) {
		return channels
	}
	if matched := preferChannelsByExplicitOpenAIWireAPI(channels, relayFormat); len(matched) > 0 {
		return matched
	}
	if preferredAPIType >= 0 {
		return preferChannelsByAPIType(channels, preferredAPIType)
	}
	return channels
}

func preferAbilitiesByRequestFormat(abilities []Ability, preferredAPIType int, relayFormat relayconstant.RelayFormat) []Ability {
	if preferredAPIType < 0 && !isOpenAIWireRelayFormat(relayFormat) {
		return abilities
	}

	channelIDs := make([]int, 0, len(abilities))
	for _, ability := range abilities {
		channelIDs = append(channelIDs, ability.ChannelId)
	}

	var channels []Channel
	if err := DB.Select("id, type, setting").Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
		return abilities
	}

	channelsByID := make(map[int]Channel, len(channels))
	channelPointers := make([]*Channel, 0, len(channels))
	for i := range channels {
		channelsByID[channels[i].Id] = channels[i]
		channelPointers = append(channelPointers, &channels[i])
	}

	if matchedChannels := preferChannelsByExplicitOpenAIWireAPI(channelPointers, relayFormat); len(matchedChannels) > 0 {
		matchedIDs := make(map[int]struct{}, len(matchedChannels))
		for _, ch := range matchedChannels {
			matchedIDs[ch.Id] = struct{}{}
		}
		matchedAbilities := make([]Ability, 0, len(abilities))
		for _, ability := range abilities {
			if _, ok := matchedIDs[ability.ChannelId]; ok {
				matchedAbilities = append(matchedAbilities, ability)
			}
		}
		if len(matchedAbilities) > 0 {
			return matchedAbilities
		}
	}

	if preferredAPIType < 0 {
		return abilities
	}

	matched := make([]Ability, 0, len(abilities))
	for _, ability := range abilities {
		ch, ok := channelsByID[ability.ChannelId]
		if !ok {
			continue
		}
		apiType, _ := common.ChannelType2APIType(ch.Type)
		if apiType == preferredAPIType {
			matched = append(matched, ability)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return abilities
}

func preferChannelsByExplicitOpenAIWireAPI(channels []*Channel, relayFormat relayconstant.RelayFormat) []*Channel {
	if !isOpenAIWireRelayFormat(relayFormat) {
		return nil
	}

	matched := make([]*Channel, 0, len(channels))
	for _, ch := range channels {
		wire, ok := explicitlyConfiguredOpenAIWireAPI(ch)
		if !ok || !openAIWireAPISelectableForRelayFormat(wire, relayFormat) {
			continue
		}
		matched = append(matched, ch)
	}
	return matched
}

func isOpenAIWireRelayFormat(relayFormat relayconstant.RelayFormat) bool {
	switch relayFormat {
	case relayconstant.RelayFormatOpenAI, relayconstant.RelayFormatOpenAIResponses, relayconstant.RelayFormatOpenAIResponsesCompaction:
		return true
	default:
		return false
	}
}

func openAIWireAPISelectableForRelayFormat(wire shared.OpenAIWireAPI, relayFormat relayconstant.RelayFormat) bool {
	switch relayFormat {
	case relayconstant.RelayFormatOpenAI, relayconstant.RelayFormatOpenAIResponses:
		return wire == shared.OpenAIWireAPIChat || wire == shared.OpenAIWireAPIResponses
	case relayconstant.RelayFormatOpenAIResponsesCompaction:
		return wire == shared.OpenAIWireAPIResponses
	default:
		return false
	}
}

func explicitlyConfiguredOpenAIWireAPI(channel *Channel) (shared.OpenAIWireAPI, bool) {
	if channel == nil || channel.Setting == nil || *channel.Setting == "" {
		return "", false
	}

	var raw map[string]json.RawMessage
	if err := jsonx.Unmarshal([]byte(*channel.Setting), &raw); err != nil {
		return "", false
	}

	rawWire, ok := raw["openai_wire_api"]
	if !ok {
		return "", false
	}

	var configured shared.OpenAIWireAPI
	if err := jsonx.Unmarshal(rawWire, &configured); err != nil {
		return "", false
	}

	wire, ok := configured.Normalize()
	if !ok || wire == shared.OpenAIWireAPIBoth {
		return "", false
	}
	return wire, true
}
