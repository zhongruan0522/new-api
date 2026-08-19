package service

import (
	"strings"
	"testing"
	"time"

	"github.com/NookMux/NookMux/pkg/jsonx"
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

// TestComputeGlmActivityTimeRange 验证时间窗始终是 365 天且 endTime 是当天 23:59:59。
// 把起点锁定为本地时区的整数天，避免在跨时区环境下被解析为 next day / previous day。
func TestComputeGlmActivityTimeRange(t *testing.T) {
	start, end := ComputeGlmActivityTimeRange()
	const layout = "2006-01-02 15:04:05"
	tStart, err := time.Parse(layout, start)
	if err != nil {
		t.Fatalf("parse startTime: %v", err)
	}
	tEnd, err := time.Parse(layout, end)
	if err != nil {
		t.Fatalf("parse endTime: %v", err)
	}
	if tStart.Hour() != 0 || tStart.Minute() != 0 || tStart.Second() != 0 {
		t.Fatalf("startTime = %q, want 00:00:00", start)
	}
	if tEnd.Hour() != 23 || tEnd.Minute() != 59 || tEnd.Second() != 59 {
		t.Fatalf("endTime = %q, want 23:59:59", end)
	}
	if got := int(tEnd.Sub(tStart).Hours() / 24); got != glmActivityLookbackDays {
		t.Fatalf("range days = %d, want %d", got, glmActivityLookbackDays)
	}
}

// TestIsGlmActivitySuccess 覆盖用户给出的成功判定：
// - code 为 0/200/null 且 success !== false → 成功
// - success 显式为 false → 失败
// - code 为其它业务码 → 失败
func TestIsGlmActivitySuccess(t *testing.T) {
	cases := []struct {
		name string
		resp *glmActivityResp
		want bool
	}{
		{"code=0", &glmActivityResp{Code: intPtr(0)}, true},
		{"code=200", &glmActivityResp{Code: intPtr(200)}, true},
		{"code=nil success=nil", &glmActivityResp{}, true},
		{"success=false overrides", &glmActivityResp{Code: intPtr(0), Success: boolPtr(false)}, false},
		{"success=true with code=0", &glmActivityResp{Code: intPtr(0), Success: boolPtr(true)}, true},
		{"code=500", &glmActivityResp{Code: intPtr(500)}, false},
		{"nil response", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isGlmActivitySuccess(tc.resp); got != tc.want {
				t.Fatalf("isGlmActivitySuccess = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCreateBigModelUsageHeaders_Authorization 校验 helper 输出的头至少包含
// Authorization，且不会因为前后空格污染智谱上游校验逻辑。
func TestCreateBigModelUsageHeaders_Authorization(t *testing.T) {
	h := createBigModelUsageHeaders("  my-key-123  ")
	if got := strings.TrimSpace(h.Get("Authorization")); got != "my-key-123" {
		t.Fatalf("Authorization = %q, want %q", h.Get("Authorization"), "my-key-123")
	}
	if h.Get("Referer") == "" || h.Get("Origin") == "" {
		t.Fatal("Referer/Origin must be present to mimic BigModel console origin")
	}
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

// 上游重置卡列表的 recordId 是数字（数据库自增 ID），曾按 string 解析导致
// "cannot unmarshal number into Go struct field ... recordId of type string"。
// 这里用上游真实响应结构做回归，覆盖解析与归一化两条路径。
func TestGlmResetCardListParse_NumericRecordId(t *testing.T) {
	raw := `{
		"code": 200,
		"msg": "操作成功",
		"data": {
			"customerId": 18211736137540903,
			"targetType": "PERSONAL",
			"organizationId": null,
			"projectId": null,
			"lastFiveHourResetTime": null,
			"lastWeekResetTime": null,
			"fiveHourResets": [],
			"weekResets": [
				{
					"recordId": 8938,
					"expireTime": "2026-08-26 23:59:59",
					"available": true
				}
			]
		},
		"success": true
	}`

	var resp glmResetCardListResp
	if err := jsonx.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal reset card list failed: %v", err)
	}
	if !resp.Success {
		t.Fatal("Success = false, want true")
	}
	if resp.Data == nil {
		t.Fatal("Data = nil, want parsed")
	}
	if len(resp.Data.FiveHourResets) != 0 {
		t.Fatalf("FiveHourResets length = %d, want 0", len(resp.Data.FiveHourResets))
	}
	if len(resp.Data.WeekResets) != 1 {
		t.Fatalf("WeekResets length = %d, want 1", len(resp.Data.WeekResets))
	}
	if got := resp.Data.WeekResets[0].RecordId; got != 8938 {
		t.Fatalf("RecordId = %d, want 8938", got)
	}

	week := normalizeGlmResetCards(resp.Data.WeekResets)
	if len(week) != 1 {
		t.Fatalf("normalized week cards length = %d, want 1", len(week))
	}
	if week[0].RecordId != 8938 {
		t.Fatalf("normalized RecordId = %d, want 8938", week[0].RecordId)
	}
	if !week[0].Available || !week[0].Priority {
		t.Fatal("first available card must keep available=true and be marked priority")
	}
}
