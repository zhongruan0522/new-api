package operation_setting

import (
	"fmt"
	"math"
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

func SubscriptionModelRatios2JSONString() string {
	bytes, err := common.Marshal(subscriptionSetting.ModelRatios)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func SanitizeSubscriptionModelRatiosJSON(jsonStr string) (string, error) {
	ratios, err := parseSubscriptionModelRatios(jsonStr)
	if err != nil {
		return "", err
	}
	bytes, err := common.Marshal(ratios)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func UpdateSubscriptionModelRatiosByJSONString(jsonStr string) error {
	ratios, err := parseSubscriptionModelRatios(jsonStr)
	if err != nil {
		return err
	}
	subscriptionSetting.ModelRatios = ratios
	return nil
}

func GetSubscriptionModelRatioCopy() map[string]float64 {
	result := make(map[string]float64, len(subscriptionSetting.ModelRatios))
	for modelName, ratio := range subscriptionSetting.ModelRatios {
		result[modelName] = ratio
	}
	return result
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

func parseSubscriptionModelRatios(jsonStr string) (map[string]float64, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		jsonStr = "{}"
	}

	raw := map[string]float64{}
	if err := common.UnmarshalJsonStr(jsonStr, &raw); err != nil {
		return nil, fmt.Errorf("套餐模型倍率必须是合法 JSON: %w", err)
	}

	ratios := make(map[string]float64, len(raw))
	for modelName, ratio := range raw {
		normalizedModel := ratio_setting.FormatMatchingModelName(strings.TrimSpace(modelName))
		if normalizedModel == "" {
			return nil, fmt.Errorf("套餐模型名称不能为空")
		}
		if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			return nil, fmt.Errorf("套餐模型 %s 的倍率必须大于 0", normalizedModel)
		}
		ratios[normalizedModel] = ratio
	}

	return ratios, nil
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
