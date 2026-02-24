package relay

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RelayStoredVideo serves videos persisted for "multimodal auto convert to URL".
// This is intentionally a lightweight unauthenticated endpoint using signed URLs (sig, optional exp).
func RelayStoredVideo(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id is required",
		})
		return
	}

	exp, now, ok := verifyStoredAssetSignature(c, "stored_video", id)
	if !ok {
		return
	}

	v, err := model.GetStoredVideoByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "stored_video_not_found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "read_stored_video_failed",
		})
		return
	}

	contentType := strings.TrimSpace(v.MimeType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	maxAge := int64(24 * 60 * 60)
	if exp > 0 {
		maxAge = exp - now
		if maxAge < 0 {
			maxAge = 0
		}
	}
	if maxAge > 24*60*60 {
		maxAge = 24 * 60 * 60
	}
	c.Writer.Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d", maxAge))
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

	c.Data(http.StatusOK, contentType, []byte(v.Data))
}
