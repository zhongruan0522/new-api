package controller

import (
	"testing"

	"github.com/zhongruan0522/new-api/setting/dashboard_setting"
)

func TestIsUserQuotaRangeTooLongAllowsOneMonthWindow(t *testing.T) {
	cfg := dashboard_setting.GetDashboardConfig()
	maxRange := int64(cfg.MaxTimeRangeDays) * 24 * 60 * 60
	start := int64(1_700_000_000)
	end := start + maxRange

	if isUserQuotaRangeTooLong(start, end) {
		t.Fatalf("expected %d-second range to be allowed", maxRange)
	}
}

func TestIsUserQuotaRangeTooLongRejectsOverOneMonthWindow(t *testing.T) {
	cfg := dashboard_setting.GetDashboardConfig()
	maxRange := int64(cfg.MaxTimeRangeDays) * 24 * 60 * 60
	start := int64(1_700_000_000)
	end := start + maxRange + 1

	if !isUserQuotaRangeTooLong(start, end) {
		t.Fatalf("expected range longer than %d seconds to be rejected", maxRange)
	}
}

func TestIsUserQuotaRangeTooLongIgnoresInvalidRange(t *testing.T) {
	if isUserQuotaRangeTooLong(0, 0) {
		t.Fatal("expected empty range to be ignored by span validation")
	}
	if isUserQuotaRangeTooLong(200, 100) {
		t.Fatal("expected reversed range to be ignored by span validation")
	}
}

// TestIsUserQuotaRangeTooLongRespectsMaxTimeRangeDaysConfig 验证收紧
// MaxTimeRangeDays 配置后，超出新上限的范围会被拒绝（修复 issue #111 的核心断言）。
func TestIsUserQuotaRangeTooLongRespectsMaxTimeRangeDaysConfig(t *testing.T) {
	cfg := dashboard_setting.GetDashboardConfig()
	original := cfg.MaxTimeRangeDays
	cfg.MaxTimeRangeDays = 7
	defer func() { cfg.MaxTimeRangeDays = original }()

	day := int64(24 * 60 * 60)
	start := int64(1_700_000_000)

	// 7 天刚好不超
	if isUserQuotaRangeTooLong(start, start+7*day) {
		t.Fatal("expected 7-day range to be allowed when MaxTimeRangeDays=7")
	}
	// 8 天超了
	if !isUserQuotaRangeTooLong(start, start+8*day) {
		t.Fatal("expected 8-day range to be rejected when MaxTimeRangeDays=7")
	}
}
