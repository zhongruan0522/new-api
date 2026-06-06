package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/types"
)

func TestShouldRetryNativeEmptyUsageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	apiErr := service.NewEmptyUsageRetryError(&relaycommon.RelayInfo{RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI}})

	if !shouldRetry(c, apiErr, 1) {
		t.Fatal("expected native empty usage error to enter automatic retry")
	}
}
