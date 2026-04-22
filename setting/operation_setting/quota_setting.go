package operation_setting

import "github.com/zhongruan0522/new-api/setting/config"

type QuotaSetting struct {
	FreeModelPreConsumedQuota int `json:"free_model_pre_consumed_quota"` // 免费模型预消耗额度（Token 数），0 表示关闭
}

// 默认配置
var quotaSetting = QuotaSetting{
	FreeModelPreConsumedQuota: 500,
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
