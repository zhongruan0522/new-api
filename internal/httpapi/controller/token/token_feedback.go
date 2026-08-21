package tokencontroller

import (
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/store/token"
	"github.com/gin-gonic/gin"
)

func GetTokenForFeedback(c *gin.Context) (*tokenstore.Token, error) {
	token, err := tokenstore.GetTokenByKey(httpapi.GetContextKeyString(c, common.ContextKeyTokenKey), false)
	if err != nil {
		return nil, err
	}

	quotaType := httpapi.GetContextKeyInt(c, common.ContextKeyTokenQuotaType)
	if quotaType == 0 && !token.UnlimitedQuota {
		quotaType = token.QuotaType
		if quotaType == 0 {
			quotaType = 1
		}
	}
	if quotaType == 2 || quotaType == 3 {
		return tokenstore.GetTokenById(httpapi.GetContextKeyInt(c, common.ContextKeyTokenId))
	}
	return token, nil
}
