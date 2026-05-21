package model

import (
	"sort"
	"sync"
	"time"

	"github.com/zhongruan0522/new-api/common"
)

var (
	dynamicRatioRules    []DynamicRatioRule
	dynamicRatioCacheLock sync.RWMutex
)

// InitDynamicRatioCache 初始化动态倍率缓存
func InitDynamicRatioCache() {
	var rules []DynamicRatioRule
	err := DB.Where("enable = ?", true).Order("priority ASC, id ASC").Find(&rules).Error
	if err != nil {
		common.SysError("failed to load dynamic ratio rules: " + err.Error())
		return
	}

	dynamicRatioCacheLock.Lock()
	dynamicRatioRules = rules
	dynamicRatioCacheLock.Unlock()

	common.SysLog("dynamic ratio rules synced from database")
}

// SyncDynamicRatioCache 定时同步缓存
func SyncDynamicRatioCache(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing dynamic ratio rules from database")
		InitDynamicRatioCache()
	}
}

// RefreshDynamicRatioCache 主动刷新缓存（CRUD 后调用）
func RefreshDynamicRatioCache() {
	InitDynamicRatioCache()
}

// SetDynamicRatioRulesForTest 测试用：直接设置缓存规则
func SetDynamicRatioRulesForTest(rules []DynamicRatioRule) {
	// 排序：priority ASC, id ASC
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority < rules[j].Priority
		}
		return rules[i].Id < rules[j].Id
	})

	dynamicRatioCacheLock.Lock()
	dynamicRatioRules = rules
	dynamicRatioCacheLock.Unlock()
}

// GetDynamicRatioRulesFromCache 获取缓存中的规则（测试用）
func GetDynamicRatioRulesFromCache() []DynamicRatioRule {
	dynamicRatioCacheLock.RLock()
	defer dynamicRatioCacheLock.RUnlock()
	result := make([]DynamicRatioRule, len(dynamicRatioRules))
	copy(result, dynamicRatioRules)
	return result
}
