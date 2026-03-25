package relay

import (
	"math"
	"testing"

	"github.com/zhongruan0522/new-api/service"
)

func TestCalculateStreamSpeed(t *testing.T) {
	// 这些场景覆盖共享吐字速度计算中的正常路径与回退路径。
	tests := []struct {
		name                  string
		useTimeMs             int64
		frtMs                 int64
		completionTokens      int
		receivedResponseCount int
		wantSpeed             float64
		wantOK                bool
	}{
		{
			name:                  "normal stream uses generation window",
			useTimeMs:             3500,
			frtMs:                 2500,
			completionTokens:      82,
			receivedResponseCount: 8,
			wantSpeed:             82,
			wantOK:                true,
		},
		{
			name:                  "single flush falls back to total latency",
			useTimeMs:             18303,
			frtMs:                 18300,
			completionTokens:      1269,
			receivedResponseCount: 1,
			wantSpeed:             69.33289624651697,
			wantOK:                true,
		},
		{
			name:                  "abnormal spike falls back to total latency",
			useTimeMs:             21300,
			frtMs:                 21297,
			completionTokens:      1269,
			receivedResponseCount: 6,
			wantSpeed:             59.5774647887324,
			wantOK:                true,
		},
		{
			name:                  "invalid duration returns false",
			useTimeMs:             0,
			frtMs:                 0,
			completionTokens:      80,
			receivedResponseCount: 4,
			wantOK:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSpeed, gotOK := service.CalculateStreamSpeed(tt.useTimeMs, tt.frtMs, tt.completionTokens, tt.receivedResponseCount)
			if gotOK != tt.wantOK {
				t.Fatalf("CalculateStreamSpeed() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if math.Abs(gotSpeed-tt.wantSpeed) > 1e-6 {
				t.Fatalf("CalculateStreamSpeed() speed = %v, want %v", gotSpeed, tt.wantSpeed)
			}
		})
	}
}
