package channelstore

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"testing"
)

// seedSearchChannelsTestChannels 插入 25 条渠道：
//   - id 1~15（含 1~10 为 type OpenAI 且 enabled，11~15 为 type Anthropic 且禁用）
//   - id 16~25 为 type OpenAI 且 enabled
//
// name 统一带 "search-ch-" 前缀，配合 keyword 过滤隔离其他测试数据。
func seedSearchChannelsTestChannels(t *testing.T) {
	t.Helper()
	for i := 1; i <= 25; i++ {
		channelType := constant.ChannelTypeOpenAI
		status := common.ChannelStatusEnabled
		if i > 10 && i <= 15 {
			channelType = constant.ChannelTypeAnthropic
			status = common.ChannelStatusAutoDisabled
		}
		createChannelCacheTestChannel(t, Channel{
			Name:   fmt.Sprintf("search-ch-%02d", i),
			Type:   channelType,
			Status: status,
			Models: "gpt-4",
			Group:  "Coding",
		})
	}
}

// TestSearchChannelsPagination 验证 SQL 层分页：25 条数据按 20/页分页，
// 第 1 页 20 条、第 2 页 5 条，排序与 offset/offset 组合正确。
func TestSearchChannelsPagination(t *testing.T) {
	setupChannelCacheTestDB(t)
	seedSearchChannelsTestChannels(t)

	page1, err := SearchChannels("search-ch-", "", "", false, 0, "", "", 0, 20)
	if err != nil {
		t.Fatalf("SearchChannels page1 error: %v", err)
	}
	if len(page1) != 20 {
		t.Fatalf("page1 should contain 20 channels, got %d", len(page1))
	}
	if page1[0].Name != "search-ch-01" || page1[19].Name != "search-ch-20" {
		t.Fatalf("page1 boundary wrong: first=%s last=%s", page1[0].Name, page1[19].Name)
	}

	page2, err := SearchChannels("search-ch-", "", "", false, 0, "", "", 20, 20)
	if err != nil {
		t.Fatalf("SearchChannels page2 error: %v", err)
	}
	if len(page2) != 5 {
		t.Fatalf("page2 should contain 5 channels, got %d", len(page2))
	}
	if page2[0].Name != "search-ch-21" || page2[4].Name != "search-ch-25" {
		t.Fatalf("page2 boundary wrong: first=%s last=%s", page2[0].Name, page2[4].Name)
	}
}

// TestSearchChannelsLimitDefaults 验证 limit 规范化：<=0 用默认上限 100，
// 超过 500 截断为 500（防滥用保护）。
func TestSearchChannelsLimitDefaults(t *testing.T) {
	setupChannelCacheTestDB(t)
	seedSearchChannelsTestChannels(t)

	// limit<=0 → 默认 100，25 条全部返回
	all, err := SearchChannels("search-ch-", "", "", false, 0, "", "", 0, 0)
	if err != nil {
		t.Fatalf("SearchChannels default limit error: %v", err)
	}
	if len(all) != 25 {
		t.Fatalf("default limit should return all 25 channels, got %d", len(all))
	}

	if got := normalizeSearchChannelsLimit(-1); got != 100 {
		t.Fatalf("normalizeSearchChannelsLimit(-1) = %d, want 100", got)
	}
	if got := normalizeSearchChannelsLimit(0); got != 100 {
		t.Fatalf("normalizeSearchChannelsLimit(0) = %d, want 100", got)
	}
	if got := normalizeSearchChannelsLimit(501); got != 500 {
		t.Fatalf("normalizeSearchChannelsLimit(501) = %d, want 500", got)
	}
	if got := normalizeSearchChannelsLimit(500); got != 500 {
		t.Fatalf("normalizeSearchChannelsLimit(500) = %d, want 500", got)
	}
}

// TestSearchChannelsWithMetaCounts 验证跨页聚合语义：
//   - 无过滤时 total=25，typeCounts 只含两种 type（OpenAI=20, Anthropic=5）；
//   - status=enabled 过滤时 total=20，typeCounts 只统计 enabled（Anthropic 禁用条目消失）；
//   - status=0（非 enabled）过滤时 total=5，typeCounts 只含 Anthropic=5；
//   - type 过滤影响 items 与 total，但不影响 typeCounts（保持原实现
//     "status 过滤后、type 过滤前统计" 的语义）。
func TestSearchChannelsWithMetaCounts(t *testing.T) {
	setupChannelCacheTestDB(t)
	seedSearchChannelsTestChannels(t)

	// 无过滤
	_, total, typeCounts, err := SearchChannelsWithMeta("search-ch-", "", "", false, 0, "", "", -1, -1, 0, 20)
	if err != nil {
		t.Fatalf("SearchChannelsWithMeta error: %v", err)
	}
	if total != 25 {
		t.Fatalf("total = %d, want 25", total)
	}
	if len(typeCounts) != 2 || typeCounts[int64(constant.ChannelTypeOpenAI)] != 20 || typeCounts[int64(constant.ChannelTypeAnthropic)] != 5 {
		t.Fatalf("typeCounts = %v, want OpenAI=20 Anthropic=5", typeCounts)
	}

	// status=enabled：只统计启用渠道
	_, total, typeCounts, err = SearchChannelsWithMeta("search-ch-", "", "", false, 0, "", "", common.ChannelStatusEnabled, -1, 0, 20)
	if err != nil {
		t.Fatalf("SearchChannelsWithMeta enabled error: %v", err)
	}
	if total != 20 {
		t.Fatalf("enabled total = %d, want 20", total)
	}
	if len(typeCounts) != 1 || typeCounts[int64(constant.ChannelTypeOpenAI)] != 20 {
		t.Fatalf("enabled typeCounts = %v, want only OpenAI=20", typeCounts)
	}

	// status=0（非 enabled）：只统计禁用渠道
	_, total, typeCounts, err = SearchChannelsWithMeta("search-ch-", "", "", false, 0, "", "", 0, -1, 0, 20)
	if err != nil {
		t.Fatalf("SearchChannelsWithMeta disabled error: %v", err)
	}
	if total != 5 {
		t.Fatalf("disabled total = %d, want 5", total)
	}
	if len(typeCounts) != 1 || typeCounts[int64(constant.ChannelTypeAnthropic)] != 5 {
		t.Fatalf("disabled typeCounts = %v, want only Anthropic=5", typeCounts)
	}

	// type 过滤：total/items 收敛到该 type，typeCounts 仍按全部 where（含 status）统计
	items, total, typeCounts, err := SearchChannelsWithMeta("search-ch-", "", "", false, 0, "", "", common.ChannelStatusEnabled, int(constant.ChannelTypeOpenAI), 0, 20)
	if err != nil {
		t.Fatalf("SearchChannelsWithMeta type filter error: %v", err)
	}
	if total != 20 {
		t.Fatalf("type-filtered total = %d, want 20", total)
	}
	if len(items) != 20 {
		t.Fatalf("type-filtered items = %d, want 20", len(items))
	}
	for _, item := range items {
		if item.Type != constant.ChannelTypeOpenAI {
			t.Fatalf("type-filtered items should all be OpenAI, got type %d (%s)", item.Type, item.Name)
		}
	}
	if len(typeCounts) != 1 || typeCounts[int64(constant.ChannelTypeOpenAI)] != 20 {
		t.Fatalf("typeCounts should ignore type filter, got %v", typeCounts)
	}
}
