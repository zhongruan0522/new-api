package service

import (
	"testing"
)

// buildGlmPlanQuotaData 的套餐代次判定必须与智谱官方前端一致：
//   - limits 含 type=CREDIT_LIMIT 条目 → V3 积分套餐；
//   - 无 CREDIT_LIMIT 但含每周额度（TOKENS_LIMIT, unit=6）→ V2；
//   - 仅每 5 小时额度（TOKENS_LIMIT, unit=3）→ V1。

func TestBuildGlmPlanQuotaData_V1OnlyFiveHourTokens(t *testing.T) {
	lim := &glmLimitResp{}
	lim.Data.Limits = []struct {
		Type          string       `json:"type"`
		Unit          int          `json:"unit"`
		Percentage    int          `json:"percentage"`
		CurrentValue  int          `json:"currentValue"`
		Usage         int          `json:"usage"`
		NextResetTime glmResetTime `json:"nextResetTime"`
		UsageDetails  []struct {
			ModelCode string `json:"modelCode"`
			Usage     int    `json:"usage"`
		} `json:"usageDetails"`
	}{
		{Type: glmLimitTypeTokens, Unit: 3, Percentage: 20, CurrentValue: 2000, Usage: 10000},
		{Type: glmLimitTypeTime, Unit: 5, Percentage: 10, CurrentValue: 3, Usage: 100},
	}

	data := buildGlmPlanQuotaData(nil, lim)

	if data.PlanVersion != glmPlanVersion1 {
		t.Fatalf("PlanVersion = %q, want %q", data.PlanVersion, glmPlanVersion1)
	}
	if data.IsCreditPlan {
		t.Fatal("IsCreditPlan = true, want false for V1")
	}
	if data.TokenLimit == nil {
		t.Fatal("TokenLimit = nil, want present for V1")
	}
	if data.WeeklyLimit != nil {
		t.Fatal("WeeklyLimit should be nil for V1 (no unit=6 entry)")
	}
	if data.McpToolLimit == nil {
		t.Fatal("McpToolLimit = nil, want present for V1")
	}
	if data.CreditLimit != nil || data.CreditWeeklyLimit != nil {
		t.Fatal("credit limits should be nil for V1")
	}
}

func TestBuildGlmPlanQuotaData_V2WeeklyTokens(t *testing.T) {
	lim := &glmLimitResp{}
	lim.Data.Limits = []struct {
		Type          string       `json:"type"`
		Unit          int          `json:"unit"`
		Percentage    int          `json:"percentage"`
		CurrentValue  int          `json:"currentValue"`
		Usage         int          `json:"usage"`
		NextResetTime glmResetTime `json:"nextResetTime"`
		UsageDetails  []struct {
			ModelCode string `json:"modelCode"`
			Usage     int    `json:"usage"`
		} `json:"usageDetails"`
	}{
		{Type: glmLimitTypeTokens, Unit: 3, Percentage: 30},
		{Type: glmLimitTypeTokens, Unit: 6, Percentage: 45},
		{Type: glmLimitTypeTime, Unit: 5, Percentage: 5},
	}

	data := buildGlmPlanQuotaData(nil, lim)

	if data.PlanVersion != glmPlanVersion2 {
		t.Fatalf("PlanVersion = %q, want %q", data.PlanVersion, glmPlanVersion2)
	}
	if data.IsCreditPlan {
		t.Fatal("IsCreditPlan = true, want false for V2")
	}
	if data.WeeklyLimit == nil {
		t.Fatal("WeeklyLimit = nil, want present for V2")
	}
	if data.CreditLimit != nil || data.CreditWeeklyLimit != nil {
		t.Fatal("credit limits should be nil for V2")
	}
}

func TestBuildGlmPlanQuotaData_V3CreditPlan(t *testing.T) {
	lim := &glmLimitResp{}
	lim.Data.Limits = []struct {
		Type          string       `json:"type"`
		Unit          int          `json:"unit"`
		Percentage    int          `json:"percentage"`
		CurrentValue  int          `json:"currentValue"`
		Usage         int          `json:"usage"`
		NextResetTime glmResetTime `json:"nextResetTime"`
		UsageDetails  []struct {
			ModelCode string `json:"modelCode"`
			Usage     int    `json:"usage"`
		} `json:"usageDetails"`
	}{
		{Type: glmLimitTypeCredit, Unit: 3, Percentage: 25, CurrentValue: 2500, Usage: 10000},
		{Type: glmLimitTypeCredit, Unit: 6, Percentage: 60, CurrentValue: 6000, Usage: 10000},
	}

	data := buildGlmPlanQuotaData(nil, lim)

	if data.PlanVersion != glmPlanVersion3 {
		t.Fatalf("PlanVersion = %q, want %q", data.PlanVersion, glmPlanVersion3)
	}
	if !data.IsCreditPlan {
		t.Fatal("IsCreditPlan = false, want true for V3")
	}
	if data.CreditLimit == nil {
		t.Fatal("CreditLimit = nil, want 5-hour credit limit for V3")
	}
	if data.CreditLimit.CurrentValue != 2500 || data.CreditLimit.Usage != 10000 {
		t.Fatalf("CreditLimit values = %d/%d, want 2500/10000", data.CreditLimit.CurrentValue, data.CreditLimit.Usage)
	}
	if data.CreditWeeklyLimit == nil {
		t.Fatal("CreditWeeklyLimit = nil, want weekly credit limit for V3")
	}
	if data.CreditWeeklyLimit.Percentage != 60 {
		t.Fatalf("CreditWeeklyLimit.Percentage = %d, want 60", data.CreditWeeklyLimit.Percentage)
	}
	// V3 积分套餐官网只展示两条积分额度卡片，不再返回 Tokens/MCP 条目
	if data.TokenLimit != nil || data.WeeklyLimit != nil || data.McpToolLimit != nil {
		t.Fatal("tokens/MCP limits should be nil for pure V3 credit response")
	}
}

func TestBuildGlmPlanQuotaData_EmptyLimits(t *testing.T) {
	data := buildGlmPlanQuotaData(nil, &glmLimitResp{})
	if data.PlanVersion != glmPlanVersion1 {
		t.Fatalf("PlanVersion = %q, want fallback %q", data.PlanVersion, glmPlanVersion1)
	}
	if data.IsCreditPlan {
		t.Fatal("IsCreditPlan should be false when no limits")
	}
}
