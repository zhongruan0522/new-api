package zhipu_4v

import (
	"testing"

	"github.com/NookMux/NookMux/internal/constant"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/internal/types"
)

func zhipuInfo(baseURL string, relayMode int, relayFormat types.RelayFormat) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeZhipu_v4,
			ChannelBaseUrl: baseURL,
		},
		RelayMode:   relayMode,
		RelayFormat: relayFormat,
	}
}

// 智谱 GLM 套餐的 OpenAI Responses 协议端点独立于 Chat Completions 端点：
// 国内为 https://open.bigmodel.cn/api/v1，国际为 https://api.z.ai/api/v1。
// 详见 https://docs.bigmodel.cn/cn/coding-plan/quick-start 与
// https://docs.z.ai/devpack/quick-start 的接入端点说明。
func TestGetRequestURL_PlanResponsesUsesDedicatedResponsesEndpoint(t *testing.T) {
	cases := []struct {
		name    string
		plan    string
		wantURL string
	}{
		{
			name:    "glm-coding-plan domestic",
			plan:    "glm-coding-plan",
			wantURL: "https://open.bigmodel.cn/api/v1/responses",
		},
		{
			name:    "glm-coding-plan international",
			plan:    "glm-coding-plan-international",
			wantURL: "https://api.z.ai/api/v1/responses",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := zhipuInfo(tc.plan, relayconstant.RelayModeResponses, types.RelayFormatOpenAIResponses)

			got, err := (&Adaptor{}).GetRequestURL(info)
			if err != nil {
				t.Fatalf("GetRequestURL error = %v", err)
			}
			if got != tc.wantURL {
				t.Fatalf("responses URL = %q, want %q", got, tc.wantURL)
			}
		})
	}
}

// 回归保护：套餐渠道的 Chat Completions 仍走 coding/paas/v4 端点。
func TestGetRequestURL_PlanChatCompletionsUnchanged(t *testing.T) {
	cases := []struct {
		name    string
		plan    string
		wantURL string
	}{
		{
			name:    "glm-coding-plan domestic",
			plan:    "glm-coding-plan",
			wantURL: "https://open.bigmodel.cn/api/coding/paas/v4/chat/completions",
		},
		{
			name:    "glm-coding-plan international",
			plan:    "glm-coding-plan-international",
			wantURL: "https://api.z.ai/api/coding/paas/v4/chat/completions",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := zhipuInfo(tc.plan, relayconstant.RelayModeChatCompletions, types.RelayFormatOpenAI)

			got, err := (&Adaptor{}).GetRequestURL(info)
			if err != nil {
				t.Fatalf("GetRequestURL error = %v", err)
			}
			if got != tc.wantURL {
				t.Fatalf("chat URL = %q, want %q", got, tc.wantURL)
			}
		})
	}
}

// 回归保护：套餐渠道的 Claude 协议仍走 api/anthropic 端点。
func TestGetRequestURL_PlanClaudeUnchanged(t *testing.T) {
	cases := []struct {
		name    string
		plan    string
		wantURL string
	}{
		{
			name:    "glm-coding-plan domestic",
			plan:    "glm-coding-plan",
			wantURL: "https://open.bigmodel.cn/api/anthropic/v1/messages",
		},
		{
			name:    "glm-coding-plan international",
			plan:    "glm-coding-plan-international",
			wantURL: "https://api.z.ai/api/anthropic/v1/messages",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := zhipuInfo(tc.plan, relayconstant.RelayModeChatCompletions, types.RelayFormatClaude)

			got, err := (&Adaptor{}).GetRequestURL(info)
			if err != nil {
				t.Fatalf("GetRequestURL error = %v", err)
			}
			if got != tc.wantURL {
				t.Fatalf("claude URL = %q, want %q", got, tc.wantURL)
			}
		})
	}
}

// 回归保护：非套餐的普通智谱渠道仍走通用 paas/v4 端点。
func TestGetRequestURL_NonPlanChannelUnchanged(t *testing.T) {
	info := zhipuInfo("https://open.bigmodel.cn", relayconstant.RelayModeChatCompletions, types.RelayFormatOpenAI)

	got, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	want := "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	if got != want {
		t.Fatalf("non-plan chat URL = %q, want %q", got, want)
	}
}

// ResponsesBaseURL 与渠道 openai_wire_api 设置正交：
// 管理员把 openai_wire_api 配置为 chat 时，OpenAIWireHelper 会把下游的
// /v1/responses 请求转换成 Chat 请求（info.RelayMode 被改写为
// RelayModeChatCompletions），此时 GetRequestURL 必须仍走 coding/paas/v4
// 的 chat/completions 端点，不能因为存在 ResponsesBaseURL 就改路由。
// 本测试固化该语义：URL 选择只跟随转换后的 RelayMode。
func TestGetRequestURL_PlanWireChatConversionStillUsesChatEndpoint(t *testing.T) {
	// 模拟 relayResponsesDownstreamToChatUpstream 转换后的状态：
	// 下游是 /v1/responses 请求，但已被改写为 Chat 上游。
	info := zhipuInfo("glm-coding-plan", relayconstant.RelayModeChatCompletions, types.RelayFormatOpenAI)
	info.RequestURLPath = "/v1/chat/completions"

	got, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	want := "https://open.bigmodel.cn/api/coding/paas/v4/chat/completions"
	if got != want {
		t.Fatalf("wire=chat converted URL = %q, want %q", got, want)
	}
}

// ResponsesBaseURL 缺失（其他套餐未提供独立端点）时回退到 OpenAIBaseURL，
// 与既有行为兼容。xiaomi-coding-plan 在 ChannelSpecialBases 中只有
// OpenAIBaseURL，可用于验证该回退路径。
func TestGetRequestURL_PlanWithoutResponsesBaseFallsBackToOpenAIBaseURL(t *testing.T) {
	info := zhipuInfo("xiaomi-coding-plan", relayconstant.RelayModeResponses, types.RelayFormatOpenAIResponses)

	got, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	want := "https://token-plan-cn.xiaomimimo.com/v1/responses"
	if got != want {
		t.Fatalf("fallback responses URL = %q, want %q", got, want)
	}
}
