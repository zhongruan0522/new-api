package model

import "github.com/zhongruan0522/new-api/common"

// cleanupRemovedTokenSetting 移除已废弃的 token_setting 配置项
// 该设置已硬编码为 1000，不再从数据库读取
func cleanupRemovedTokenSetting() {
	if DB == nil {
		return
	}
	for _, key := range []string{
		"token_setting.max_user_tokens",
	} {
		if res := DB.Delete(&Option{Key: key}); res.Error != nil {
			common.SysError("failed to remove option " + key + ": " + res.Error.Error())
		} else if res.RowsAffected > 0 {
			common.SysLog("removed obsolete option: " + key)
		}
	}
}
