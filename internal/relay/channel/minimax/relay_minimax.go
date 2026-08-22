package minimax

import (
	"fmt"

	channelconstant "github.com/NookMux/NookMux/internal/domain/channel/constant"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/internal/relay/constant"
)

func GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseUrl := info.ChannelBaseUrl
	if baseUrl == "" {
		baseUrl = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeMiniMax]
	}

	// CodingPlan 模式下，Claude 格式走专用地址
	if specialPlan, ok := channelconstant.ChannelSpecialBases[baseUrl]; ok {
		if info.RelayFormat == constant.RelayFormatClaude && specialPlan.ClaudeBaseURL != "" {
			return fmt.Sprintf("%s/v1/messages", specialPlan.ClaudeBaseURL), nil
		}
	}

	switch info.RelayMode {
	case constant.RelayModeChatCompletions:
		// MiniMax 已兼容 OpenAI Chat 规范，直接使用标准路径
		return fmt.Sprintf("%s/chat/completions", baseUrl), nil
	case constant.RelayModeAudioSpeech:
		return fmt.Sprintf("%s/t2a_v2", baseUrl), nil
	case constant.RelayModeImagesGenerations:
		return fmt.Sprintf("%s/image_generation", baseUrl), nil
	default:
		return "", fmt.Errorf("unsupported relay mode: %d", info.RelayMode)
	}
}
