package service

import (
	"math"
	"testing"
	"time"

	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func TestAppendStreamMetrics(t *testing.T) {
	startTime := time.Unix(0, 0)
	relayInfo := &relaycommon.RelayInfo{
		IsStream:              true,
		StartTime:             startTime,
		FirstResponseTime:     startTime.Add(2500 * time.Millisecond),
		ReceivedResponseCount: 8,
	}

	other := map[string]interface{}{}
	AppendStreamMetrics(other, relayInfo, 3500, 82)

	speed, ok := other["speed"].(float64)
	if !ok {
		t.Fatalf("AppendStreamMetrics() did not record speed: %#v", other)
	}
	if math.Abs(speed-82) > 1e-6 {
		t.Fatalf("AppendStreamMetrics() speed = %v, want %v", speed, 82.0)
	}
}

func TestAppendStreamMetricsSkipsNonStreamRequests(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{IsStream: false}
	other := map[string]interface{}{}

	// 非流式请求不应生成吐字速度，避免日志语义混淆。
	AppendStreamMetrics(other, relayInfo, 3500, 82)

	if _, exists := other["speed"]; exists {
		t.Fatalf("AppendStreamMetrics() unexpectedly recorded speed: %#v", other)
	}
}
