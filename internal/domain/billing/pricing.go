package billing

import (
	"github.com/NookMux/NookMux/internal/config/ratio"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
)

// 上下文分段计费（计费 PRD 阶段 2）：档位匹配改用归一化 BillingUsage 的
// 普通输入、输出、缓存读取、缓存写入四个维度（四者之和 = TotalProcessedTokens），
// 不再依赖请求侧格式推断的语义开关。命中档位后的价格快照仍由
// ApplyContextPricingResult 写回 PriceData 并经 appendBillingInfo 写入现有
// 计费快照位置。

// ContextTokensForTier 分档匹配 tokens：普通输入 + 输出 + 缓存读取 + 缓存写入。
// Gemini toolUsePromptTokens 是审计字段，不进入分档（PRD 3.1）。
func ContextTokensForTier(bu *BillingUsage) int {
	if bu == nil {
		return 0
	}
	total := bu.TotalProcessedTokens()
	if total < 0 {
		return 0
	}
	return total
}

// ApplyContextPricingForBillingUsage 对归一化用量匹配上下文计费档位，
// 命中时把档位价格写回 priceData。
func ApplyContextPricingForBillingUsage(modelName string, bu *BillingUsage, priceData *contract.PriceData) (*contract.ContextPricingResult, bool, error) {
	contextTokens := ContextTokensForTier(bu)
	result, enabled, err := ratio.MatchContextPricingTier(modelName, contextTokens)
	if err != nil || !enabled || result == nil {
		return result, enabled, err
	}
	ratio.ApplyContextPricingResult(priceData, result)
	return result, true, nil
}
