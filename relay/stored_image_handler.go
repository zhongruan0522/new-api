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

// RelayStoredImage serves images persisted for "multimodal auto convert to URL".
//
// This is intentionally a lightweight unauthenticated endpoint using signed URLs (sig, optional exp).
func RelayStoredImage(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id is required",
		})
		return
	}

	exp, now, ok := verifyStoredAssetSignature(c, "stored_image", id)
	if !ok {
		return
	}

	img, err := model.GetStoredImageByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "stored_image_not_found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "read_stored_image_failed",
		})
		return
	}

	contentType := strings.TrimSpace(img.MimeType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Basic caching: keep private; if exp is present align to it, otherwise use a fixed max-age.
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

	c.Data(http.StatusOK, contentType, []byte(img.Data))
}
