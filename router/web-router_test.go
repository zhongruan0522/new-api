package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestAcceptsBrotli(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		want           bool
	}{
		{
			name:           "prefers brotli",
			acceptEncoding: "gzip, br",
			want:           true,
		},
		{
			name:           "skips disabled brotli",
			acceptEncoding: "br;q=0, gzip",
			want:           false,
		},
		{
			name:           "allows weighted brotli",
			acceptEncoding: "gzip, br;q=0.8",
			want:           true,
		},
		{
			name:           "skips zero decimal brotli",
			acceptEncoding: "gzip, br;q=0.0",
			want:           false,
		},
		{
			name:           "empty",
			acceptEncoding: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := acceptsBrotli(tt.acceptEncoding); got != tt.want {
				t.Fatalf("acceptsBrotli() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStaticAssetCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(staticAssetCache())
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/static/js/app.js", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got := resp.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable static asset cache", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got := resp.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("API Cache-Control = %q, want empty", got)
	}
}

func TestPrecompressedStaticAssetsServesBrotli(t *testing.T) {
	gin.SetMode(gin.TestMode)

	compressed := []byte("pretend-brotli-compressed")

	buildFS := fstest.MapFS{
		"web/dist/static/js/app.js.br": {
			Data: compressed,
		},
	}

	router := gin.New()
	router.Use(staticAssetCache())
	router.Use(precompressedStaticAssetsFS(buildFS, "web/dist"))
	router.GET("/*path", func(c *gin.Context) {
		c.String(http.StatusOK, "fallback")
	})

	req := httptest.NewRequest(http.MethodGet, "/static/js/app.js", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if got := resp.Header().Get("Content-Encoding"); got != "br" {
		t.Fatalf("Content-Encoding = %q, want br", got)
	}
	if got := resp.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Fatalf("Vary = %q, want Accept-Encoding", got)
	}
	if got := resp.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable static asset cache", got)
	}

	if got := resp.Body.Bytes(); !bytes.Equal(got, compressed) {
		t.Fatalf("body = %q, want %q", got, compressed)
	}
}

func TestPrecompressedStaticAssetsFallsThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(precompressedStaticAssetsFS(fstest.MapFS{}, "web/dist"))
	router.GET("/*path", func(c *gin.Context) {
		c.String(http.StatusOK, "fallback")
	})

	req := httptest.NewRequest(http.MethodGet, "/static/js/missing.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got := resp.Body.String(); got != "fallback" {
		t.Fatalf("body = %q, want fallback", got)
	}
	if got := resp.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}
