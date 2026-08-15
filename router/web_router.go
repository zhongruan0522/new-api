package router

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/controller"
	"github.com/NookMux/NookMux/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

type WebAssets struct {
	BuildFS   embed.FS
	IndexPage []byte
}

func SetWebRouter(router *gin.Engine, assets WebAssets) {
	webFS := common.EmbedFolder(assets.BuildFS, "web/dist")

	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(staticAssetCache())
	router.Use(precompressedStaticAssets(assets.BuildFS))
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(static.Serve("/", webFS))
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
	})
}

func staticAssetCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") || c.Request.URL.Path == "/favicon.ico" {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	}
}

func precompressedStaticAssets(buildFS embed.FS) gin.HandlerFunc {
	return precompressedStaticAssetsFS(buildFS, "web/dist")
}

func precompressedStaticAssetsFS(buildFS fs.FS, root string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			return
		}
		if !strings.HasPrefix(c.Request.URL.Path, "/static/") {
			return
		}

		if !acceptsBrotli(c.GetHeader("Accept-Encoding")) {
			return
		}

		assetPath := path.Clean(strings.TrimPrefix(c.Request.URL.Path, "/"))
		if assetPath == "." || strings.HasPrefix(assetPath, "../") || strings.Contains(assetPath, "/../") || !strings.HasPrefix(assetPath, "static/") {
			return
		}

		compressedPath := path.Join(root, assetPath+".br")
		file, err := buildFS.Open(compressedPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return
			}
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		seeker, ok := file.(io.ReadSeeker)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		contentType := mime.TypeByExtension(path.Ext(assetPath))
		if contentType != "" {
			c.Header("Content-Type", contentType)
		}
		c.Header("Content-Encoding", "br")
		c.Header("Vary", "Accept-Encoding")
		http.ServeContent(c.Writer, c.Request, path.Base(assetPath), stat.ModTime(), seeker)
		c.Abort()
	}
}

func acceptsBrotli(acceptEncoding string) bool {
	for _, value := range strings.Split(acceptEncoding, ",") {
		token, allowed := parseAcceptEncodingToken(value)
		if !allowed {
			continue
		}
		if token == "br" {
			return true
		}
	}
	return false
}

func parseAcceptEncodingToken(value string) (string, bool) {
	parts := strings.Split(value, ";")
	token := strings.ToLower(strings.TrimSpace(parts[0]))
	for _, part := range parts[1:] {
		param := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(part), " ", ""))
		if strings.HasPrefix(param, "q=") {
			q, err := strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64)
			if err == nil && q <= 0 {
				return token, false
			}
		}
	}
	return token, token != ""
}
