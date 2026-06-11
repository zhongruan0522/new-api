package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/setting"

	"github.com/gin-gonic/gin"
)

func TestPlaygroundRequestContextRejectsAccessTokenBeforeNext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	router := gin.New()
	router.POST("/pg/chat/completions",
		func(c *gin.Context) {
			c.Set("id", 7)
			c.Set("use_access_token", true)
			c.Next()
		},
		PlaygroundRequestContext(),
		func(c *gin.Context) {
			called = true
			c.JSON(http.StatusOK, gin.H{"success": true})
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/pg/chat/completions", strings.NewReader(`{"model":"gpt-4o","group":"default","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if called {
		t.Fatal("next handler was called for playground access token request")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "暂不支持使用 access token") {
		t.Fatalf("expected access token rejection, got: %s", recorder.Body.String())
	}
}

func TestPlaygroundSelectedGroupAppliesBeforeModelRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldRedisEnabled := common.RedisEnabled
	oldRateLimitEnabled := setting.ModelRequestRateLimitEnabled
	oldDuration := setting.ModelRequestRateLimitDurationMinutes
	oldTotalCount := setting.ModelRequestRateLimitCount
	oldSuccessCount := setting.ModelRequestRateLimitSuccessCount
	setting.ModelRequestRateLimitMutex.RLock()
	oldGroupLimits := setting.ModelRequestRateLimitGroup
	setting.ModelRequestRateLimitMutex.RUnlock()

	common.RedisEnabled = false
	setting.ModelRequestRateLimitEnabled = true
	setting.ModelRequestRateLimitDurationMinutes = 1
	setting.ModelRequestRateLimitCount = 0
	setting.ModelRequestRateLimitSuccessCount = 1000
	setting.ModelRequestRateLimitMutex.Lock()
	setting.ModelRequestRateLimitGroup = map[string][2]int{
		"vip": {1, 1000},
	}
	setting.ModelRequestRateLimitMutex.Unlock()
	inMemoryRateLimiter = common.InMemoryRateLimiter{}

	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		setting.ModelRequestRateLimitEnabled = oldRateLimitEnabled
		setting.ModelRequestRateLimitDurationMinutes = oldDuration
		setting.ModelRequestRateLimitCount = oldTotalCount
		setting.ModelRequestRateLimitSuccessCount = oldSuccessCount
		setting.ModelRequestRateLimitMutex.Lock()
		setting.ModelRequestRateLimitGroup = oldGroupLimits
		setting.ModelRequestRateLimitMutex.Unlock()
		inMemoryRateLimiter = common.InMemoryRateLimiter{}
	})

	router := gin.New()
	router.POST("/pg/chat/completions",
		func(c *gin.Context) {
			c.Set("id", 42)
			common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
			common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
			c.Next()
		},
		PlaygroundRequestContext(),
		ModelRequestRateLimit(),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"group": common.GetContextKeyString(c, constant.ContextKeyTokenGroup),
			})
		},
	)

	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodPost, "/pg/chat/completions", strings.NewReader(`{"model":"gpt-4o","group":"vip","messages":[]}`))
	firstReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(first, firstReq)

	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d, body = %s", first.Code, http.StatusOK, first.Body.String())
	}
	if !strings.Contains(first.Body.String(), `"group":"vip"`) {
		t.Fatalf("selected playground group was not written before rate limit, body = %s", first.Body.String())
	}

	second := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodPost, "/pg/chat/completions", strings.NewReader(`{"model":"gpt-4o","group":"vip","messages":[]}`))
	secondReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(second, secondReq)

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d, body = %s", second.Code, http.StatusTooManyRequests, second.Body.String())
	}
}
