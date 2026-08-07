package common

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/NookMux/NookMux/constant"
)

func TestGenRelayInfoOpenAI_StreamOptionsByChannelType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		channel int
		want    bool
	}{
		{"moonshot", constant.ChannelTypeMoonshot, true},
		{"minimax", constant.ChannelTypeMiniMax, true},
		{"siliconflow", constant.ChannelTypeSiliconFlow, true},
		{"openrouter_not_listed", constant.ChannelTypeOpenRouter, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
			c.Set(string(constant.ContextKeyChannelType), tc.channel)

			info := GenRelayInfoOpenAI(c, nil)
			info.InitChannelMeta(c)
			if info.ChannelMeta == nil {
				t.Fatal("ChannelMeta is nil")
			}
			if info.ChannelMeta.SupportStreamOptions != tc.want {
				t.Fatalf("SupportStreamOptions = %v, want %v for channel %d", info.ChannelMeta.SupportStreamOptions, tc.want, tc.channel)
			}
		})
	}
}
