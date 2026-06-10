package model

import (
	"strings"
	"testing"
	"time"

	"github.com/zhongruan0522/new-api/common"
)

func int64Ptr(v int64) *int64 {
	return &v
}

// toParsedRules 测试辅助：将原始规则转为预解析后的缓存规则
func toParsedRules(rules []DynamicRatioRule) []parsedDynamicRatioRule {
	return parseDynamicRatioRules(rules)
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0, got %f", ratio)
	}
}

// 测试：分组不匹配
func TestMatchDynamicRatio_GroupNotMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "default", "gpt-4", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 (no match), got %f", ratio)
	}
}

// 测试：并发条件满足（当前 > 阈值）
func TestMatchDynamicRatio_ConcurrencyMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 1.5, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 15, time.Now())
	if ratio != 1.5 {
		t.Errorf("expected 1.5, got %f", ratio)
	}
}

// 测试：并发条件不满足（当前 = 阈值，严格大于）
func TestMatchDynamicRatio_ConcurrencyEqual(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 1.5, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 10, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 (concurrency = threshold, strict >), got %f", ratio)
	}
}

// 测试：并发条件不满足（当前 < 阈值）
func TestMatchDynamicRatio_ConcurrencyBelow(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Ratio: 1.5, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 5, time.Now())
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
	if ratio != 0 {
		t.Errorf("expected 0 (Sunday not in weekdays), got %f", ratio)
	}
}

func TestMatchDynamicRatio_EmptyWeekdaysMatchesEveryDay(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Weekdays: "[]", Ratio: 1.5, Priority: 0},
	}

	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, makeTime(10, 0, time.Sunday))
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
	if ratio != 1.5 {
		t.Errorf("expected 1.5, got %f", ratio)
	}
}

func TestMatchDynamicRatio_EqualTimeRangeMatchesAllDay(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "00:00", EndTime: "00:00", Ratio: 1.5, Priority: 0},
	}

	for _, now := range []time.Time{
		makeTime(0, 0, time.Monday),
		makeTime(12, 30, time.Wednesday),
		makeTime(23, 59, time.Sunday),
	} {
		ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
		if ratio != 1.5 {
			t.Errorf("expected 1.5 for all-day range at %s, got %f", now.Format("15:04"), ratio)
		}
	}
}

// 测试：时间段不匹配（不跨天）
func TestMatchDynamicRatio_TimeNotMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", StartTime: "09:00", EndTime: "18:00", Ratio: 1.5, Priority: 0},
	}
	now := makeTime(20, 0, time.Wednesday)
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
	if ratio != 0 {
		t.Errorf("expected 0 (12:00 not in 22:00-06:00), got %f", ratio)
	}
}

func TestMatchDynamicRatio_CrossDayUsesPreviousWeekdayAfterMidnight(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Weekdays: "[1]", StartTime: "22:00", EndTime: "06:00", Ratio: 2.0, Priority: 0},
	}

	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, makeTime(3, 0, time.Tuesday))
	if ratio != 2.0 {
		t.Errorf("expected 2.0 for Tuesday 03:00 matching Monday overnight rule, got %f", ratio)
	}
}

func TestMatchDynamicRatio_UsesNowLocation(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load Asia/Shanghai location: %v", err)
	}

	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Weekdays: "[1]", StartTime: "08:00", EndTime: "09:00", Ratio: 2.0, Priority: 0},
	}
	instant := time.Date(2026, 5, 18, 0, 30, 0, 0, time.UTC)

	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, instant.In(shanghai))
	if ratio != 2.0 {
		t.Errorf("expected 2.0 for Asia/Shanghai 08:30 Monday, got %f", ratio)
	}

	ratio = matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, instant)
	if ratio != 0 {
		t.Errorf("expected 0 for UTC 00:30 Monday outside 08:00-09:00, got %f", ratio)
	}
}

// 测试：复合条件全满足
func TestMatchDynamicRatio_CompositeAllMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(10), Weekdays: "[1,2,3,4,5]", StartTime: "09:00", EndTime: "18:00", Ratio: 2.5, Priority: 0},
	}
	now := makeTime(10, 0, time.Wednesday) // Wednesday=3, 10:00
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 15, now)
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 5, now) // concurrency too low
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 15, time.Now())
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 20, time.Now())
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 15, time.Now())
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 15, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (id 3 < 5), got %f", ratio)
	}
}

// 测试：Enable=false 不参与匹配
func TestMatchDynamicRatio_DisabledRule(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: false, Group: "vip", Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 (disabled rule), got %f", ratio)
	}
}

// 测试：无任何规则匹配
func TestMatchDynamicRatio_NoMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Concurrency: int64Ptr(100), Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "default", "gpt-4", 0, time.Now())
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
	ratio := GetMatchedDynamicRatio("vip", "gpt-4")
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
	ratio := GetMatchedDynamicRatio("vip", "gpt-4")
	if ratio != 2.0 {
		t.Errorf("expected 2.0 (global enabled), got %f", ratio)
	}
}

func TestGetDynamicRatioStatusForGroupsMatchesAnyUsableGroup(t *testing.T) {
	originalEnabled := common.DynamicRatioEnabled
	t.Cleanup(func() {
		common.DynamicRatioEnabled = originalEnabled
		SetDynamicRatioRulesForTest(nil)
	})

	common.DynamicRatioEnabled = true
	SetDynamicRatioRulesForTest([]DynamicRatioRule{
		{Id: 1, Enable: true, Group: "default", Ratio: 1.5, Priority: 0},
	})

	status := GetDynamicRatioStatusForGroups([]string{"root", "default"})
	if status.ActiveRatio != 1.5 {
		t.Errorf("expected active ratio 1.5 from usable default group, got %f", status.ActiveRatio)
	}
	if status.ActiveGroup != "default" {
		t.Errorf("expected active group default, got %s", status.ActiveGroup)
	}
	if status.RulesCount != 1 {
		t.Errorf("expected one visible rule, got %d", status.RulesCount)
	}
}

// 测试：无并发条件的规则之间按 priority 排序
func TestMatchDynamicRatio_NoConcurrencyPriority(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 1.5, Priority: 5},
		{Id: 2, Enable: true, Group: "vip", Ratio: 2.0, Priority: 2},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, time.Now())
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
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
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, now)
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

func TestDynamicRatioStatusDoesNotExposeRuleModels(t *testing.T) {
	originalEnabled := common.DynamicRatioEnabled
	common.DynamicRatioEnabled = true
	t.Cleanup(func() {
		common.DynamicRatioEnabled = originalEnabled
		SetDynamicRatioRulesForTest(nil)
	})

	SetDynamicRatioRulesForTest([]DynamicRatioRule{
		{Id: 1, Enable: true, Group: "default", Models: `["sensitive-model*"]`, Ratio: 2.0, Priority: 0},
	})

	status := GetDynamicRatioStatus("default")
	payload, err := common.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}

	if strings.Contains(string(payload), "sensitive-model") || strings.Contains(string(payload), `"models"`) {
		t.Fatalf("dynamic ratio status exposed model filters: %s", payload)
	}
}

// --- 模型匹配测试 ---

// 测试：规则无模型限制时匹配所有模型
func TestMatchDynamicRatio_NoModelsMatchesAll(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 2.0, Priority: 0},
	}
	for _, model := range []string{"gpt-4", "claude-3-opus", "gemini-pro", ""} {
		ratio := matchDynamicRatio(toParsedRules(rules), "vip", model, 0, time.Now())
		if ratio != 2.0 {
			t.Errorf("expected 2.0 for model %q (no model filter), got %f", model, ratio)
		}
	}
}

// 测试：精确模型匹配
func TestMatchDynamicRatio_ExactModelMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Models: `["gpt-4","claude-3-opus"]`, Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 for exact model match, got %f", ratio)
	}
}

// 测试：模型不匹配
func TestMatchDynamicRatio_ModelNotMatch(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Models: `["gpt-4","claude-3-opus"]`, Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gemini-pro", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 for model not in list, got %f", ratio)
	}
}

// 测试：前缀通配符匹配 gpt-4*
func TestMatchDynamicRatio_WildcardPrefix(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Models: `["gpt-4*"]`, Ratio: 2.0, Priority: 0},
	}
	for _, model := range []string{"gpt-4", "gpt-4o", "gpt-4-turbo", "gpt-4o-mini"} {
		ratio := matchDynamicRatio(toParsedRules(rules), "vip", model, 0, time.Now())
		if ratio != 2.0 {
			t.Errorf("expected 2.0 for model %q matching gpt-4*, got %f", model, ratio)
		}
	}
	// 不匹配的模型
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-3.5-turbo", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 for model gpt-3.5-turbo not matching gpt-4*, got %f", ratio)
	}
}

// 测试：后缀通配符匹配 *-preview
func TestMatchDynamicRatio_WildcardSuffix(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Models: `["*-preview"]`, Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4-preview", 0, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 for model matching *-preview, got %f", ratio)
	}
	ratio = matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 for model not matching *-preview, got %f", ratio)
	}
}

// 测试：全通配符 * 匹配所有模型
func TestMatchDynamicRatio_WildcardAll(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Models: `["*"]`, Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "anything", 0, time.Now())
	if ratio != 2.0 {
		t.Errorf("expected 2.0 for wildcard *, got %f", ratio)
	}
}

// 测试：有模型条件的规则优先于无模型条件的规则
func TestMatchDynamicRatio_ModelRulePriority(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Ratio: 1.5, Priority: 0},
		{Id: 2, Enable: true, Group: "vip", Models: `["gpt-4*"]`, Ratio: 2.5, Priority: 0},
	}
	// gpt-4o 应匹配有模型条件的规则
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "gpt-4o", 0, time.Now())
	if ratio != 2.5 {
		t.Errorf("expected 2.5 (model-specific rule wins), got %f", ratio)
	}
	// claude 应匹配无模型条件的规则
	ratio = matchDynamicRatio(toParsedRules(rules), "vip", "claude-3", 0, time.Now())
	if ratio != 1.5 {
		t.Errorf("expected 1.5 (fallback to no-model rule), got %f", ratio)
	}
}

// 测试：空模型名不匹配有模型限制的规则
func TestMatchDynamicRatio_EmptyModelWithModelsFilter(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Models: `["gpt-4"]`, Ratio: 2.0, Priority: 0},
	}
	ratio := matchDynamicRatio(toParsedRules(rules), "vip", "", 0, time.Now())
	if ratio != 0 {
		t.Errorf("expected 0 for empty model name with model filter, got %f", ratio)
	}
}

// 测试：混合多个模型模式
func TestMatchDynamicRatio_MixedModelPatterns(t *testing.T) {
	rules := []DynamicRatioRule{
		{Id: 1, Enable: true, Group: "vip", Models: `["gpt-4*","claude-3-opus","*-preview"]`, Ratio: 2.0, Priority: 0},
	}
	for _, tc := range []struct {
		model    string
		expected float64
	}{
		{"gpt-4", 2.0},
		{"gpt-4o-mini", 2.0},
		{"claude-3-opus", 2.0},
		{"my-model-preview", 2.0},
		{"claude-3-sonnet", 0},
		{"gemini-pro", 0},
	} {
		ratio := matchDynamicRatio(toParsedRules(rules), "vip", tc.model, 0, time.Now())
		if ratio != tc.expected {
			t.Errorf("model %q: expected %f, got %f", tc.model, tc.expected, ratio)
		}
	}
}

// 测试：Validate 拒绝无效的 Models JSON
func TestDynamicRatioRuleValidateInvalidModelsJson(t *testing.T) {
	rule := DynamicRatioRule{
		Enable: true,
		Group:  "default",
		Models: "not-json",
		Ratio:  1.5,
	}
	err := rule.Validate()
	if err == nil {
		t.Fatal("expected invalid models JSON to fail validation")
	}
}

// 测试：Validate 拒绝 Models 中有空字符串
func TestDynamicRatioRuleValidateEmptyModelName(t *testing.T) {
	rule := DynamicRatioRule{
		Enable: true,
		Group:  "default",
		Models: `["gpt-4", ""]`,
		Ratio:  1.5,
	}
	err := rule.Validate()
	if err == nil {
		t.Fatal("expected empty model name to fail validation")
	}
}

// 测试：matchModelPattern
func TestMatchModelPattern(t *testing.T) {
	tests := []struct {
		model    string
		pattern  string
		expected bool
	}{
		{"gpt-4", "gpt-4", true},
		{"gpt-4o", "gpt-4", false},
		{"gpt-4o", "gpt-4*", true},
		{"gpt-4-turbo", "gpt-4*", true},
		{"gpt-3.5-turbo", "gpt-4*", false},
		{"my-model-preview", "*-preview", true},
		{"my-model", "*-preview", false},
		{"anything", "*", true},
		{"", "gpt-4", false},
		{"", "*", false},
	}
	for _, tc := range tests {
		result := matchModelPattern(tc.model, tc.pattern)
		if result != tc.expected {
			t.Errorf("matchModelPattern(%q, %q) = %v, want %v", tc.model, tc.pattern, result, tc.expected)
		}
	}
}
