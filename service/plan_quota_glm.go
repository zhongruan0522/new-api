package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/NookMux/NookMux/common"
)

// GLM 套餐查询的 API 端点
const (
	glmSubscriptionPath        = "/api/biz/subscription/list?pageSize=10&pageNum=1"
	glmQuotaLimitPath          = "/api/monitor/usage/quota/limit"
	glmCreditUsageActivityPath = "/api/monitor/usage/credit-usage/activity"
	glmActivityAccountPersonal = "1" // 个人套餐

	// GlmActivityAccountPersonal 是导出别名，供 controller 按业务语义引用，
	// 其值仍以 glmActivityAccountPersonal 为单一来源，避免在调用处写字面量。
	GlmActivityAccountPersonal = glmActivityAccountPersonal
)

// glmActivityLookbackDays 智谱 GLM 个人套餐活跃数据展示窗口：当前日期往前推 365 天。
// 智谱活动接口在更长时间窗下响应体可能显著增大，保持 365 天与智谱官方前端一致。
const glmActivityLookbackDays = 365

// 智谱套餐额度类型。TOKENS_LIMIT/TIME_LIMIT 为按 Tokens 计费的 V1/V2 套餐条目，
// CREDIT_LIMIT 为 V3 积分套餐特有的按积分计费条目（参考智谱官网前端 CREDIT_LIMIT 常量）。
const (
	glmLimitTypeTokens = "TOKENS_LIMIT"
	glmLimitTypeTime   = "TIME_LIMIT"
	glmLimitTypeCredit = "CREDIT_LIMIT"
)

// GLM 套餐版本。智谱官方前端按 V1/V2/V3 区分套餐代次：
//   - V1：仅每 5 小时额度（unit=3），无每周额度；
//   - V2：每 5 小时 + 每周额度（unit=6），按 Tokens 计费；
//   - V3：额度条目 type=CREDIT_LIMIT，按积分计费。
const (
	glmPlanVersion1 = "V1"
	glmPlanVersion2 = "V2"
	glmPlanVersion3 = "V3"
)

// glmAuthFailureCode 是智谱 biz/monitor 接口在 API Key 无效或认证失败时返回的固定错误码。
// 智谱在 Key 失效时会返回 {"code":1000,"msg":"Authentication Failed"/"身份验证失败。","success":false}，
// 且 HTTP 状态码仍为 200，必须从响应体识别，否则会被误判为风控或空数据。
const glmAuthFailureCode = 1000

// ErrGlmKeyInvalid 表示智谱 API Key 无效或认证失败。
// Controller 通过 errors.Is 识别该错误，向前端返回明确的「Key 无效」提示，
// 而不是把空数据当作风控或空白用量展示。
var ErrGlmKeyInvalid = errors.New("glm api key invalid or authentication failed")

// glmBizResp 智谱 biz/monitor 接口通用响应外壳，用于识别认证失败等业务层错误。
type glmBizResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
}

// isGlmAuthFailure 判断响应体是否为智谱认证失败（Key 无效）。
// 智谱的 msg 会随 Accept-Language 在 "Authentication Failed" 与 "身份验证失败。" 间切换，
// 因此以稳定的 code 码作为判定依据，而不是匹配 msg 文案。
func isGlmAuthFailure(body []byte) bool {
	var resp glmBizResp
	if err := common.Unmarshal(body, &resp); err != nil {
		return false
	}
	return resp.Code == glmAuthFailureCode
}

// GlmPlanQuotaData 聚合了智谱 GLM 套餐的所有可展示信息
type GlmPlanQuotaData struct {
	PlanName      string           `json:"plan_name"`
	PlanVersion   string           `json:"plan_version,omitempty"` // 套餐代次: V1 / V2 / V3
	IsCreditPlan  bool             `json:"is_credit_plan"`         // 是否为按积分计费的 V3 套餐
	ProductLevel  string           `json:"product_level"`
	ProductName   string           `json:"product_name"`
	EffectiveDate string           `json:"effective_date"`
	ExpiryDate    string           `json:"expiry_date"`
	AutoRenew     bool             `json:"auto_renew"`
	WeeklyLimit   *GlmLimitInfo    `json:"weekly_limit,omitempty"`
	TokenLimit    *GlmLimitInfo    `json:"token_limit,omitempty"`
	McpToolLimit  *GlmMcpLimitInfo `json:"mcp_tool_limit,omitempty"`
	// CreditLimit 为 V3 积分套餐的 5 小时积分额度（unit=3, type=CREDIT_LIMIT）。
	CreditLimit *GlmCreditLimitInfo `json:"credit_limit,omitempty"`
	// CreditWeeklyLimit 为 V3 积分套餐的周积分额度（unit=6, type=CREDIT_LIMIT）。
	CreditWeeklyLimit *GlmCreditLimitInfo `json:"credit_weekly_limit,omitempty"`
}

// GlmLimitInfo 通用限额信息
type GlmLimitInfo struct {
	Percentage    int    `json:"percentage"`
	NextResetTime string `json:"next_reset_time,omitempty"`
	Status        string `json:"status"`
}

// GlmCreditLimitInfo 积分限额信息。
// 智谱积分套餐会同时返回已用积分（currentValue）与总额度积分（usage），
// 官网以 "currentValue / usage 积分" 形式展示。
type GlmCreditLimitInfo struct {
	Percentage    int    `json:"percentage"`
	CurrentValue  int    `json:"current_value"`
	Usage         int    `json:"usage"`
	NextResetTime string `json:"next_reset_time,omitempty"`
	Status        string `json:"status"`
}

// GlmMcpLimitInfo MCP 工具限额信息
type GlmMcpLimitInfo struct {
	Percentage    int             `json:"percentage"`
	CurrentUsage  string          `json:"current_usage,omitempty"`
	NextResetTime string          `json:"next_reset_time,omitempty"`
	Status        string          `json:"status"`
	Tools         []GlmToolDetail `json:"tools,omitempty"`
}

// GlmToolDetail MCP 工具详情
type GlmToolDetail struct {
	Name  string `json:"name"`
	Usage int    `json:"usage"`
}

// glmResetTime 兼容智谱 nextResetTime 既可能返回字符串，也可能返回 Unix 时间戳。
// 统一规范成 RFC3339 字符串后，再交给前端按 zaicontrol 的展示逻辑格式化。
type glmResetTime string

// UnmarshalJSON 兼容字符串和数字两种时间表示，避免严格 string 反序列化失败。
func (t *glmResetTime) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*t = ""
		return nil
	}

	var text string
	if err := common.Unmarshal(data, &text); err == nil {
		normalized, err := normalizeGlmResetTime(text)
		if err != nil {
			return err
		}
		*t = glmResetTime(normalized)
		return nil
	}

	var number json.Number
	if err := common.Unmarshal(data, &number); err == nil {
		normalized, err := normalizeGlmResetTime(number.String())
		if err != nil {
			return err
		}
		*t = glmResetTime(normalized)
		return nil
	}

	return fmt.Errorf("nextResetTime 字段格式不支持: %s", raw)
}

// normalizeGlmResetTime 将时间字符串统一整理成前端稳定可识别的格式。
func normalizeGlmResetTime(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if !isNumericTimestamp(trimmed) {
		return trimmed, nil
	}

	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return "", fmt.Errorf("解析 nextResetTime 时间戳失败: %w", err)
	}

	return normalizeUnixTimestamp(parsed).Format(time.RFC3339), nil
}

// isNumericTimestamp 判断字符串是否是纯数字时间戳，便于兼容被包装成字符串的时间戳。
func isNumericTimestamp(value string) bool {
	hasDigit := false
	for i, r := range value {
		if r == '-' && i == 0 {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
		hasDigit = true
	}
	return hasDigit
}

// normalizeUnixTimestamp 按位数推断秒/毫秒/微秒/纳秒时间戳，保持与 JS Date 的时间点语义一致。
func normalizeUnixTimestamp(value int64) time.Time {
	absValue := value
	if absValue < 0 {
		absValue = -absValue
	}

	switch {
	case absValue >= 1_000_000_000_000_000_000:
		return time.Unix(0, value).UTC()
	case absValue >= 1_000_000_000_000_000:
		return time.Unix(0, value*int64(time.Microsecond)).UTC()
	case absValue >= 1_000_000_000_000:
		return time.UnixMilli(value).UTC()
	default:
		return time.Unix(value, 0).UTC()
	}
}

// glmSubscriptionResp 智谱订阅接口返回格式
type glmSubscriptionResp struct {
	Data []struct {
		ProductName      string `json:"productName"`
		CurrentRenewTime string `json:"currentRenewTime"`
		NextRenewTime    string `json:"nextRenewTime"`
		AutoRenew        int    `json:"autoRenew"`
	} `json:"data"`
}

// glmLimitResp 智谱限额接口返回格式
type glmLimitResp struct {
	Data struct {
		Limits []struct {
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
		} `json:"limits"`
	} `json:"data"`
}

// GlmRiskCheckResult 风控检测结果
type GlmRiskCheckResult struct {
	IsRisk bool   `json:"is_risk"`
	RawMsg string `json:"raw_msg,omitempty"`
}

// CheckGlmRiskStatus 检测智谱账号是否被风控；proxyURL 非空时请求经渠道代理发出。
// 调用 https://open.bigmodel.cn/api/biz/labelCustomer/isRiskCustomer
func CheckGlmRiskStatus(apiKey string, proxyURL string) (*GlmRiskCheckResult, error) {
	url := "https://open.bigmodel.cn/api/biz/labelCustomer/isRiskCustomer"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Key 放在 Authorization 头中做身份验证
	req.Header.Set("Authorization", strings.TrimSpace(apiKey))

	client, err := newGlmHttpClient(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("代理客户端创建失败: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, string(body))
	}

	// 智谱在 Key 无效时仍返回 HTTP 200，但响应体为认证失败（code=1000）。
	// 此时 data 缺失，必须优先识别为 Key 无效，避免被下面的逻辑误判为「已风控」。
	if isGlmAuthFailure(body) {
		return nil, ErrGlmKeyInvalid
	}

	// 解析响应，期望格式: {"code":200,"msg":"操作成功","data":false,"success":true}
	var resp struct {
		Code    int         `json:"code"`
		Msg     string      `json:"msg"`
		Data    interface{} `json:"data"`
		Success bool        `json:"success"`
	}
	if err := common.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// data 不是 false 就视为风控
	isRisk := true
	if boolVal, ok := resp.Data.(bool); ok && !boolVal {
		isRisk = false
	}

	return &GlmRiskCheckResult{
		IsRisk: isRisk,
		RawMsg: resp.Msg,
	}, nil
}

// newGlmHttpClient 返回经渠道代理的 GLM 后端客户端（15s 超时）。
// 超时会覆盖共享客户端，需拷贝实例。
func newGlmHttpClient(proxyURL string) (*http.Client, error) {
	baseClient, err := GetHttpClientWithProxy(proxyURL)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport:     baseClient.Transport,
		CheckRedirect: baseClient.CheckRedirect,
		Timeout:       15 * time.Second,
	}, nil
}

// FetchGlmPlanQuota 从智谱后端拉取套餐额度数据
// apiKey: 渠道的 API Key
// planBaseURL: 套餐的基础 URL (glm-coding-plan 或 glm-coding-plan-international)
func FetchGlmPlanQuota(apiKey string, planBaseURL string, proxyURL string) (*GlmPlanQuotaData, error) {
	apiBase := getGlmApiBase(planBaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("无法确定套餐对应的 API 地址")
	}

	// 并行拉取订阅和限额
	subscriptionCh := make(chan *glmSubscriptionResp)
	limitCh := make(chan *glmLimitResp)
	errCh := make(chan error, 2)

	go func() {
		resp, err := fetchGlmAPI(apiBase, glmSubscriptionPath, apiKey, proxyURL, nil)
		if err != nil {
			errCh <- fmt.Errorf("获取订阅信息失败: %w", err)
			return
		}
		var sub glmSubscriptionResp
		if err := common.Unmarshal(resp, &sub); err != nil {
			errCh <- fmt.Errorf("解析订阅信息失败: %w", err)
			return
		}
		subscriptionCh <- &sub
	}()

	go func() {
		resp, err := fetchGlmAPI(apiBase, glmQuotaLimitPath, apiKey, proxyURL, nil)
		if err != nil {
			errCh <- fmt.Errorf("获取限额信息失败: %w", err)
			return
		}
		var lim glmLimitResp
		if err := common.Unmarshal(resp, &lim); err != nil {
			errCh <- fmt.Errorf("解析限额信息失败: %w", err)
			return
		}
		limitCh <- &lim
	}()

	var subscription *glmSubscriptionResp
	var limits *glmLimitResp

	for i := 0; i < 2; i++ {
		select {
		case sub := <-subscriptionCh:
			subscription = sub
		case lim := <-limitCh:
			limits = lim
		case err := <-errCh:
			return nil, err
		}
	}

	return buildGlmPlanQuotaData(subscription, limits), nil
}

// getGlmApiBase 根据套餐标识返回对应的 API 基础地址
func getGlmApiBase(planBaseURL string) string {
	switch planBaseURL {
	case "glm-coding-plan-international":
		return "https://api.z.ai"
	default:
		return "https://www.bigmodel.cn"
	}
}

// fetchGlmAPI 向智谱后端发送请求，Key 由后端注入，不会暴露给客户端；
// proxyURL 非空时请求经渠道代理发出。extraHeaders 为 nil 时使用默认头，
// 与历史 fetchGlmAPI 行为一致；调用方若需要补充请求头（如 UsageActivity）
// 可通过该参数传入。
func fetchGlmAPI(baseURL, path, apiKey, proxyURL string, extraHeaders http.Header) ([]byte, error) {
	url := strings.TrimRight(baseURL, "/") + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", strings.TrimSpace(apiKey))
	if extraHeaders == nil {
		// 模拟浏览器从智谱官网发起的请求
		req.Header.Set("Referer", "https://bigmodel.cn/")
		req.Header.Set("Origin", "https://bigmodel.cn")
	} else {
		for key, values := range extraHeaders {
			if len(values) == 0 {
				continue
			}
			// Authorization 始终以参数 apiKey 为准，避免被 extraHeaders 覆盖泄露到上游
			if strings.EqualFold(key, "Authorization") {
				continue
			}
			req.Header[key] = append([]string(nil), values...)
		}
	}

	client, err := newGlmHttpClient(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("代理客户端创建失败: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, string(body))
	}

	// 智谱在 Key 无效时仍返回 HTTP 200，响应体为认证失败（code=1000）。
	// 这里返回明确的 Key 无效错误，避免上层把空数据当作用量空白展示。
	if isGlmAuthFailure(body) {
		return nil, ErrGlmKeyInvalid
	}

	return body, nil
}

// createBigModelUsageHeaders 构建访问 BigModel / Z.ai 个人套餐活跃接口所需的请求头。
// Authorization 由上游 OAuth + zcode 服务换取的 Coding Plan API Key 注入；
// Referer/Origin 模拟智谱/BigModel 官网控制台，确保与智谱官方前端请求一致。
// func 形式方便后续在不修改 fetchGlmAPI 调用的前提下扩展头字段。
func createBigModelUsageHeaders(apiKey string) http.Header {
	h := http.Header{}
	if trimmed := strings.TrimSpace(apiKey); trimmed != "" {
		h.Set("Authorization", trimmed)
	}
	h.Set("Referer", "https://bigmodel.cn/")
	h.Set("Origin", "https://bigmodel.cn")
	h.Set("Accept", "application/json")
	return h
}

// buildGlmPlanQuotaData 将原始 API 返回组装为前端展示结构
func buildGlmPlanQuotaData(sub *glmSubscriptionResp, lim *glmLimitResp) *GlmPlanQuotaData {
	data := &GlmPlanQuotaData{}

	// 解析订阅信息
	if sub != nil && len(sub.Data) > 0 {
		pkg := sub.Data[0]
		data.ProductName = pkg.ProductName
		data.ProductLevel = getGlmPackageLevel(pkg.ProductName)
		data.EffectiveDate = pkg.CurrentRenewTime
		data.ExpiryDate = pkg.NextRenewTime
		data.AutoRenew = pkg.AutoRenew != 0
	}

	if lim == nil {
		return data
	}

	hasWeekly := false
	hasCredit := false
	for _, l := range lim.Data.Limits {
		switch {
		case l.Type == glmLimitTypeCredit:
			// 积分额度（V3 套餐）：unit=3 为 5 小时积分额度，unit=6 为周积分额度
			hasCredit = true
			credit := &GlmCreditLimitInfo{
				Percentage:    l.Percentage,
				CurrentValue:  l.CurrentValue,
				Usage:         l.Usage,
				NextResetTime: string(l.NextResetTime),
				Status:        getGlmUsageStatus(l.Percentage),
			}
			if l.Unit == 6 {
				data.CreditWeeklyLimit = credit
			} else {
				data.CreditLimit = credit
			}
		case l.Type == glmLimitTypeTokens && l.Unit == 6:
			// 每周限额（V2 套餐特有）
			hasWeekly = true
			data.WeeklyLimit = &GlmLimitInfo{
				Percentage:    l.Percentage,
				NextResetTime: string(l.NextResetTime),
				Status:        getGlmUsageStatus(l.Percentage),
			}
		case l.Type == glmLimitTypeTokens:
			// 每5小时限额
			data.TokenLimit = &GlmLimitInfo{
				Percentage:    l.Percentage,
				NextResetTime: string(l.NextResetTime),
				Status:        getGlmUsageStatus(l.Percentage),
			}
		case l.Type == glmLimitTypeTime:
			// MCP工具限额
			mcp := &GlmMcpLimitInfo{
				Percentage:    l.Percentage,
				CurrentUsage:  fmt.Sprintf("%d/%d", l.CurrentValue, l.Usage),
				NextResetTime: string(l.NextResetTime),
				Status:        getGlmUsageStatus(l.Percentage),
			}
			toolNameMap := map[string]string{
				"search-prime": "联网搜索",
				"web-reader":   "网页读取",
				"zread":        "开源仓库",
			}
			for _, detail := range l.UsageDetails {
				name := detail.ModelCode
				if mapped, ok := toolNameMap[detail.ModelCode]; ok {
					name = mapped
				}
				mcp.Tools = append(mcp.Tools, GlmToolDetail{
					Name:  name,
					Usage: detail.Usage,
				})
			}
			data.McpToolLimit = mcp
		}
	}

	// 判定套餐代次：与智谱官方前端一致，limits 含 CREDIT_LIMIT 条目即为 V3 积分套餐；
	// 否则按是否含每周额度（unit=6）区分 V2 / V1。
	data.IsCreditPlan = hasCredit
	switch {
	case hasCredit:
		data.PlanVersion = glmPlanVersion3
	case hasWeekly:
		data.PlanVersion = glmPlanVersion2
	default:
		data.PlanVersion = glmPlanVersion1
	}

	return data
}

// getGlmPackageLevel 根据产品名推断套餐等级
func getGlmPackageLevel(productName string) string {
	name := strings.ToLower(productName)
	if strings.Contains(name, "lite") || strings.Contains(name, "基础") {
		return "Lite"
	}
	if strings.Contains(name, "pro") || strings.Contains(name, "专业") {
		return "Pro"
	}
	if strings.Contains(name, "max") || strings.Contains(name, "旗舰") || strings.Contains(name, "企业") {
		return "Max"
	}
	return "Standard"
}

// getGlmUsageStatus 根据百分比返回充裕/适中/紧张
func getGlmUsageStatus(percentage int) string {
	if percentage >= 80 {
		return "紧张"
	}
	if percentage >= 50 {
		return "适中"
	}
	return "充裕"
}

// FetchGlmUsageData 代理拉取 GLM 用量图表数据，直接透传原始 JSON；
// proxyURL 非空时请求经渠道代理发出。
func FetchGlmUsageData(apiKey string, planBaseURL string, dataType string, startTime string, endTime string, proxyURL string) (json.RawMessage, error) {
	apiBase := getGlmApiBase(planBaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("无法确定套餐对应的 API 地址")
	}

	var path string
	switch dataType {
	case "model":
		path = "/api/monitor/usage/model-usage"
	case "tool":
		path = "/api/monitor/usage/tool-usage"
	case "performance":
		path = "/api/monitor/usage/model-performance-day"
	default:
		return nil, fmt.Errorf("不支持的数据类型: %s", dataType)
	}

	if startTime != "" && endTime != "" {
		path += fmt.Sprintf("?startTime=%s&endTime=%s", url.QueryEscape(startTime), url.QueryEscape(endTime))
	}

	body, err := fetchGlmAPI(apiBase, path, apiKey, proxyURL, nil)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}

// glmActivityResp 智谱 credit-usage/activity 接口响应结构。
// 智谱在 success 字段缺失或为 true 且 code 为 0/200/null 时视为成功；
// 与 glmBizResp 不同之处在于该接口的成功判定更宽松，因此单独建模。
type glmActivityResp struct {
	Code    *int               `json:"code"`
	Msg     string             `json:"msg"`
	Success *bool              `json:"success"`
	Data    *GlmActivityPayload `json:"data"`
}

// GlmActivityPayload 智谱 credit-usage/activity 接口 data 字段结构。
type GlmActivityPayload struct {
	Summary *GlmActivitySummary `json:"summary"`
	Series  []GlmActivityDay    `json:"series"`
}

// GlmActivitySummary 智谱活跃数据汇总：累计 Token、峰值、累计使用时长与连续天数。
type GlmActivitySummary struct {
	TotalTokens          int64   `json:"totalTokens"`
	PeakDailyTokens      int64   `json:"peakDailyTokens"`
	PeakDailyTokensDate  string  `json:"peakDailyTokensDate"`
	TotalUsageDurationMs float64 `json:"totalUsageDurationMs"`
	CurrentStreakDays    int     `json:"currentStreakDays"`
	LongestStreakDays    int     `json:"longestStreakDays"`
}

// GlmActivityDay 单日 Token 活动条目，供前端日历热力图直接消费。
type GlmActivityDay struct {
	Date           string `json:"date"`
	TotalTokens    int64  `json:"totalTokens"`
	ModelCallCount int64  `json:"modelCallCount"`
	MCPCalls       int64  `json:"mcpCalls"`
}

// GlmPlanActivityData 智谱个人套餐活跃数据展示结构，与 GlmPlanQuotaData 风格保持一致。
// Series 缺失时（账号无活跃活动）保留为 nil，前端按空数据渲染。
type GlmPlanActivityData struct {
	Summary *GlmActivitySummary `json:"summary,omitempty"`
	Series  []GlmActivityDay    `json:"series,omitempty"`
}

// isGlmActivitySuccess 按用户提供的判定规则判断 credit-usage/activity 响应是否成功：
// code 为 0/200/null 且 success !== false。缺失的指针字段视为未给出，不参与否定。
func isGlmActivitySuccess(resp *glmActivityResp) bool {
	if resp == nil {
		return false
	}
	if resp.Success != nil && !*resp.Success {
		return false
	}
	if resp.Code == nil {
		return true
	}
	switch *resp.Code {
	case 0, 200:
		return true
	default:
		return false
	}
}

// ComputeGlmActivityTimeRange 按服务器时区计算个人套餐活跃接口的查询时间窗：
// startTime = 今天 00:00:00 往前推 glmActivityLookbackDays 天，
// endTime   = 服务器当天 23:59:59。
// 输出格式与智谱官方前端保持一致（YYYY-MM-DD HH:MM:SS），由 controller 在请求时按需生成。
func ComputeGlmActivityTimeRange() (startTime, endTime string) {
	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	start := end.AddDate(0, 0, -glmActivityLookbackDays)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	return start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05")
}

// FetchGlmCreditUsageActivity 调用智谱 credit-usage/activity 接口拉取个人套餐活跃数据。
// startTime/endTime 由 controller 按服务器时区计算后传入，避免被前端改值绕开 365 天语义。
// proxyURL 非空时请求经渠道代理发出。
func FetchGlmCreditUsageActivity(apiKey string, planBaseURL string, accountType string, startTime string, endTime string, proxyURL string) (*GlmPlanActivityData, error) {
	apiBase := getGlmApiBase(planBaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("无法确定套餐对应的 API 地址")
	}
	if accountType == "" {
		accountType = glmActivityAccountPersonal
	}

	path := fmt.Sprintf("%s?type=%s&startTime=%s&endTime=%s",
		glmCreditUsageActivityPath,
		url.QueryEscape(accountType),
		url.QueryEscape(startTime),
		url.QueryEscape(endTime),
	)

	body, err := fetchGlmAPI(apiBase, path, apiKey, proxyURL, createBigModelUsageHeaders(apiKey))
	if err != nil {
		return nil, err
	}

	// 与 other glm 接口一致，智谱在 Key 无效时仍可能返回 200 + 业务 code=1000，
	// fetchGlmAPI 已把该情况升级为 ErrGlmKeyInvalid，这里在解析前再次兜底。
	if isGlmAuthFailure(body) {
		return nil, ErrGlmKeyInvalid
	}

	var resp glmActivityResp
	if err := common.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析活跃数据响应失败: %w", err)
	}
	if !isGlmActivitySuccess(&resp) {
		msg := strings.TrimSpace(resp.Msg)
		if msg == "" {
			msg = "智谱返回非成功状态"
		}
		return nil, fmt.Errorf("%s (code=%v)", msg, resp.Code)
	}

	data := &GlmPlanActivityData{}
	if resp.Data != nil {
		data.Summary = resp.Data.Summary
		data.Series = resp.Data.Series
	}
	if data.Series == nil {
		data.Series = []GlmActivityDay{}
	}
	return data, nil
}
