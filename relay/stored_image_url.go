package relay

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// defaultStoredAssetSignedURLTTL:
//   - 0   => no expiry by default (URL is valid until the asset is cleaned from DB)
//   - >0  => add exp+sig for short-lived URLs
const defaultStoredAssetSignedURLTTL time.Duration = 0

func buildStoredImageURL(c *gin.Context, imageID string) string {
	return buildStoredAssetURLWithTTL(c, "image", "stored_image", imageID, defaultStoredAssetSignedURLTTL)
}

func buildStoredVideoURL(c *gin.Context, videoID string) string {
	return buildStoredAssetURLWithTTL(c, "video", "stored_video", videoID, defaultStoredAssetSignedURLTTL)
}

func buildStoredAssetURLWithTTL(c *gin.Context, routeType string, scope string, id string, ttl time.Duration) string {
	id = strings.TrimSpace(id)
	if id == "" || routeType == "" || scope == "" {
		return ""
	}

	var path string
	if ttl == 0 {
		sig := common.GenerateHMAC(fmt.Sprintf("%s:%s", scope, id))
		path = fmt.Sprintf("/mcp/%s/%s?sig=%s", routeType, url.PathEscape(id), sig)
	} else {
		exp := time.Now().Add(ttl).Unix()
		sig := common.GenerateHMAC(fmt.Sprintf("%s:%s:%d", scope, id, exp))
		path = fmt.Sprintf("/mcp/%s/%s?exp=%d&sig=%s", routeType, url.PathEscape(id), exp, sig)
	}

	base := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	if base == "" {
		base = guessBaseURLFromRequest(c)
	}
	if base == "" {
		return path
	}
	return base + path
}

func guessBaseURLFromRequest(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	proto := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = c.Request.Host
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	return proto + "://" + host
}
