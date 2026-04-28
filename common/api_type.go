package common

import "github.com/zhongruan0522/new-api/constant"

func ChannelType2APIType(channelType int) (int, bool) {
	apiType := -1
	switch channelType {
	case constant.ChannelTypeOpenAI:
		apiType = constant.APITypeOpenAI
	case constant.ChannelTypeAnthropic:
		apiType = constant.APITypeAnthropic
	case constant.ChannelTypeGemini:
		apiType = constant.APITypeGemini
	case constant.ChannelTypeZhipu_v4:
		apiType = constant.APITypeZhipuV4
	case constant.ChannelTypeOllama:
		apiType = constant.APITypeOllama
	case constant.ChannelTypeAws:
		apiType = constant.APITypeAws
	case constant.ChannelTypeSiliconFlow:
		apiType = constant.APITypeSiliconFlow
	case constant.ChannelTypeVertexAi:
		apiType = constant.APITypeVertexAi
	case constant.ChannelTypeDeepSeek:
		apiType = constant.APITypeDeepSeek
	case constant.ChannelTypeOpenRouter:
		apiType = constant.APITypeOpenRouter
	case constant.ChannelTypeMoonshot:
		apiType = constant.APITypeMoonshot
	case constant.ChannelTypeMiniMax:
		apiType = constant.APITypeMiniMax
	case constant.ChannelTypeXiaomi:
		apiType = constant.APITypeXiaomi
	}
	if apiType == -1 {
		return constant.APITypeOpenAI, false
	}
	return apiType, true
}
