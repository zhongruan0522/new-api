package common

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/constant"
)

func TestGetRequestBodyDoesNotKeepLegacyBytesForDiskStorage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldConfig := GetDiskCacheConfig()
	oldMaxRequestBodyMB := constant.MaxRequestBodyMB
	SetDiskCacheConfig(DiskCacheConfig{
		Enabled:     true,
		ThresholdMB: 0,
		MaxSizeMB:   1,
		Path:        t.TempDir(),
	})
	constant.MaxRequestBodyMB = 1
	t.Cleanup(func() {
		SetDiskCacheConfig(oldConfig)
		constant.MaxRequestBodyMB = oldMaxRequestBodyMB
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := strings.Repeat("x", 4096)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	c.Request.ContentLength = int64(len(body))

	got, err := GetRequestBody(c)
	if err != nil {
		t.Fatalf("GetRequestBody returned error: %v", err)
	}
	if string(got) != body {
		t.Fatal("GetRequestBody returned unexpected body")
	}
	if _, exists := c.Get(KeyRequestBody); exists {
		t.Fatal("disk-backed body was also stored in legacy byte cache")
	}

	again, err := GetRequestBody(c)
	if err != nil {
		t.Fatalf("second GetRequestBody returned error: %v", err)
	}
	if string(again) != body {
		t.Fatal("second GetRequestBody returned unexpected body")
	}
	CleanupBodyStorage(c)
}

func BenchmarkGetRequestBodyDiskStorage(b *testing.B) {
	gin.SetMode(gin.TestMode)

	oldConfig := GetDiskCacheConfig()
	oldMaxRequestBodyMB := constant.MaxRequestBodyMB
	SetDiskCacheConfig(DiskCacheConfig{
		Enabled:     true,
		ThresholdMB: 0,
		MaxSizeMB:   64,
		Path:        b.TempDir(),
	})
	constant.MaxRequestBodyMB = 1
	b.Cleanup(func() {
		SetDiskCacheConfig(oldConfig)
		constant.MaxRequestBodyMB = oldMaxRequestBodyMB
	})

	body := strings.Repeat("x", 64<<10)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
		c.Request.ContentLength = int64(len(body))

		if _, err := GetRequestBody(c); err != nil {
			b.Fatalf("GetRequestBody returned error: %v", err)
		}
		if _, exists := c.Get(KeyRequestBody); exists {
			b.Fatal("disk-backed body was also stored in legacy byte cache")
		}
		CleanupBodyStorage(c)
	}
}
