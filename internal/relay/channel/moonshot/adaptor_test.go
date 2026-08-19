package moonshot

import (
	"testing"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
)

func TestConvertOpenAIRequestKimiK26UsesOnlyAllowedTemperature(t *testing.T) {
	request := &shared.GeneralOpenAIRequest{
		Model:       "kimi-k2.6",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error: %v", err)
	}
	convertedRequest, ok := converted.(*shared.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("converted request type = %T, want *shared.GeneralOpenAIRequest", converted)
	}
	if convertedRequest.Temperature == nil || *convertedRequest.Temperature != 1.0 {
		t.Fatalf("temperature = %v, want 1.0", convertedRequest.Temperature)
	}
}

func TestConvertOpenAIRequestKimiK26KeepsOmittedTemperatureOmitted(t *testing.T) {
	request := &shared.GeneralOpenAIRequest{
		Model: "kimi-k2.6",
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error: %v", err)
	}
	convertedRequest, ok := converted.(*shared.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("converted request type = %T, want *shared.GeneralOpenAIRequest", converted)
	}
	if convertedRequest.Temperature != nil {
		t.Fatalf("temperature = %v, want nil", *convertedRequest.Temperature)
	}
}

func TestConvertOpenAIRequestOtherMoonshotModelKeepsTemperature(t *testing.T) {
	request := &shared.GeneralOpenAIRequest{
		Model:       "kimi-k2.5",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.5",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertOpenAIRequest error: %v", err)
	}
	convertedRequest, ok := converted.(*shared.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("converted request type = %T, want *shared.GeneralOpenAIRequest", converted)
	}
	if convertedRequest.Temperature == nil || *convertedRequest.Temperature != 0.7 {
		t.Fatalf("temperature = %v, want 0.7", convertedRequest.Temperature)
	}
}
