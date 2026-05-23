package model

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

// DynamicRatioRule 动态倍率规则
type DynamicRatioRule struct {
	Id          int64   `json:"id" gorm:"primaryKey"`
	Enable      bool    `json:"enable" gorm:"default:true"`
	Group       string  `json:"group" gorm:"not null;index"`
	Concurrency *int64  `json:"concurrency" gorm:""`
	Weekdays    string  `json:"weekdays" gorm:"default:''"`
	StartTime   string  `json:"start_time" gorm:"default:''"`
	EndTime     string  `json:"end_time" gorm:"default:''"`
	Ratio       float64 `json:"ratio" gorm:"not null"`
	AppliesToSubscription bool `json:"applies_to_subscription" gorm:"default:false"`
	Priority    int     `json:"priority" gorm:"default:0;index"`
	CreatedAt   int64   `json:"created_at" gorm:"not null"`
	UpdatedAt   int64   `json:"updated_at" gorm:"not null"`
}

func (DynamicRatioRule) TableName() string {
	return "dynamic_ratio_rules"
}

// Validate 校验规则字段
func (r *DynamicRatioRule) Validate() error {
	if r.Group == "" {
		return fmt.Errorf("分组不能为空")
	}
	if !ratio_setting.ContainsGroupRatio(r.Group) {
		return fmt.Errorf("分组 %s 不存在", r.Group)
	}
	if r.Ratio <= 0 {
		return fmt.Errorf("倍率必须大于 0")
	}
	if r.Concurrency != nil && *r.Concurrency <= 0 {
		return fmt.Errorf("并发阈值必须大于 0")
	}
	if (r.StartTime == "") != (r.EndTime == "") {
		return fmt.Errorf("开始时间和结束时间必须同时设置或同时留空")
	}
	if r.StartTime != "" {
		if _, err := time.Parse("15:04", r.StartTime); err != nil {
			return fmt.Errorf("开始时间格式错误，应为 HH:MM")
		}
	}
	if r.EndTime != "" {
		if _, err := time.Parse("15:04", r.EndTime); err != nil {
			return fmt.Errorf("结束时间格式错误，应为 HH:MM")
		}
	}
	if r.Weekdays != "" {
		var days []int
		if err := common.UnmarshalJsonStr(r.Weekdays, &days); err != nil {
			return fmt.Errorf("星期格式错误，应为 JSON 数组")
		}
		for _, d := range days {
			if d < 0 || d > 6 {
				return fmt.Errorf("星期值必须在 0-6 范围内")
			}
		}
	}
	return nil
}

// GetDynamicRatioRules 获取所有规则
func GetDynamicRatioRules() ([]*DynamicRatioRule, error) {
	var rules []*DynamicRatioRule
	err := DB.Order("priority ASC, id ASC").Find(&rules).Error
	return rules, err
}

// GetDynamicRatioRuleById 按 ID 获取规则
func GetDynamicRatioRuleById(id int64) (*DynamicRatioRule, error) {
	var rule DynamicRatioRule
	err := DB.Where("id = ?", id).First(&rule).Error
	return &rule, err
}

// CreateDynamicRatioRule 创建规则
func CreateDynamicRatioRule(rule *DynamicRatioRule) error {
	now := time.Now().Unix()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	return DB.Create(rule).Error
}

// UpdateDynamicRatioRule 更新规则
func UpdateDynamicRatioRule(rule *DynamicRatioRule) error {
	rule.UpdatedAt = time.Now().Unix()
	return DB.Save(rule).Error
}

// DeleteDynamicRatioRule 删除规则
func DeleteDynamicRatioRule(id int64) error {
	return DB.Where("id = ?", id).Delete(&DynamicRatioRule{}).Error
}

// ReorderDynamicRatioRules 重排优先级
func ReorderDynamicRatioRules(ids []int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&DynamicRatioRule{}).Where("id = ?", id).Update("priority", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DynamicRatioStatus 用户端动态倍率状态
type DynamicRatioStatus struct {
	Enabled     bool                  `json:"enabled"`
	ActiveRatio float64               `json:"active_ratio"`
	ActiveGroup string                `json:"active_group,omitempty"`
	Timezone    string                `json:"timezone"`
	RulesCount  int                   `json:"rules_count"`
	Rules       []DynamicRatioSummary `json:"rules"`
}

// DynamicRatioSummary 规则摘要
type DynamicRatioSummary struct {
	Group       string  `json:"group"`
	Concurrency *int64  `json:"concurrency"`
	Weekdays    string  `json:"weekdays"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Ratio       float64 `json:"ratio"`
	Priority    int     `json:"priority"`
}

// GetDynamicRatioStatus 获取指定分组的动态倍率状态
func GetDynamicRatioStatus(group string) DynamicRatioStatus {
	return GetDynamicRatioStatusForGroups([]string{group})
}

// GetDynamicRatioStatusForGroups 获取多个可用分组中的动态倍率状态
func GetDynamicRatioStatusForGroups(groups []string) DynamicRatioStatus {
	status := DynamicRatioStatus{
		Enabled:     common.DynamicRatioEnabled,
		ActiveRatio: 1.0,
		Timezone:    common.StartupTimezoneName(),
	}

	if !common.DynamicRatioEnabled {
		return status
	}

	groups = normalizeDynamicRatioGroups(groups)
	if len(groups) == 0 {
		return status
	}
	groupSet := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		groupSet[group] = struct{}{}
	}

	dynamicRatioCacheLock.RLock()
	rules := make([]parsedDynamicRatioRule, len(dynamicRatioRules))
	copy(rules, dynamicRatioRules)
	dynamicRatioCacheLock.RUnlock()

	var groupRules []parsedDynamicRatioRule
	for _, r := range rules {
		if _, ok := groupSet[r.Group]; ok {
			groupRules = append(groupRules, r)
		}
	}

	status.RulesCount = len(groupRules)
	status.Rules = make([]DynamicRatioSummary, 0, len(groupRules))
	for _, r := range groupRules {
		status.Rules = append(status.Rules, DynamicRatioSummary{
			Group:       r.Group,
			Concurrency: r.Concurrency,
			Weekdays:    r.Weekdays,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
			Ratio:       r.Ratio,
			Priority:    r.Priority,
		})
	}

	// 计算当前生效倍率
	concurrency := getActiveConnections()
	now := common.NowInStartupTimezone()
	hasActiveRatio := false
	for _, group := range groups {
		activeRatio := matchDynamicRatio(rules, group, concurrency, now)
		if activeRatio <= 0 {
			continue
		}
		if !hasActiveRatio || math.Abs(activeRatio-1) > math.Abs(status.ActiveRatio-1) {
			status.ActiveRatio = activeRatio
			status.ActiveGroup = group
			hasActiveRatio = true
		}
	}

	return status
}

func normalizeDynamicRatioGroups(groups []string) []string {
	groupSet := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		groupSet[group] = struct{}{}
	}

	result := make([]string, 0, len(groupSet))
	for group := range groupSet {
		result = append(result, group)
	}
	sort.Strings(result)
	return result
}

// GetMatchedDynamicRatio 从缓存中匹配动态倍率，返回倍率值（0 表示未命中）
func GetMatchedDynamicRatio(group string) float64 {
	return getMatchedDynamicRatio(group, false)
}

func GetMatchedSubscriptionDynamicRatio(group string) float64 {
	return getMatchedDynamicRatio(group, true)
}

func getMatchedDynamicRatio(group string, subscriptionOnly bool) float64 {
	if !common.DynamicRatioEnabled {
		return 0
	}

	dynamicRatioCacheLock.RLock()
	rules := dynamicRatioRules
	dynamicRatioCacheLock.RUnlock()
	if subscriptionOnly {
		filtered := make([]parsedDynamicRatioRule, 0, len(rules))
		for _, rule := range rules {
			if rule.AppliesToSubscription {
				filtered = append(filtered, rule)
			}
		}
		rules = filtered
	}

	return matchDynamicRatio(rules, group, getActiveConnections(), common.NowInStartupTimezone())
}

// matchDynamicRatio 核心匹配逻辑（纯函数，方便测试）
func matchDynamicRatio(rules []parsedDynamicRatioRule, group string, concurrency int64, now time.Time) float64 {
	type scoredRule struct {
		rule           parsedDynamicRatioRule
		hasConcurrency bool
		concurrencyGap int64
	}

	var matched []scoredRule
	currentMinutes := now.Hour()*60 + now.Minute()

	for _, r := range rules {
		if !r.Enable {
			continue
		}
		if r.Group != group {
			continue
		}

		effectiveWeekday := int(now.Weekday())

		// 检查时间段条件（使用预解析的分钟值）
		if r.HasTimeRange {
			startMinutes := r.ParsedStartMin
			endMinutes := r.ParsedEndMin

			if startMinutes <= endMinutes {
				// 不跨天
				if currentMinutes < startMinutes || currentMinutes >= endMinutes {
					continue
				}
			} else {
				// 跨天
				if currentMinutes < startMinutes && currentMinutes >= endMinutes {
					continue
				}
				if currentMinutes < endMinutes {
					effectiveWeekday = int(now.AddDate(0, 0, -1).Weekday())
				}
			}
		}

		// 检查并发条件
		if r.Concurrency != nil {
			if concurrency <= *r.Concurrency {
				// 当前并发 <= 阈值，不满足（需要严格大于）
				continue
			}
		}

		// 检查星期条件（使用预解析的数组）
		if r.ParsedWeekdays != nil {
			found := false
			for _, d := range r.ParsedWeekdays {
				if d == effectiveWeekday {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		sr := scoredRule{
			rule:           r,
			hasConcurrency: r.Concurrency != nil,
		}
		if r.Concurrency != nil {
			sr.concurrencyGap = concurrency - *r.Concurrency
		}
		matched = append(matched, sr)
	}

	if len(matched) == 0 {
		return 0
	}

	// 优先级裁决
	best := matched[0]
	for _, m := range matched[1:] {
		if m.hasConcurrency && !best.hasConcurrency {
			best = m
		} else if !m.hasConcurrency && best.hasConcurrency {
			continue
		} else if m.hasConcurrency && best.hasConcurrency {
			if m.concurrencyGap < best.concurrencyGap {
				best = m
			} else if m.concurrencyGap == best.concurrencyGap {
				if m.rule.Priority < best.rule.Priority {
					best = m
				} else if m.rule.Priority == best.rule.Priority && m.rule.Id < best.rule.Id {
					best = m
				}
			}
		} else {
			// 都没有并发条件
			if m.rule.Priority < best.rule.Priority {
				best = m
			} else if m.rule.Priority == best.rule.Priority && m.rule.Id < best.rule.Id {
				best = m
			}
		}
	}

	return best.rule.Ratio
}

// getActiveConnections 获取当前 relay 并发数
func getActiveConnections() int64 {
	if common.GetActiveConnectionsFunc != nil {
		return common.GetActiveConnectionsFunc()
	}
	return 0
}
