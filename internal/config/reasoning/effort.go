package reasoning

import (
	"strings"

	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/relay/channel/openrouter"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

// 已知的 effort 级别，跨厂商统一词表。
// OpenAI:  none, minimal, low, medium, high, xhigh, max
// Claude:  low, medium, high, xhigh, max
// Gemini:  minimal, low, medium, high
var knownEfforts = map[string]bool{
	"none":    true,
	"minimal": true,
	"low":     true,
	"medium":  true,
	"high":    true,
	"xhigh":   true,
	"max":     true,
}

// NormalizeEffort 小写化、去空白后校验 effort 值。
// 返回归一化后的值；未知值返回空字符串。
func NormalizeEffort(raw string) string {
	e := strings.ToLower(strings.TrimSpace(raw))
	if knownEfforts[e] {
		return e
	}
	return ""
}

// IsThinkingEnabled 判断 Claude/Gemini/通用 thinking 参数是否表示"启用了思考"。
func IsThinkingEnabled(thinkingType string) bool {
	t := strings.ToLower(strings.TrimSpace(thinkingType))
	return t == "enabled" || t == "adaptive"
}

// ExtractEffortFromOpenAIRequest 从 OpenAI 兼容请求中解析 reasoning effort。
// 覆盖以下来源（优先级从高到低）：
//  1. request.ReasoningEffort（顶层 reasoning_effort）
//  2. request.Reasoning.effort（OpenRouter / Responses 风格）
//  3. request.THINKING（zhipu_v4 等透传的 Claude 风格 thinking，含 type=enabled/adaptive）
//  4. request.EnableThinking（Ali Qwen 等透传，布尔语义：启用时无显式 effort）
func ExtractEffortFromOpenAIRequest(request *shared.GeneralOpenAIRequest) string {
	if request == nil {
		return ""
	}

	// 1. 顶层 reasoning_effort
	if effort := NormalizeEffort(request.ReasoningEffort); effort != "" {
		return effort
	}

	// 2. Reasoning JSON (OpenRouter / Responses)
	if len(request.Reasoning) > 0 {
		var rr openrouter.RequestReasoning
		if err := jsonx.Unmarshal(request.Reasoning, &rr); err == nil {
			if effort := NormalizeEffort(rr.Effort); effort != "" {
				return effort
			}
			// Reasoning.enabled=true 但无 effort -> 启用了思考但没传强度
			if rr.Enabled {
				return "auto"
			}
		}
	}

	// 3. THINKING (zhipu_v4 / Claude-style passthrough)
	if len(request.THINKING) > 0 {
		var thinking shared.Thinking
		if err := jsonx.Unmarshal(request.THINKING, &thinking); err == nil {
			if IsThinkingEnabled(thinking.Type) {
				return "auto"
			}
		}
	}

	// 4. EnableThinking (Ali Qwen passthrough)
	if len(request.EnableThinking) > 0 {
		var enabled bool
		if err := jsonx.Unmarshal(request.EnableThinking, &enabled); err == nil && enabled {
			return "auto"
		}
	}

	return ""
}

// ExtractEffortFromOpenAIResponsesRequest 从 OpenAI Responses API 请求中解析 reasoning effort。
// 覆盖以下来源：
//  1. request.Reasoning.Effort（reasoning.effort）
//  2. request.EnableThinking（Ali Qwen 等透传）
func ExtractEffortFromOpenAIResponsesRequest(request *shared.OpenAIResponsesRequest) string {
	if request == nil {
		return ""
	}

	// 1. reasoning.effort
	if request.Reasoning != nil {
		if effort := NormalizeEffort(request.Reasoning.Effort); effort != "" {
			return effort
		}
	}

	// 2. EnableThinking (Ali Qwen passthrough)
	if len(request.EnableThinking) > 0 {
		var enabled bool
		if err := jsonx.Unmarshal(request.EnableThinking, &enabled); err == nil && enabled {
			return "auto"
		}
	}

	return ""
}

// ExtractEffortFromClaudeRequest 从 Claude Messages 请求中解析 reasoning effort。
// 覆盖以下来源（优先级从高到低）：
//  1. output_config.effort（Claude 官方 effort 参数）
//  2. thinking.type = enabled/adaptive（启用了思考但无显式 effort）
func ExtractEffortFromClaudeRequest(request *shared.ClaudeRequest) string {
	if request == nil {
		return ""
	}

	// 1. output_config.effort
	if len(request.OutputConfig) > 0 {
		var config shared.ClaudeOutputConfig
		if err := jsonx.Unmarshal(request.OutputConfig, &config); err == nil {
			// Claude effort "max" 在本项目统一映射为 "xhigh"
			if strings.EqualFold(config.Effort, "max") {
				return "xhigh"
			}
			if effort := NormalizeEffort(config.Effort); effort != "" {
				return effort
			}
		}
	}

	// 2. thinking.type = enabled / adaptive
	if request.Thinking != nil && IsThinkingEnabled(request.Thinking.Type) {
		return "auto"
	}

	return ""
}

// ExtractEffortFromGeminiRequest 从 Gemini 请求中解析 reasoning effort。
// 覆盖以下来源（优先级从高到低）：
//  1. thinkingConfig.thinkingLevel（Gemini 3+ 的 level 参数）
//  2. thinkingConfig.thinkingBudget > 0 或 IncludeThoughts=true（启用了思考但无显式 level）
func ExtractEffortFromGeminiRequest(request *shared.GeminiChatRequest) string {
	if request == nil {
		return ""
	}

	tc := request.GenerationConfig.ThinkingConfig
	if tc == nil {
		return ""
	}

	// 1. thinkingLevel
	if effort := NormalizeEffort(tc.ThinkingLevel); effort != "" {
		return effort
	}

	// 2. IncludeThoughts=true 或有 ThinkingBudget
	if tc.IncludeThoughts {
		return "auto"
	}
	if tc.ThinkingBudget != nil && *tc.ThinkingBudget != 0 {
		return "auto"
	}

	return ""
}
