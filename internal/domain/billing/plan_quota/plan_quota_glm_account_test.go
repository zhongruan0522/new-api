package planquota

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// parseGlmAccountReport 必须兼容上游账户报告的多种响应形态：
//   - 金额字段可能为 null（todaySpendAmount、creditBalance 等）；
//   - 科学计数法（frozenBalance=0E-9）必须正常解析；
//   - 未建模字段随时可能在 null 与数字/对象间变化，不允许导致解析失败。

func TestParseGlmAccountReport_SamplePayload(t *testing.T) {
	body := []byte(`{
		"code": 200,
		"msg": "操作成功",
		"data": {
			"balance": 0.015345150,
			"rechargeAmount": 50.830000,
			"giveAmount": 862.050000,
			"totalSpendAmount": 912.864654850,
			"todaySpendAmount": null,
			"availableBalance": 0.015345150,
			"frozenBalance": 0E-9,
			"creditBalance": null,
			"availableCreditBalance": null,
			"creditStatus": "NOT_OPEN",
			"modelSpendAmountList": null,
			"isKA": false
		},
		"success": true
	}`)

	report, err := parseGlmAccountReport(body)
	if err != nil {
		t.Fatalf("parseGlmAccountReport returned error: %v", err)
	}
	if report.Balance == nil || *report.Balance != 0.015345150 {
		t.Fatalf("Balance = %v, want 0.015345150", report.Balance)
	}
	if report.RechargeAmount == nil || *report.RechargeAmount != 50.83 {
		t.Fatalf("RechargeAmount = %v, want 50.83", report.RechargeAmount)
	}
	if report.GiveAmount == nil || *report.GiveAmount != 862.05 {
		t.Fatalf("GiveAmount = %v, want 862.05", report.GiveAmount)
	}
	if report.TotalSpendAmount == nil || *report.TotalSpendAmount != 912.86465485 {
		t.Fatalf("TotalSpendAmount = %v, want 912.86465485", report.TotalSpendAmount)
	}
	if report.AvailableBalance == nil || *report.AvailableBalance != 0.015345150 {
		t.Fatalf("AvailableBalance = %v, want 0.015345150", report.AvailableBalance)
	}
	if report.FrozenBalance == nil || *report.FrozenBalance != 0 {
		t.Fatalf("FrozenBalance = %v, want 0 (from 0E-9)", report.FrozenBalance)
	}
}

func TestParseGlmAccountReport_ToleratesExtraFieldsWithValues(t *testing.T) {
	// 上游字段可能从 null 变成有值：creditBalance 变数字、
	// modelSpendAmountList 变数组、新增未知顶层字段，都不允许报错。
	body := []byte(`{
		"code": 200,
		"msg": "操作成功",
		"data": {
			"balance": 12.5,
			"availableBalance": 10.5,
			"creditBalance": 88.25,
			"availableCreditBalance": 80,
			"creditStatus": "OPEN",
			"modelSpendAmountList": [{"modelName": "glm-4.6", "spendAmount": 1.5}],
			"isKA": true,
			"brandNewField": {"nested": [1, 2, 3]}
		},
		"success": true,
		"extraTopLevel": "ignored"
	}`)

	report, err := parseGlmAccountReport(body)
	if err != nil {
		t.Fatalf("parseGlmAccountReport returned error: %v", err)
	}
	if report.Balance == nil || *report.Balance != 12.5 {
		t.Fatalf("Balance = %v, want 12.5", report.Balance)
	}
}

func TestParseGlmAccountReport_MissingOptionalFields(t *testing.T) {
	// 上游精简响应体时，未给出的金额字段应保持 nil 而不是报错。
	body := []byte(`{"code": 200, "msg": "操作成功", "data": {"balance": 3.14}, "success": true}`)

	report, err := parseGlmAccountReport(body)
	if err != nil {
		t.Fatalf("parseGlmAccountReport returned error: %v", err)
	}
	if report.Balance == nil || *report.Balance != 3.14 {
		t.Fatalf("Balance = %v, want 3.14", report.Balance)
	}
	if report.RechargeAmount != nil || report.GiveAmount != nil ||
		report.TotalSpendAmount != nil || report.AvailableBalance != nil ||
		report.FrozenBalance != nil {
		t.Fatal("unreported fields should stay nil")
	}
}

func TestParseGlmAccountReport_Failure(t *testing.T) {
	body := []byte(`{"code": 500, "msg": "系统异常", "success": false}`)

	report, err := parseGlmAccountReport(body)
	if err == nil {
		t.Fatal("parseGlmAccountReport should fail on explicit failure response")
	}
	if !strings.Contains(err.Error(), "系统异常") {
		t.Fatalf("error should carry upstream msg, got: %v", err)
	}
	if report != nil {
		t.Fatal("report should be nil on failure")
	}
}

func TestParseGlmAccountReport_MissingData(t *testing.T) {
	body := []byte(`{"code": 200, "msg": "操作成功", "success": true}`)

	if _, err := parseGlmAccountReport(body); err == nil {
		t.Fatal("parseGlmAccountReport should fail when data is missing")
	}
}

func TestPickGlmPersistBalance(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	// 优先活跃可用余额
	report := &GlmAccountReportData{Balance: f(1), AvailableBalance: f(2)}
	if v, err := PickGlmPersistBalance(report); err != nil || v != 2 {
		t.Fatalf("PickGlmPersistBalance = %v, %v; want 2, nil", v, err)
	}

	// availableBalance 缺失时回退当前余额
	report = &GlmAccountReportData{Balance: f(1)}
	if v, err := PickGlmPersistBalance(report); err != nil || v != 1 {
		t.Fatalf("PickGlmPersistBalance = %v, %v; want 1, nil", v, err)
	}

	// 两者均缺失属于上游数据异常，必须报错暴露
	report = &GlmAccountReportData{RechargeAmount: f(5)}
	if _, err := PickGlmPersistBalance(report); err == nil {
		t.Fatal("PickGlmPersistBalance should fail when no balance field present")
	}

	if _, err := PickGlmPersistBalance(nil); err == nil {
		t.Fatal("PickGlmPersistBalance should fail on nil report")
	}
}

// TestFetchGlmAccountReportRequestHeaders 验证账户报告请求按官方调用形式携带
// Bearer Key、浏览器 UA 与 Accept，且认证失败（code=1000）升级为 ErrGlmKeyInvalid。
func TestFetchGlmAccountReportRequestHeaders(t *testing.T) {
	var gotAuth, gotUA, gotAccept string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		if r.URL.Path != glmAccountReportPath {
			t.Errorf("request path = %q, want %q", r.URL.Path, glmAccountReportPath)
		}
		_, _ = io.WriteString(w, `{"code":200,"msg":"操作成功","data":{"balance":1.5},"success":true}`)
	}))
	defer srv.Close()

	report, err := fetchGlmAPI(srv.URL, glmAccountReportPath, "Bearer test-key", "", glmAccountReportHeaders())
	if err != nil {
		t.Fatalf("fetchGlmAPI returned error: %v", err)
	}
	parsed, err := parseGlmAccountReport(report)
	if err != nil {
		t.Fatalf("parseGlmAccountReport returned error: %v", err)
	}
	if parsed.Balance == nil || *parsed.Balance != 1.5 {
		t.Fatalf("Balance = %v, want 1.5", parsed.Balance)
	}

	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-key")
	}
	if gotUA != glmBrowserUserAgent {
		t.Errorf("User-Agent = %q, want browser UA %q", gotUA, glmBrowserUserAgent)
	}
	if gotAccept != "*/*" {
		t.Errorf("Accept = %q, want */*", gotAccept)
	}
}

// TestFetchGlmAccountReport_AuthFailureUpgraded 验证 code=1000 的 200 响应
// 被 fetchGlmAPI 升级为 ErrGlmKeyInvalid，而不是当作空数据返回。
func TestFetchGlmAccountReport_AuthFailureUpgraded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":1000,"msg":"身份验证失败。","success":false}`)
	}))
	defer srv.Close()

	_, err := fetchGlmAPI(srv.URL, glmAccountReportPath, "Bearer bad-key", "", glmAccountReportHeaders())
	if err == nil || !strings.Contains(err.Error(), ErrGlmKeyInvalid.Error()) {
		t.Fatalf("fetchGlmAPI should upgrade auth failure to ErrGlmKeyInvalid, got: %v", err)
	}
}
