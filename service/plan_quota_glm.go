package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
)

// GLM 套餐查询的 API 端点
const (
	glmSubscriptionPath = "/api/biz/subscription/list?pageSize=10&pageNum=1"
	glmQuotaLimitPath   = "/api/monitor/usage/quota/limit"
)

// GlmPlanQuotaData 聚合了智谱 GLM 套餐的所有可展示信息
type GlmPlanQuotaData struct {
	PlanName      string           `json:"plan_name"`
	PlanVersion   string           `json:"plan_version"` // "新" 或 "旧"，unit=6 为新套餐
	ProductLevel  string           `json:"product_level"`
	ProductName   string           `json:"product_name"`
	EffectiveDate string           `json:"effective_date"`
	ExpiryDate    string           `json:"expiry_date"`
	AutoRenew     bool             `json:"auto_renew"`
	WeeklyLimit   *GlmLimitInfo    `json:"weekly_limit,omitempty"`
	TokenLimit    *GlmLimitInfo    `json:"token_limit,omitempty"`
	McpToolLimit  *GlmMcpLimitInfo `json:"mcp_tool_limit,omitempty"`
}

// GlmLimitInfo 通用限额信息
type GlmLimitInfo struct {
	Percentage    int    `json:"percentage"`
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

// FetchGlmPlanQuota 从智谱后端拉取套餐额度数据
// apiKey: 渠道的 API Key
// baseURL: 套餐的基础 URL (glm-coding-plan 或 glm-coding-plan-international)
func FetchGlmPlanQuota(apiKey string, planBaseURL string) (*GlmPlanQuotaData, error) {
	apiBase := getGlmApiBase(planBaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("无法确定套餐对应的 API 地址")
	}

	// 并行拉取订阅和限额
	subscriptionCh := make(chan *glmSubscriptionResp)
	limitCh := make(chan *glmLimitResp)
	errCh := make(chan error, 2)

	go func() {
		resp, err := fetchGlmAPI(apiBase, glmSubscriptionPath, apiKey)
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
		resp, err := fetchGlmAPI(apiBase, glmQuotaLimitPath, apiKey)
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

// fetchGlmAPI 向智谱后端发送请求，Key 由后端注入，不会暴露给客户端
func fetchGlmAPI(baseURL, path, apiKey string) ([]byte, error) {
	url := strings.TrimRight(baseURL, "/") + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 模拟浏览器从智谱官网发起的请求
	req.Header.Set("Authorization", strings.TrimSpace(apiKey))
	req.Header.Set("Referer", "https://bigmodel.cn/")
	req.Header.Set("Origin", "https://bigmodel.cn")

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, string(body))
	}

	return io.ReadAll(res.Body)
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

	// 解析限额信息，同时判断新老套餐
	if lim != nil && len(lim.Data.Limits) > 0 {
		hasWeekly := false
		for _, l := range lim.Data.Limits {
			if l.Unit == 6 {
				hasWeekly = true
			}
			switch {
			case l.Unit == 6:
				// 每周限额（新套餐特有）
				data.WeeklyLimit = &GlmLimitInfo{
					Percentage:    l.Percentage,
					NextResetTime: string(l.NextResetTime),
					Status:        getGlmUsageStatus(l.Percentage),
				}
			case l.Type == "TOKENS_LIMIT":
				// 每5小时限额
				data.TokenLimit = &GlmLimitInfo{
					Percentage:    l.Percentage,
					NextResetTime: string(l.NextResetTime),
					Status:        getGlmUsageStatus(l.Percentage),
				}
			case l.Type == "TIME_LIMIT":
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
		if hasWeekly {
			data.PlanVersion = "新"
		} else {
			data.PlanVersion = "旧"
		}
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

// FetchGlmUsageData 代理拉取 GLM 用量图表数据，直接透传原始 JSON
func FetchGlmUsageData(apiKey string, planBaseURL string, dataType string, startTime string, endTime string) (json.RawMessage, error) {
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

	body, err := fetchGlmAPI(apiBase, path, apiKey)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}
