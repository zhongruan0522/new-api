package channelcontroller

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolveChannelTestUserIDUsesRequestUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("id", 42)

	userID, err := resolveChannelTestUserID(ctx)
	if err != nil {
		t.Fatalf("resolveChannelTestUserID returned error: %v", err)
	}

	if userID != 42 {
		t.Fatalf("resolveChannelTestUserID returned %d, want request user 42", userID)
	}
}
