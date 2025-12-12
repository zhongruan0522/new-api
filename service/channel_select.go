package service

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
func CacheGetRandomSatisfiedChannel(c *gin.Context, tokenGroup string, modelName string, retry int) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := tokenGroup
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	if tokenGroup == "auto" {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}
		autoGroups := GetUserAutoGroup(userGroup)
		startIndex := 0
		priorityRetry := retry
		crossGroupRetry := common.GetContextKeyBool(c, constant.ContextKeyTokenCrossGroupRetry)
		if crossGroupRetry && retry > 0 {
			logger.LogDebug(c, "Auto group retry cross group, retry: %d", retry)
			if lastIndex, exists := common.GetContextKey(c, constant.ContextKeyAutoGroupIndex); exists {
				if idx, ok := lastIndex.(int); ok {
					startIndex = idx + 1
					priorityRetry = 0
				}
			}
			logger.LogDebug(c, "Auto group retry cross group, start index: %d", startIndex)
		}

		for i := startIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			logger.LogDebug(c, "Auto selecting group: %s", autoGroup)
			channel, _ = model.GetRandomSatisfiedChannel(autoGroup, modelName, priorityRetry)
			if channel == nil {
				priorityRetry = 0
				continue
			} else {
				c.Set("auto_group", autoGroup)
				common.SetContextKey(c, constant.ContextKeyAutoGroupIndex, i)
				selectGroup = autoGroup
				logger.LogDebug(c, "Auto selected group: %s", autoGroup)
				break
			}
		}
	} else {
		channel, err = model.GetRandomSatisfiedChannel(tokenGroup, modelName, retry)
		if err != nil {
			return nil, tokenGroup, err
		}
	}
	return channel, selectGroup, nil
}
