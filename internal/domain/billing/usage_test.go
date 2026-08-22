package billing

import (
	"net/http/httptest"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/domain/shared"
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
	// base = (100-40) + 40*0.5 = 80; completion = 50*3 = 150; quota = (80+150)*2*1 = 460
	if settlement.quota != 460 {
		t.Fatalf("quota = %d, want 460", settlement.quota)
	}
	if settlement.totalTokensZero {
		t.Fatal("settlement should not be flagged as empty usage")
	}
}

// Claude 语义：input_tokens 不含缓存 tokens，base 不扣除缓存，缓存按 CacheRatio 叠加。
func TestCalculateUsageClaudeSemanticKeepsCacheInBase(t *testing.T) {
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
	// base = 100 + 40*0.5 = 120; completion = 150; quota = (120+150)*2 = 540
	if settlement.quota != 540 {
		t.Fatalf("quota = %d, want 540", settlement.quota)
	}
	if !settlement.isClaudeUsageSemantic {
		t.Fatal("expected claude usage semantic to be detected")
	}
}

// 按次计费路径：ModelPrice * QuotaPerUnit * GroupRatio，token 数不参与计算。
func TestCalculateUsageUsePricePath(t *testing.T) {
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
