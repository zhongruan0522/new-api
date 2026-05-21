package model

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zhongruan0522/new-api/common"
)

// parsedDynamicRatioRule 预解析后的缓存规则，避免热路径重复 JSON 解析
type parsedDynamicRatioRule struct {
	DynamicRatioRule       // 嵌入原始规则，保留所有原始字段供前端展示
	ParsedWeekdays   []int // 预解析后的星期数组，nil 表示不限
	ParsedStartMin   int   // 预解析后的开始时间（分钟），-1 表示不限
	ParsedEndMin     int   // 预解析后的结束时间（分钟），-1 表示不限
	HasTimeRange     bool  // 是否有时间条件
}

var (
	dynamicRatioRules     []parsedDynamicRatioRule
	dynamicRatioCacheLock sync.RWMutex
)

// parseDynamicRatioRules 将 DB 规则转换为预解析后的缓存规则
func parseDynamicRatioRules(rules []DynamicRatioRule) []parsedDynamicRatioRule {
	result := make([]parsedDynamicRatioRule, 0, len(rules))
	for _, r := range rules {
		parsed := parsedDynamicRatioRule{
			DynamicRatioRule: r,
			ParsedStartMin:   -1,
			ParsedEndMin:     -1,
		}

		// 预解析 Weekdays
		if r.Weekdays != "" {
			var days []int
			if err := common.UnmarshalJsonStr(r.Weekdays, &days); err == nil && len(days) > 0 {
				parsed.ParsedWeekdays = days
			}
		}

		// 预解析 StartTime / EndTime
		if r.StartTime != "" && r.EndTime != "" {
			startParts := strings.Split(r.StartTime, ":")
			endParts := strings.Split(r.EndTime, ":")
			if len(startParts) == 2 && len(endParts) == 2 {
				sh, _ := strconv.Atoi(startParts[0])
				sm, _ := strconv.Atoi(startParts[1])
				eh, _ := strconv.Atoi(endParts[0])
				em, _ := strconv.Atoi(endParts[1])
				parsed.ParsedStartMin = sh*60 + sm
				parsed.ParsedEndMin = eh*60 + em
				parsed.HasTimeRange = parsed.ParsedStartMin != parsed.ParsedEndMin
			}
		}

		result = append(result, parsed)
	}
	return result
}

// InitDynamicRatioCache 初始化动态倍率缓存
func InitDynamicRatioCache() {
	var rules []DynamicRatioRule
	err := DB.Where("enable = ?", true).Order("priority ASC, id ASC").Find(&rules).Error
	if err != nil {
		common.SysError("failed to load dynamic ratio rules: " + err.Error())
		return
	}

	parsed := parseDynamicRatioRules(rules)

	dynamicRatioCacheLock.Lock()
	dynamicRatioRules = parsed
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

	parsed := parseDynamicRatioRules(rules)

	dynamicRatioCacheLock.Lock()
	dynamicRatioRules = parsed
	dynamicRatioCacheLock.Unlock()
}

// GetDynamicRatioRulesFromCache 获取缓存中的规则（测试用）
func GetDynamicRatioRulesFromCache() []parsedDynamicRatioRule {
	dynamicRatioCacheLock.RLock()
	defer dynamicRatioCacheLock.RUnlock()
	result := make([]parsedDynamicRatioRule, len(dynamicRatioRules))
	copy(result, dynamicRatioRules)
	return result
}
