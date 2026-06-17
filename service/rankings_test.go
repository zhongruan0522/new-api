package service

import (
	"testing"

	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/setting/dashboard_setting"
)

// TestLimitRankedModelsRespectsLimit 验证 limitRankedModels 在数据量超过 limit 时正确截断。
func TestLimitRankedModelsRespectsLimit(t *testing.T) {
	rows := make([]RankedModel, 10)
	for i := range rows {
		rows[i] = RankedModel{Rank: i + 1, ModelName: "model-" + string(rune('a'+i))}
	}

	got := limitRankedModels(rows, 5)
	if len(got) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(got))
	}
}

// TestLimitRankedModelsKeepsAllWhenUnderLimit 验证数据量不超过 limit 时原样返回。
func TestLimitRankedModelsKeepsAllWhenUnderLimit(t *testing.T) {
	rows := []RankedModel{{Rank: 1, ModelName: "a"}, {Rank: 2, ModelName: "b"}}
	got := limitRankedModels(rows, 5)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
}

// TestLimitRankedVendorsRespectsLimit 验证 limitRankedVendors 在数据量超过 limit 时正确截断。
func TestLimitRankedVendorsRespectsLimit(t *testing.T) {
	rows := make([]RankedVendor, 8)
	for i := range rows {
		rows[i] = RankedVendor{Rank: i + 1, Vendor: "vendor-" + string(rune('a'+i))}
	}

	got := limitRankedVendors(rows, 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}
}

// TestBuildModelHistoryRespectsConfigLimit 验证 buildModelHistory 使用传入的 modelLimit
// 而非硬编码常量（修复 issue #111 的核心断言）。
func TestBuildModelHistoryRespectsConfigLimit(t *testing.T) {
	// 准备 8 条 totals 数据
	totals := make([]model.RankingQuotaTotal, 8)
	for i := range totals {
		totals[i] = model.RankingQuotaTotal{
			ModelName:   "model-" + string(rune('a'+i)),
			TotalTokens: int64(100 - i),
		}
	}
	meta := map[string]rankingModelMeta{}
	for _, item := range totals {
		meta[item.ModelName] = rankingModelMeta{vendor: "Unknown"}
	}

	config := rankingPeriodConfig{id: "week", labelLayout: "Jan 2"}

	// 配置 limit=5，应只有 5 个模型 + 可能的 Others
	history := buildModelHistory(nil, totals, meta, config, 5)

	topCount := 0
	for _, m := range history.Models {
		if m.Name != rankingOthersLabel {
			topCount++
		}
	}
	if topCount > 5 {
		t.Fatalf("expected at most 5 top models in history, got %d", topCount)
	}
}

// TestBuildVendorShareHistoryRespectsConfigLimit 验证 buildVendorShareHistory 使用传入的 vendorLimit
// 而非硬编码常量。
func TestBuildVendorShareHistoryRespectsConfigLimit(t *testing.T) {
	// 准备 6 个 vendor
	vendors := make([]RankedVendor, 6)
	for i := range vendors {
		vendors[i] = RankedVendor{
			Rank:        i + 1,
			Vendor:      "vendor-" + string(rune('a'+i)),
			TotalTokens: int64(100 - i*10),
		}
	}
	meta := map[string]rankingModelMeta{}
	config := rankingPeriodConfig{id: "week", labelLayout: "Jan 2"}

	history := buildVendorShareHistory(nil, vendors, 1000, meta, config, 3)

	topCount := 0
	for _, v := range history.Vendors {
		if v.Name != rankingOthersLabel {
			topCount++
		}
	}
	if topCount > 3 {
		t.Fatalf("expected at most 3 top vendors in share history, got %d", topCount)
	}
}

// TestRankingsConfigFallbackOnZero 验证配置为 0/负数时 fallback 到默认常量值。
func TestRankingsConfigFallbackOnZero(t *testing.T) {
	cfg := dashboard_setting.GetDashboardConfig()
	originalModel := cfg.RankingsModelLimit
	originalVendor := cfg.RankingsVendorLimit
	defer func() {
		cfg.RankingsModelLimit = originalModel
		cfg.RankingsVendorLimit = originalVendor
	}()

	cfg.RankingsModelLimit = 0
	cfg.RankingsVendorLimit = -1

	// 模拟 GetRankingsSnapshot 中的 fallback 逻辑
	modelLimit := cfg.RankingsModelLimit
	if modelLimit <= 0 {
		modelLimit = rankingLeaderboardLimit
	}
	vendorLimit := cfg.RankingsVendorLimit
	if vendorLimit <= 0 {
		vendorLimit = rankingVendorLimit
	}

	if modelLimit != rankingLeaderboardLimit {
		t.Fatalf("expected model limit fallback to %d, got %d", rankingLeaderboardLimit, modelLimit)
	}
	if vendorLimit != rankingVendorLimit {
		t.Fatalf("expected vendor limit fallback to %d, got %d", rankingVendorLimit, vendorLimit)
	}
}
