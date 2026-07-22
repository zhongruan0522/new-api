package common

import (
	"io"
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

func TestUnmarshalBodyReusableStreamsDiskBackedJSON(t *testing.T) {
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

	body := `{"prompt":"` + strings.Repeat("x", 4096) + `","n":1}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.ContentLength = int64(len(body))

	var payload struct {
		Prompt string `json:"prompt"`
		N      int    `json:"n"`
	}
	if err := UnmarshalBodyReusable(c, &payload); err != nil {
		t.Fatalf("UnmarshalBodyReusable returned error: %v", err)
	}
	if payload.N != 1 || payload.Prompt != strings.Repeat("x", 4096) {
		t.Fatal("UnmarshalBodyReusable decoded unexpected payload")
	}
	if _, exists := c.Get(KeyRequestBody); exists {
		t.Fatal("disk-backed JSON was materialized into legacy byte cache")
	}
	reusableBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read reusable request body: %v", err)
	}
	if string(reusableBody) != body {
		t.Fatal("request body was not reset after disk-backed JSON decode")
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

	for b.Loop() {
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
