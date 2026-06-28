package helper

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func responseModelName(info *relaycommon.RelayInfo) string {
	if info == nil {
		return ""
	}
	return info.GetResponseModelName()
}

func MaskTextResponseModel(response *dto.OpenAITextResponse, info *relaycommon.RelayInfo) {
	model := responseModelName(info)
	if response == nil || model == "" {
		return
	}
	response.Model = model
}

func MaskChatStreamResponseModel(response *dto.ChatCompletionsStreamResponse, info *relaycommon.RelayInfo) {
	model := responseModelName(info)
	if response == nil || model == "" {
		return
	}
	response.Model = model
}

func MaskResponsesResponseModel(response *dto.OpenAIResponsesResponse, info *relaycommon.RelayInfo) {
	model := responseModelName(info)
	if response == nil || model == "" {
		return
	}
	response.Model = model
}

func MaskResponsesStreamResponseModel(response *dto.ResponsesStreamResponse, info *relaycommon.RelayInfo) {
	if response == nil || response.Response == nil {
		return
	}
	MaskResponsesResponseModel(response.Response, info)
}

func MaskJSONModelPaths(data []byte, info *relaycommon.RelayInfo, paths ...string) []byte {
	model := responseModelName(info)
	if model == "" || len(data) == 0 || !gjson.ValidBytes(data) {
		return data
	}
	patched := string(data)
	for _, path := range paths {
		if path == "" || !gjson.Get(patched, path).Exists() {
			continue
		}
		updated, err := sjson.Set(patched, path, model)
		if err != nil {
			continue
		}
		patched = updated
	}
	return []byte(patched)
}

func MaskTopLevelModelJSON(data []byte, info *relaycommon.RelayInfo) []byte {
	return MaskJSONModelPaths(data, info, "model")
}

func MaskResponseEventModelJSON(data []byte, info *relaycommon.RelayInfo) []byte {
	return MaskJSONModelPaths(data, info, "model", "response.model")
}

func MaskClaudeEventModelJSON(data []byte, info *relaycommon.RelayInfo) []byte {
	return MaskJSONModelPaths(data, info, "model", "message.model")
}

func MaskRealtimeEventModelJSON(data []byte, info *relaycommon.RelayInfo) []byte {
	return MaskJSONModelPaths(data, info, "model", "response.model", "session.model")
}
