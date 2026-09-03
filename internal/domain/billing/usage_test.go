package billing

import (
	"net/http/httptest"
	"testing"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/httpapi"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/gin-gonic/gin"
)

func newUsageTestContext() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	return c
}

func newUsageTestRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName:        "gpt-4o",
		RequestConversionChain: []relayconstant.RelayFormat{relayconstant.RelayFormatOpenAI},
		UsageSource:            relayconstant.UsageSourceOpenAIChat,
		PriceData: contract.PriceData{
			ModelRatio:         2,
			CompletionRatio:    3,
			CacheRatio:         0.5,
			CacheCreationRatio: 1.25,
			GroupRatioInfo: contract.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}
}

// 常规倍率路径：prompt_tokens 含缓存 tokens 时先扣除再按 CacheRatio 计入。
func TestCalculateUsageTextQuotaWithCacheTokens(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()
	usage := &shared.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	usage.PromptTokensDetails.CachedTokens = 40

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, usage)
	if apiErr != nil {
		t.Fatalf("CalculateUsage() error = %v", apiErr)
	}
	// 普通输入 = (100-40) + 40*0.5 = 80; completion = 50*3 = 150; quota = (80+150)*2*1 = 460
	if settlement.quota != 460 {
		t.Fatalf("quota = %d, want 460", settlement.quota)
	}
	if settlement.totalTokensZero {
		t.Fatal("settlement should not be flagged as empty usage")
	}
}

// 阶段 2：usage 语义只由显式来源标识决定，请求侧格式不再参与。
// OpenAI Chat 语义（prompt_tokens 含缓存）即使客户端走 Claude 格式请求，
// 也按归一化口径扣除缓存； quota 与 PRD 3.4 OpenAI Chat 公式一致。
func TestCalculateUsageCacheSubtractionFollowsUsageSource(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()
	relayInfo.FinalRequestRelayFormat = relayconstant.RelayFormatClaude
	usage := &shared.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	usage.PromptTokensDetails.CachedTokens = 40

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, usage)
	if apiErr != nil {
		t.Fatalf("CalculateUsage() error = %v", apiErr)
	}
	// 普通输入 = (100-40) + 40*0.5 = 80; completion = 150; quota = (80+150)*2 = 460
	if settlement.quota != 460 {
		t.Fatalf("quota = %d, want 460", settlement.quota)
	}
}

// 阶段 2：携带真实 token 用量但来源未标识属于归一化失败，显式报错，
// 不再回退聚合公式静默计费。
func TestCalculateUsageMissingUsageSourceFailsExplicitly(t *testing.T) {
	relayInfo := newUsageTestRelayInfo()
	relayInfo.UsageSource = relayconstant.UsageSourceNone
	usage := &shared.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, usage)
	if apiErr == nil {
		t.Fatal("expected explicit error when usage source is not identified")
	}
	if settlement != nil {
		t.Fatalf("settlement = %#v, want nil on normalization failure", settlement)
	}
}

// 全零用量 + 来源未标识：属于"上游无 usage"而非归一化失败，
// 原生请求保持可重试语义。
func TestCalculateUsageZeroUsageWithoutSourceKeepsRetrySemantics(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()
	relayInfo.UsageSource = relayconstant.UsageSourceNone

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, &shared.Usage{})
	if apiErr == nil {
		t.Fatal("expected retryable error for native zero usage")
	}
	if settlement != nil {
		t.Fatalf("settlement = %#v, want nil on retry error", settlement)
	}
}

// 本地计数伪 usage（Gemini 流式兜底：source=Gemini 但无原始 metadata）：
// 按聚合口径计费，不因 metadata 缺失而失败，billing_details 不落列。
func TestCalculateUsageLocalCountFallbackUsesAggregate(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()
	relayInfo.UsageSource = relayconstant.UsageSourceGemini
	relayInfo.UsageGeminiMetadata = nil
	usage := &shared.Usage{PromptTokens: 0, CompletionTokens: 1400, TotalTokens: 1400}
	ctx := newUsageTestContext()
	httpapi.SetContextKey(ctx, common.ContextKeyLocalCountTokens, true)

	settlement, apiErr := CalculateUsage(ctx, relayInfo, usage)
	if apiErr != nil {
		t.Fatalf("CalculateUsage() error = %v", apiErr)
	}
	// 聚合口径：(0 + 1400×3) × 2 = 8400
	if settlement.quota != 8400 {
		t.Fatalf("quota = %d, want 8400", settlement.quota)
	}
	if settlement.billingDetailsJSON != "" {
		t.Fatalf("local-count usage must not write billing_details, got %q", settlement.billingDetailsJSON)
	}
}

// 按次计费路径：ModelPrice * QuotaPerUnit * GroupRatio，token 数不参与计算。
func TestCalculateUsageUsePricePath(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()
	relayInfo.PriceData.UsePrice = true
	relayInfo.PriceData.ModelPrice = 0.02
	usage := &shared.Usage{PromptTokens: 1000, CompletionTokens: 1000, TotalTokens: 2000}

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, usage)
	if apiErr != nil {
		t.Fatalf("CalculateUsage() error = %v", apiErr)
	}
	if settlement.quota != 10000 {
		t.Fatalf("quota = %d, want 10000", settlement.quota)
	}
}

// usage 为 nil 时按预估 prompt tokens 兜底并追加提示文案。
func TestCalculateUsageNilUsageFallsBackToEstimate(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()
	relayInfo.SetEstimatePromptTokens(30)

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, nil)
	if apiErr != nil {
		t.Fatalf("CalculateUsage() error = %v", apiErr)
	}
	// prompt=30, completion=0 → (30)*2 = 60
	if settlement.quota != 60 {
		t.Fatalf("quota = %d, want 60", settlement.quota)
	}
	if settlement.promptTokens != 30 || settlement.completionTokens != 0 {
		t.Fatalf("tokens = %d/%d, want 30/0", settlement.promptTokens, settlement.completionTokens)
	}
}

// 未发生规范转换且 totalTokens == 0：返回可重试错误，调用方不应继续落账。
func TestCalculateUsageEmptyUsageNativeReturnsRetryError(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, &shared.Usage{})
	if apiErr == nil {
		t.Fatal("expected retryable error for native empty usage")
	}
	if settlement != nil {
		t.Fatalf("settlement = %#v, want nil on retry error", settlement)
	}
}

// 发生规范转换后 totalTokens == 0：不触发重试，返回零额度结算（由 ApplyQuota 记日志）。
func TestCalculateUsageEmptyUsageConvertedContinues(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()
	relayInfo.RequestConversionChain = []relayconstant.RelayFormat{
		relayconstant.RelayFormatOpenAI,
		relayconstant.RelayFormatClaude,
	}

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, &shared.Usage{})
	if apiErr != nil {
		t.Fatalf("CalculateUsage() error = %v", apiErr)
	}
	if !settlement.totalTokensZero || settlement.hasToolFees {
		t.Fatalf("settlement flags = totalTokensZero=%v hasToolFees=%v, want true/false", settlement.totalTokensZero, settlement.hasToolFees)
	}
	if settlement.quota != 0 {
		t.Fatalf("quota = %d, want 0", settlement.quota)
	}
}

// 倍率非零但算出的额度为 0 时抬升为 1（最小计费粒度）。
func TestCalculateUsageFloorsQuotaToOneWhenRatioNonZero(t *testing.T) {
	setupEmptyExplicitPricePlans(t)
	relayInfo := newUsageTestRelayInfo()
	relayInfo.PriceData.ModelRatio = 0.4
	usage := &shared.Usage{PromptTokens: 1, CompletionTokens: 0, TotalTokens: 1}

	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, usage)
	if apiErr != nil {
		t.Fatalf("CalculateUsage() error = %v", apiErr)
	}
	// 1 token * 0.4 = 0.4，四舍五入为 0 后被地板逻辑抬升为 1
	if settlement.quota != 1 {
		t.Fatalf("quota = %d, want 1", settlement.quota)
	}
}
