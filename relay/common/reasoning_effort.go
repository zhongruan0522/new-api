package common

import (
	"github.com/zhongruan0522/new-api/dto"
	"github.com/zhongruan0522/new-api/setting/reasoning"
)

// EnsureReasoningEffort 确保 RelayInfo.ReasoningEffort 被正确解析和回填。
//
// 该函数作为各 adaptor 特定解析逻辑之后的通用兜底：
// 如果 adaptor 已经设置了 ReasoningEffort（如 OpenAI o系列/gpt-5、Claude、DeepSeek），
// 则保持 adaptor 的值不变；否则根据请求中携带的思维强度参数通用解析。
//
// 启用了思考但未传递强度时，回填 "auto"。
//
// 该函数不修改请求体，只读取请求中的 reasoning/thinking 相关字段。
func EnsureReasoningEffort(info *RelayInfo, request dto.Request) {
	if info == nil || request == nil {
		return
	}

	// adaptor 已经设置了 ReasoningEffort，保持不变
	if info.ReasoningEffort != "" {
		return
	}

	var effort string
	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		effort = reasoning.ExtractEffortFromOpenAIRequest(req)
	case *dto.OpenAIResponsesRequest:
		effort = reasoning.ExtractEffortFromOpenAIResponsesRequest(req)
	case *dto.ClaudeRequest:
		effort = reasoning.ExtractEffortFromClaudeRequest(req)
	case *dto.GeminiChatRequest:
		effort = reasoning.ExtractEffortFromGeminiRequest(req)
	}

	if effort != "" {
		info.ReasoningEffort = effort
	}
}
