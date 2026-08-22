package common

import ()

import "github.com/NookMux/NookMux/internal/domain/channel/constant"

// GetEndpointTypesByChannelType 获取渠道最优先端点类型（所有的渠道都支持 OpenAI 端点）
func GetEndpointTypesByChannelType(channelType int, modelName string) []constant.EndpointType {
	var endpointTypes []constant.EndpointType
	switch channelType {
	case constant.ChannelTypeAws:
		fallthrough
	case constant.ChannelTypeAnthropic:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeAnthropic, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeVertexAi:
		fallthrough
	case constant.ChannelTypeGemini:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeGemini, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeOllama:
		endpointTypes = []constant.EndpointType{
			constant.EndpointTypeOpenAI,
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeAnthropic,
		}
	case constant.ChannelTypeOpenRouter:
		// OpenRouter 原生提供 OpenAI Chat、OpenAI Responses 与 Anthropic Messages 兼容端点。
		endpointTypes = []constant.EndpointType{
			constant.EndpointTypeOpenAI,
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeAnthropic,
		}
	default:
		if IsOpenAIResponseOnlyModel(modelName) {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIResponse}
		} else {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
		}
	}
	if IsImageGenerationModel(modelName) {
		endpointTypes = append([]constant.EndpointType{constant.EndpointTypeImageGeneration}, endpointTypes...)
	}
	return endpointTypes
}
