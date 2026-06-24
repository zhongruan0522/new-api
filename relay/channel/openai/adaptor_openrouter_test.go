package openai

import (
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func TestConvertOpenAIRequestOpenRouterThinkingEnabled(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "anthropic/claude-sonnet-4",
		THINKING: []byte(`{"type":"enabled","budget_tokens":2048}`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			UpstreamModelName: "anthropic/claude-sonnet-4",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error = %v", err)
	}
	convertedRequest := converted.(*dto.GeneralOpenAIRequest)
	if convertedRequest.THINKING != nil {
		t.Fatalf("THINKING = %s, want cleared after conversion", convertedRequest.THINKING)
	}

	var reasoning map[string]any
	if err := common.Unmarshal(convertedRequest.Reasoning, &reasoning); err != nil {
		t.Fatalf("unmarshal reasoning: %v", err)
	}
	if reasoning["enabled"] != true || reasoning["max_tokens"].(float64) != 2048 {
		t.Fatalf("reasoning = %#v, want enabled=true max_tokens=2048", reasoning)
	}
}

func TestConvertOpenAIRequestOpenRouterThinkingAdaptive(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "anthropic/claude-sonnet-4",
		THINKING: []byte(`{"type":"adaptive"}`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			UpstreamModelName: "anthropic/claude-sonnet-4",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error = %v", err)
	}
	convertedRequest := converted.(*dto.GeneralOpenAIRequest)

	var reasoning map[string]any
	if err := common.Unmarshal(convertedRequest.Reasoning, &reasoning); err != nil {
		t.Fatalf("unmarshal reasoning: %v", err)
	}
	if reasoning["enabled"] != true || len(reasoning) != 1 {
		t.Fatalf("reasoning = %#v, want only enabled=true", reasoning)
	}
}
