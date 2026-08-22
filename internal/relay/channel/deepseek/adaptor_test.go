package deepseek

import (
	"encoding/json"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

func TestConvertOpenAIRequestAppliesDeepSeekV4MaxSuffix(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeDeepSeek,
			UpstreamModelName: "deepseek-v4-pro-max",
		},
	}
	request := &shared.GeneralOpenAIRequest{Model: "deepseek-v4-pro-max"}

	convertedAny, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error = %v", err)
	}
	converted := convertedAny.(*shared.GeneralOpenAIRequest)

	if converted.Model != "deepseek-v4-pro" {
		t.Fatalf("model = %q, want deepseek-v4-pro", converted.Model)
	}
	if converted.ReasoningEffort != "max" {
		t.Fatalf("reasoning_effort = %q, want max", converted.ReasoningEffort)
	}
	var thinking map[string]string
	if err := jsonx.Unmarshal(converted.THINKING, &thinking); err != nil {
		t.Fatalf("unmarshal thinking error = %v", err)
	}
	if thinking["type"] != "enabled" {
		t.Fatalf("thinking.type = %q, want enabled", thinking["type"])
	}
	if info.UpstreamModelName != "deepseek-v4-pro" {
		t.Fatalf("info.UpstreamModelName = %q, want deepseek-v4-pro", info.UpstreamModelName)
	}
	if info.ReasoningEffort != "max" {
		t.Fatalf("info.ReasoningEffort = %q, want max", info.ReasoningEffort)
	}
}

func TestConvertOpenAIRequestAppliesDeepSeekV4NoneSuffix(t *testing.T) {
	request := &shared.GeneralOpenAIRequest{Model: "deepseek-v4-flash-none"}

	convertedAny, err := (&Adaptor{}).ConvertOpenAIRequest(nil, nil, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error = %v", err)
	}
	converted := convertedAny.(*shared.GeneralOpenAIRequest)

	if converted.Model != "deepseek-v4-flash" {
		t.Fatalf("model = %q, want deepseek-v4-flash", converted.Model)
	}
	if converted.ReasoningEffort != "" {
		t.Fatalf("reasoning_effort = %q, want empty for disabled thinking", converted.ReasoningEffort)
	}
	var thinking map[string]string
	if err := jsonx.Unmarshal(converted.THINKING, &thinking); err != nil {
		t.Fatalf("unmarshal thinking error = %v", err)
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking.type = %q, want disabled", thinking["type"])
	}
}

func TestConvertClaudeRequestAppliesDeepSeekV4MaxSuffix(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeDeepSeek,
			UpstreamModelName: "deepseek-v4-pro-max",
		},
	}
	request := &shared.ClaudeRequest{Model: "deepseek-v4-pro-max"}

	convertedAny, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertClaudeRequest error = %v", err)
	}
	converted := convertedAny.(*shared.ClaudeRequest)

	if converted.Model != "deepseek-v4-pro" {
		t.Fatalf("model = %q, want deepseek-v4-pro", converted.Model)
	}
	if converted.Thinking == nil || converted.Thinking.Type != "enabled" {
		t.Fatalf("thinking = %+v, want enabled", converted.Thinking)
	}
	var outputConfig shared.ClaudeOutputConfig
	if err := jsonx.Unmarshal(converted.OutputConfig, &outputConfig); err != nil {
		t.Fatalf("unmarshal output_config error = %v", err)
	}
	if outputConfig.Effort != "max" {
		t.Fatalf("output_config.effort = %q, want max", outputConfig.Effort)
	}
	if info.UpstreamModelName != "deepseek-v4-pro" {
		t.Fatalf("info.UpstreamModelName = %q, want deepseek-v4-pro", info.UpstreamModelName)
	}
}

func TestConvertClaudeRequestAppliesDeepSeekV4NoneSuffix(t *testing.T) {
	request := &shared.ClaudeRequest{
		Model:        "deepseek-v4-flash-none",
		OutputConfig: json.RawMessage(`{"effort":"max"}`),
	}

	convertedAny, err := (&Adaptor{}).ConvertClaudeRequest(nil, nil, request)
	if err != nil {
		t.Fatalf("ConvertClaudeRequest error = %v", err)
	}
	converted := convertedAny.(*shared.ClaudeRequest)

	if converted.Model != "deepseek-v4-flash" {
		t.Fatalf("model = %q, want deepseek-v4-flash", converted.Model)
	}
	if converted.Thinking == nil || converted.Thinking.Type != "disabled" {
		t.Fatalf("thinking = %+v, want disabled", converted.Thinking)
	}
	if len(converted.OutputConfig) != 0 {
		t.Fatalf("output_config = %s, want empty when thinking is disabled", string(converted.OutputConfig))
	}
}
