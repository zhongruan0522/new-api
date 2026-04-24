package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/model"
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
