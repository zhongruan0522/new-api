package common

import (
	"testing"

	"github.com/zhongruan0522/new-api/constant"
)

func TestGetEndpointTypesByChannelTypeForOllama(t *testing.T) {
	endpoints := GetEndpointTypesByChannelType(constant.ChannelTypeOllama, "qwen3:8b")
	want := []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
		constant.EndpointTypeAnthropic,
	}

	if len(endpoints) != len(want) {
		t.Fatalf("endpoint count = %d, want %d", len(endpoints), len(want))
	}
	for i := range want {
		if endpoints[i] != want[i] {
			t.Fatalf("endpoint[%d] = %q, want %q", i, endpoints[i], want[i])
		}
	}
}
