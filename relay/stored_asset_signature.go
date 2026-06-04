package relay

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"

	"github.com/gin-gonic/gin"
)

// verifyStoredAssetSignature validates required `sig` and `exp` query params.
//
// Stored media URLs must be time-bound so leaked links do not remain valid until
// the asset is deleted.
//
// Returns:
//   - exp: the exp unix timestamp.
//   - now: current unix timestamp, useful for cache headers.
//   - ok:  whether validation passed (handler already responded when false).
func verifyStoredAssetSignature(c *gin.Context, scope string, id string) (exp int64, now int64, ok bool) {
	if c == nil {
		return 0, 0, false
	}

	scope = strings.TrimSpace(scope)
	id = strings.TrimSpace(id)
	if scope == "" || id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid asset",
		})
		return 0, 0, false
	}

	sig := strings.TrimSpace(c.Query("sig"))
	if sig == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "signed url required",
		})
		return 0, 0, false
	}

	expStr := strings.TrimSpace(c.Query("exp"))
	now = time.Now().Unix()

	if expStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "exp is required",
		})
		return 0, now, false
	}

	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || exp <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid exp",
		})
		return 0, now, false
	}
	if exp < now {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "url expired",
		})
		return 0, now, false
	}

	expected := common.GenerateHMAC(fmt.Sprintf("%s:%s:%d", scope, id, exp))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(sig)) != 1 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "invalid signature",
		})
		return 0, now, false
	}

	return exp, now, true
}
