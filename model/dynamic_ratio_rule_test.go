package model

import (
	"testing"
	"time"

	"github.com/zhongruan0522/new-api/common"
)

func int64Ptr(v int64) *int64 {
	return &v
}

// helper: 构造指定星期几的时间
// 2026-05-18 is Monday
func makeTime(hour, minute int, weekday time.Weekday) time.Time {
	// Monday = 0 offset, so weekday offset from Monday
	// time.Weekday: Sunday=0, Monday=1, ..., Saturday=6
	// We want date such that the weekday matches
	// 2026-05-18 is Monday (time.Monday = 1)
	offset := int(weekday) - int(time.Monday)
	if offset < 0 {
		offset += 7
	}
	return time.Date(2026, 5, 18+offset, hour, minute, 0, 0, time.Local)
}

// 测试：仅分组匹配，无并发/时间条件
func TestMatchDynamicRatio_BasicGroupOnly(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(rules, "vip", 0, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0, got %f", ratio)
	}
}

// 测试：分组不匹配
func TestMatchDynamicRatio_GroupNotMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(rules, "default", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 (no match), got %f", ratio)
	}
}

// 测试：并发条件满足（当前 > 阈值）
func TestMatchDynamicRatio_ConcurrencyMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 1.5, Priority: 0},
	}
	ratio := matchDynamicRatio(rules, "vip", 15, time.Now())
	if ratio != 1.5 {
		t.Errorf("expected 1.5, got %f", ratio)
	}
}

// 测试：并发条件不满足（当前 = 阈值，严格大于）
func TestMatchDynamicRatio_ConcurrencyEqual(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 1.5, Priority: 0},
	}
	ratio := matchDynamicRatio(rules, "vip", 10, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 (concurrency = threshold, strict >), got %f", ratio)
	}
}

// 测试：并发条件不满足（当前 < 阈值）
func TestMatchDynamicRatio_ConcurrencyBelow(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 1.5, Priority: 0},
	}
	ratio := matchDynamicRatio(rules, "vip", 5, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 (concurrency < threshold), got %f", ratio)
	}
}

// 测试：星期匹配
func TestMatchDynamicRatio_WeekdayMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Weekdays: "[1,2,3,4,5]", Ratio: 1.5, Priority: 0},
	}
	// Wednesday = 3
	now := makeTime(10, 0, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 1.5 {
		t.Errorf("expected 1.5, got %f", ratio)
	}
}

// 测试：星期不匹配
func TestMatchDynamicRatio_WeekdayNotMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Weekdays: "[1,2,3,4,5]", Ratio: 1.5, Priority: 0},
	}
	// Sunday = 0
	now := makeTime(10, 0, time.Sunday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 0 {
		t.Errorf("expected 0 (Sunday not in weekdays), got %f", ratio)
	}
}

func TestMatchDynamicRatio_EmptyWeekdaysMatchesEveryDay(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Weekdays: "[]", Ratio: 1.5, Priority: 0},
	}

	ratio := matchDynamicRatio(rules, "vip", 0, makeTime(10, 0, time.Sunday))
	if ratio != 1.5 {
		t.Errorf("expected 1.5 for empty weekdays, got %f", ratio)
	}
}

// 测试：时间段匹配（不跨天）
func TestMatchDynamicRatio_TimeMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "09:00", EndTime: "18:00", Ratio: 1.5, Priority: 0},
	}
	now := makeTime(12, 30, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 1.5 {
		t.Errorf("expected 1.5, got %f", ratio)
	}
}

// 测试：时间段不匹配（不跨天）
func TestMatchDynamicRatio_TimeNotMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "09:00", EndTime: "18:00", Ratio: 1.5, Priority: 0},
	}
	now := makeTime(20, 0, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 0 {
		t.Errorf("expected 0 (outside time range), got %f", ratio)
	}
}

// 测试：跨天时间段 22:00-06:00，当前 23:00
func TestMatchDynamicRatio_CrossDayMatch1(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "22:00", EndTime: "06:00", Ratio: 2.0, Priority: 0},
	}
	now := makeTime(23, 0, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (23:00 in 22:00-06:00), got %f", ratio)
	}
}

// 测试：跨天时间段 22:00-06:00，当前 03:00
func TestMatchDynamicRatio_CrossDayMatch2(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "22:00", EndTime: "06:00", Ratio: 2.0, Priority: 0},
	}
	now := makeTime(3, 0, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (03:00 in 22:00-06:00), got %f", ratio)
	}
}

// 测试：跨天时间段 22:00-06:00，当前 12:00
func TestMatchDynamicRatio_CrossDayNotMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "22:00", EndTime: "06:00", Ratio: 2.0, Priority: 0},
	}
	now := makeTime(12, 0, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 0 {
		t.Errorf("expected 0 (12:00 not in 22:00-06:00), got %f", ratio)
	}
}

func TestMatchDynamicRatio_CrossDayUsesPreviousWeekdayAfterMidnight(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Weekdays: "[1]", StartTime: "22:00", EndTime: "06:00", Ratio: 2.0, Priority: 0},
	}

	ratio := matchDynamicRatio(rules, "vip", 0, makeTime(3, 0, time.Tuesday))
	if ratio != 2.0 {
		t.Errorf("expected 2.0 for Tuesday 03:00 matching Monday overnight rule, got %f", ratio)
	}
}

// 测试：复合条件全满足
func TestMatchDynamicRatio_CompositeAllMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Weekdays: "[1,2,3,4,5]", StartTime: "09:00", EndTime: "18:00", Ratio: 2.5, Priority: 0},
	}
	now := makeTime(10, 0, time.Wednesday) // Wednesday=3, 10:00
	ratio := matchDynamicRatio(rules, "vip", 15, now)
	if ratio != 2.5 {
		t.Errorf("expected 2.5, got %f", ratio)
	}
}

// 测试：复合条件部分不满足（并发不够）
func TestMatchDynamicRatio_CompositePartialFail(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Weekdays: "[1,2,3,4,5]", StartTime: "09:00", EndTime: "18:00", Ratio: 2.5, Priority: 0},
	}
	now := makeTime(10, 0, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 5, now) // concurrency too low
	if ratio != 0 {
		t.Errorf("expected 0 (concurrency not met), got %f", ratio)
	}
}

// 测试：多规则命中，有并发条件优先于无并发条件
func TestMatchDynamicRatio_ConcurrencyPriority(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 1.5, Priority: 0},
		{Id: 2, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 2.0, Priority: 10},
	}
	ratio := matchDynamicRatio(rules, "vip", 15, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (concurrency rule wins), got %f", ratio)
	}
}

// 测试：多规则命中，并发差值最小者胜
func TestMatchDynamicRatio_ConcurrencyGapMin(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(5), Ratio: 1.5, Priority: 0},  // gap = 15
		{Id: 2, Enable: true, Group: "vip", Concurrency: int64Ptr(15), Ratio: 2.0, Priority: 0}, // gap = 5
		{Id: 3, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 3.0, Priority: 0}, // gap = 10
	}
	ratio := matchDynamicRatio(rules, "vip", 20, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (gap=5 is smallest), got %f", ratio)
	}
}

// 测试：并发差值相同时，按 priority 升序
func TestMatchDynamicRatio_PriorityTieBreak(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 1.5, Priority: 5},
		{Id: 2, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 2.0, Priority: 2},
	}
	ratio := matchDynamicRatio(rules, "vip", 15, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (priority 2 < 5), got %f", ratio)
	}
}

// 测试：并发差值和 priority 都相同时，按 id 升序
func TestMatchDynamicRatio_IdTieBreak(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 5, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 1.5, Priority: 0},
		{Id: 3, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(rules, "vip", 15, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (id 3 < 5), got %f", ratio)
	}
}

// 测试：Enable=false 不参与匹配
func TestMatchDynamicRatio_DisabledRule(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: false, Group: "vip", Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(rules, "vip", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 (disabled rule), got %f", ratio)
	}
}

// 测试：无任何规则匹配
func TestMatchDynamicRatio_NoMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(100), Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(rules, "default", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 (no matching rules), got %f", ratio)
	}
}

// 测试：全局开关关闭时直接返回 0
func TestGetMatchedDynamicRatio_Disabled(t *testing.T) {
	common.DynamicRatioEnabled = false
	SetDynamicRatioRulesForTest([]DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 2.0, Priority: 0},
	})
	ratio := GetMatchedDynamicRatio("vip")
	if ratio != 0 {
		t.Errorf("expected 0 (global disabled), got %f", ratio)
	}
}

// 测试：全局开关开启时正常匹配
func TestGetMatchedDynamicRatio_Enabled(t *testing.T) {
	common.DynamicRatioEnabled = true
	SetDynamicRatioRulesForTest([]DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 2.0, Priority: 0},
	})
	ratio := GetMatchedDynamicRatio("vip")
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (global enabled), got %f", ratio)
	}
}

// 测试：无并发条件的规则之间按 priority 排序
func TestMatchDynamicRatio_NoConcurrencyPriority(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 1.5, Priority: 5},
		{Id: 2, Enable: true, Group: "vip", Ratio: 2.0, Priority: 2},
	}
	ratio := matchDynamicRatio(rules, "vip", 0, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (priority 2 < 5), got %f", ratio)
	}
}

// 测试：边界时间 - 正好在开始时间
func TestMatchDynamicRatio_TimeBoundaryStart(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "09:00", EndTime: "18:00", Ratio: 1.5, Priority: 0},
	}
	now := makeTime(9, 0, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 1.5 {
		t.Errorf("expected 1.5 (at start time), got %f", ratio)
	}
}

// 测试：边界时间 - 正好在结束时间（不包含）
func TestMatchDynamicRatio_TimeBoundaryEnd(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "09:00", EndTime: "18:00", Ratio: 1.5, Priority: 0},
	}
	now := makeTime(18, 0, time.Wednesday)
	ratio := matchDynamicRatio(rules, "vip", 0, now)
	if ratio != 0 {
		t.Errorf("expected 0 (at end time, exclusive), got %f", ratio)
	}
}

func TestDynamicRatioRuleValidateRequiresTimeRangePair(t *testing.T) {
	rule := DynamicRatioRule{
		Enable:    true,
		Group:     "default",
		StartTime: "09:00",
		Ratio:     1.5,
	}

	err := rule.Validate()
	if err == nil {
		t.Fatal("expected missing end time to fail validation")
	}
}
