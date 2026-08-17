package controller

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/types"
	"github.com/gin-gonic/gin"
)

func TestShouldRetryNativeEmptyUsageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	apiErr := service.NewEmptyUsageRetryError(c, &relaycommon.RelayInfo{RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI}})

	if !shouldRetry(c, apiErr, 1) {
		t.Fatal("expected native empty usage error to enter automatic retry")
	}
}
