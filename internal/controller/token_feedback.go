package controller

import (
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/constant"
	"github.com/NookMux/NookMux/internal/model"
	"github.com/gin-gonic/gin"
)

func getTokenForFeedback(c *gin.Context) (*model.Token, error) {
	token, err := model.GetTokenByKey(common.GetContextKeyString(c, constant.ContextKeyTokenKey), false)
	if err != nil {
		return nil, err
	}

	quotaType := common.GetContextKeyInt(c, constant.ContextKeyTokenQuotaType)
	if quotaType == 0 && !token.UnlimitedQuota {
		quotaType = token.QuotaType
		if quotaType == 0 {
			quotaType = 1
		}
	}
	if quotaType == 2 || quotaType == 3 {
		return model.GetTokenById(common.GetContextKeyInt(c, constant.ContextKeyTokenId))
	}
	return token, nil
}
