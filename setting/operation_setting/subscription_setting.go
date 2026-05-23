package operation_setting

import (
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/config"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
)

// SubscriptionSetting stores user subscription purchase mode and per-model billing ratios.
type SubscriptionSetting struct {
	PaymentMode string             `json:"payment_mode"`
	ModelRatios map[string]float64 `json:"model_ratios"`
}

var subscriptionSetting = SubscriptionSetting{
	PaymentMode: common.SubscriptionPaymentModeBoth,
	ModelRatios: map[string]float64{},
}

func init() {
	config.GlobalConfig.Register("subscription_setting", &subscriptionSetting)
}

func GetSubscriptionSetting() *SubscriptionSetting {
	return &subscriptionSetting
}

func NormalizeSubscriptionPaymentMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case common.SubscriptionPaymentModeBalance:
		return common.SubscriptionPaymentModeBalance
	case common.SubscriptionPaymentModeCash:
		return common.SubscriptionPaymentModeCash
	default:
		return common.SubscriptionPaymentModeBoth
	}
}

func GetSubscriptionModelRatio(modelName string) (float64, bool, string) {
	if modelName == "" {
		return 1, false, ""
	}
	normalized := ratio_setting.FormatMatchingModelName(modelName)
	if ratio, ok := subscriptionSetting.ModelRatios[modelName]; ok && ratio > 0 {
		return ratio, true, modelName
	}
	if normalized != "" {
		if ratio, ok := subscriptionSetting.ModelRatios[normalized]; ok && ratio > 0 {
			return ratio, true, normalized
		}
	}
	return 1, false, common.GetStringIfEmpty(normalized, modelName)
}
