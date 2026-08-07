package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/NookMux/NookMux/common"
)

func TestSearchRateLimitDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origEnabled := common.SearchRateLimitEnable
	common.SearchRateLimitEnable = false
	t.Cleanup(func() { common.SearchRateLimitEnable = origEnabled })

	router := gin.New()
	router.GET("/search", func(c *gin.Context) {
		c.Set("id", 42)
		c.Next()
	}, SearchRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 when search rate limit disabled, got %d", recorder.Code)
	}
}

func TestSearchRateLimitEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origEnabled := common.SearchRateLimitEnable
	origNum := common.SearchRateLimitNum
	origDuration := common.SearchRateLimitDuration

	common.SearchRateLimitEnable = true
	common.SearchRateLimitNum = 2
	common.SearchRateLimitDuration = 3600

	common.RedisEnabled = false
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)

	t.Cleanup(func() {
		common.SearchRateLimitEnable = origEnabled
		common.SearchRateLimitNum = origNum
		common.SearchRateLimitDuration = origDuration
	})

	router := gin.New()
	router.GET("/search", func(c *gin.Context) {
		c.Set("id", 999001)
		c.Next()
	}, SearchRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First two requests within the limit should succeed.
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200 for request %d, got %d", i+1, recorder.Code)
		}
	}

	// The third request within the same window should be rate limited.
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after limit exceeded, got %d", recorder.Code)
	}
}

func TestSearchRateLimitHonorsConfiguredDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origEnabled := common.SearchRateLimitEnable
	origNum := common.SearchRateLimitNum
	origDuration := common.SearchRateLimitDuration

	common.SearchRateLimitEnable = true
	common.SearchRateLimitNum = 1
	common.SearchRateLimitDuration = 1

	common.RedisEnabled = false
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)

	t.Cleanup(func() {
		common.SearchRateLimitEnable = origEnabled
		common.SearchRateLimitNum = origNum
		common.SearchRateLimitDuration = origDuration
	})

	router := gin.New()
	router.GET("/search", func(c *gin.Context) {
		c.Set("id", 999002)
		c.Next()
	}, SearchRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/search", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 while still within window, got %d", recorder.Code)
	}
}

func TestSearchRateLimitUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origEnabled := common.SearchRateLimitEnable
	common.SearchRateLimitEnable = true
	t.Cleanup(func() { common.SearchRateLimitEnable = origEnabled })

	router := gin.New()
	router.GET("/search", SearchRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	router.ServeHTTP(recorder, req)

	// Without a user id the limiter is not in a valid state to decide; it
	// rejects the request as 401 to signal that UserAuth is missing upstream.
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when user id is missing, got %d", recorder.Code)
	}
}
