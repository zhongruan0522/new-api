package operation_setting

import "strings"

const (
	// Web search
	WebSearchHighTierModelPriceLow    = 10.00
	WebSearchHighTierModelPriceMedium = 10.00
	WebSearchHighTierModelPriceHigh   = 10.00
	WebSearchPriceLow                 = 25.00
	WebSearchPriceMedium              = 25.00
	WebSearchPriceHigh                = 25.00
	// File search
	FileSearchPrice = 2.5
)

const (
	// Gemini Audio Input Price
	Gemini25FlashPreviewInputAudioPrice     = 1.00
	Gemini25FlashProductionInputAudioPrice  = 1.00 // for `gemini-2.5-flash`
	Gemini25FlashLitePreviewInputAudioPrice = 0.50
	Gemini25FlashNativeAudioInputAudioPrice = 3.00
	Gemini20FlashInputAudioPrice            = 0.70
)

const (
	// Claude Web search
	ClaudeWebSearchPrice = 10.00
)

func GetClaudeWebSearchPricePerThousand() float64 {
	return ClaudeWebSearchPrice
}

func GetWebSearchPricePerThousand(modelName string, contextSize string) float64 {
	// 确定模型类型
	// https://platform.openai.com/docs/pricing Web search 价格按模型类型和 search context size 收费
	// 新版计费规则不再关联 search context size，故在const区域将各size的价格设为一致。
	// gpt-4o and gpt-4.1 models (including mini models) 等普通模型更贵，o3, o4-mini, o3-pro, and deep research models 等高级模型更便宜
	isHighTierModel := 
		strings.HasPrefix(modelName, "o3") ||
		strings.HasPrefix(modelName, "o4") ||
		strings.Contains(modelName, "deep-research")
	// 确定 search context size 对应的价格
	var priceWebSearchPerThousandCalls float64
	switch contextSize {
	case "low":
		if isHighTierModel {
			priceWebSearchPerThousandCalls = WebSearchHighTierModelPriceLow
		} else {
			priceWebSearchPerThousandCalls = WebSearchPriceLow
		}
	case "medium":
		if isHighTierModel {
			priceWebSearchPerThousandCalls = WebSearchHighTierModelPriceMedium
		} else {
			priceWebSearchPerThousandCalls = WebSearchPriceMedium
		}
	case "high":
		if isHighTierModel {
			priceWebSearchPerThousandCalls = WebSearchHighTierModelPriceHigh
		} else {
			priceWebSearchPerThousandCalls = WebSearchPriceHigh
		}
	default:
		// search context size 默认为 medium
		if isHighTierModel {
			priceWebSearchPerThousandCalls = WebSearchHighTierModelPriceMedium
		} else {
			priceWebSearchPerThousandCalls = WebSearchPriceMedium
		}
	}
	return priceWebSearchPerThousandCalls
}

func GetFileSearchPricePerThousand() float64 {
	return FileSearchPrice
}

func GetGeminiInputAudioPricePerMillionTokens(modelName string) float64 {
	if strings.HasPrefix(modelName, "gemini-2.5-flash-preview-native-audio") {
		return Gemini25FlashNativeAudioInputAudioPrice
	} else if strings.HasPrefix(modelName, "gemini-2.5-flash-preview-lite") {
		return Gemini25FlashLitePreviewInputAudioPrice
	} else if strings.HasPrefix(modelName, "gemini-2.5-flash-preview") {
		return Gemini25FlashPreviewInputAudioPrice
	} else if strings.HasPrefix(modelName, "gemini-2.5-flash") {
		return Gemini25FlashProductionInputAudioPrice
	} else if strings.HasPrefix(modelName, "gemini-2.0-flash") {
		return Gemini20FlashInputAudioPrice
	}
	return 0
}
