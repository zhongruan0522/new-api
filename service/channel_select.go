package service

import (
	"errors"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/logger"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/setting"
	"github.com/zhongruan0522/new-api/types"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx             *gin.Context
	TokenGroup      string
	ModelName       string
	Retry           *int
	RelayFormat     types.RelayFormat
	ExcludeChannelId int   // 同优先级内重试时排除上次失败的渠道
	resetNextTry    bool
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
// 尝试获取一个满足要求的随机渠道。
//
// Retry mapping: retry/2 maps to priority level, retry%2==1 means same-priority retry
// (excluding the previously failed channel).
//
// 重试映射：retry/2 映射到优先级级别，retry%2==1 表示同优先级重试（排除上次失败的渠道）。
//
// Example (3 priorities, RetryTimes=5):
// 示例（3个优先级，RetryTimes=5）：
//
//	retry=0: priority0, channel A
//	retry=1: priority0, channel B (exclude A)
//	retry=2: priority1, channel C
//	retry=3: priority1, channel D (exclude C)
//	retry=4: priority2, channel E
//	retry=5: priority2, channel F (exclude E)
//
// For "auto" tokenGroup with cross-group Retry enabled:
// 对于启用了跨分组重试的 "auto" tokenGroup：
//
//   - Each group will exhaust all its priorities before moving to the next group.
//     每个分组会用完所有优先级后才会切换到下一个分组。
//
//   - Uses ContextKeyAutoGroupIndex to track current group index.
//     使用 ContextKeyAutoGroupIndex 跟踪当前分组索引。
//
//   - Uses ContextKeyAutoGroupRetryIndex to track the global Retry count when current group started.
//     使用 ContextKeyAutoGroupRetryIndex 跟踪当前分组开始时的全局重试次数。
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := param.TokenGroup
	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)
	preferredAPIType := types.RelayFormatToPreferredAPIType(param.RelayFormat)

	if param.TokenGroup == "auto" {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}
		autoGroups := GetUserAutoGroup(userGroup)

		// startGroupIndex: the group index to start searching from
		// startGroupIndex: 开始搜索的分组索引
		startGroupIndex := 0
		crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)

		if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				startGroupIndex = idx
			}
		}

		for i := startGroupIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			// Calculate priorityIndex for current group using retry/2 mapping
			// 使用 retry/2 映射计算当前分组的优先级索引
			priorityRetry := param.GetRetry()
			// If moved to a new group, reset priorityRetry and update startRetryIndex
			// 如果切换到新分组，重置 priorityRetry 并更新 startRetryIndex
			if i > startGroupIndex {
				priorityRetry = 0
			}
			priorityIndex := priorityRetry / 2
			logger.LogDebug(param.Ctx, "Auto selecting group: %s, priorityIndex: %d", autoGroup, priorityIndex)

			channel, _ = model.GetRandomSatisfiedChannel(autoGroup, param.ModelName, priorityIndex, preferredAPIType, param.ExcludeChannelId)
			if channel == nil {
				// Current group has no available channel for this model, try next group
				// 当前分组没有该模型的可用渠道，尝试下一个分组
				logger.LogDebug(param.Ctx, "No available channel in group %s for model %s at priorityIndex %d, trying next group", autoGroup, param.ModelName, priorityIndex)
				// 重置状态以尝试下一个分组
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupRetryIndex, 0)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				continue
			}
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, autoGroup)
			selectGroup = autoGroup
			logger.LogDebug(param.Ctx, "Auto selected group: %s", autoGroup)

			// Calculate how many retry slots this group needs: 2 * numPriorities
			// 计算该分组需要多少重试槽位：2 * 优先级数量
			maxRetriesForGroup := 0
			if crossGroupRetry {
				// Get the number of priorities for this group/model
				// 获取该分组/模型的优先级数量
				numPriorities := model.GetPriorityCount(autoGroup, param.ModelName)
				maxRetriesForGroup = numPriorities*2 - 1 // -1 because retry starts from 0
			}

			// Prepare state for next retry
			// 为下一次重试准备状态
			if crossGroupRetry && priorityRetry >= maxRetriesForGroup {
				// Current group has exhausted all retries, prepare to switch to next group
				// This request still uses current group, but next retry will use next group
				// 当前分组已用完所有重试次数，准备切换到下一个分组
				// 本次请求仍使用当前分组，但下次重试将使用下一个分组
				logger.LogDebug(param.Ctx, "Current group %s retries exhausted (priorityRetry=%d >= maxRetries=%d), preparing switch to next group for next retry", autoGroup, priorityRetry, maxRetriesForGroup)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				param.ResetRetryNextTry()
			} else {
				// Stay in current group, save current state
				// 保持在当前分组，保存当前状态
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i)
			}
			break
		}
	} else {
		priorityIndex := param.GetRetry() / 2
		channel, err = model.GetRandomSatisfiedChannel(param.TokenGroup, param.ModelName, priorityIndex, preferredAPIType, param.ExcludeChannelId)
		if err != nil {
			return nil, param.TokenGroup, err
		}
	}
	return channel, selectGroup, nil
}
