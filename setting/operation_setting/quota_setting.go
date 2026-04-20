package operation_setting

import "github.com/zhongruan0522/new-api/setting/config"

type QuotaSetting struct {
	EnableFreeModelPreConsume  bool `json:"enable_free_model_pre_consume"`   // 是否对免费模型启用预消耗
	FreeModelPreConsumedQuota  int  `json:"free_model_pre_consumed_quota"`  // 免费模型预消耗额度（Token 数）
}

// 默认配置
var quotaSetting = QuotaSetting{
	EnableFreeModelPreConsume:  true,
	FreeModelPreConsumedQuota:  500,
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("quota_setting", &quotaSetting)
}

func GetQuotaSetting() *QuotaSetting {
	if quotaSetting.FreeModelPreConsumedQuota < 0 {
		quotaSetting.FreeModelPreConsumedQuota = 0
	}
	return &quotaSetting
}
