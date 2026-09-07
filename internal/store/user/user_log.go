package userstore

import (
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/log"
)

// RecordLog / RecordLogWithAdminInfo 记录"用户动作"类系统日志。
//
// 这两个函数按 userId 反查用户名（GetUsernameById），语义上属于用户域；
// 原先位于 log 包，因 log 包查询侧还需要用户缓存反查，会与用户包互相
// 依赖，故随用户名解析一起落在用户包。查询/消费/错误类日志仍在 log 包。
func RecordLog(userId int, logType int, content string) {
	if logType == logstore.LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	logEntry := &logstore.Log{
		UserId:                userId,
		Username:              username,
		CreatedAt:             common.GetTimestamp(),
		Type:                  logType,
		Content:               content,
		BillingDetailsVersion: logstore.LogBillingDetailsVersion,
	}
	err := dbstore.LOG_DB.Create(logEntry).Error
	if err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

func RecordLogWithAdminInfo(userId int, logType int, content string, adminInfo map[string]interface{}) {
	if logType == logstore.LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	logEntry := &logstore.Log{
		UserId:                userId,
		Username:              username,
		CreatedAt:             common.GetTimestamp(),
		Type:                  logType,
		Content:               content,
		BillingDetailsVersion: logstore.LogBillingDetailsVersion,
	}
	if len(adminInfo) > 0 {
		logEntry.Other = common.MapToJsonStr(map[string]interface{}{
			"admin_info": adminInfo,
		})
	}
	err := dbstore.LOG_DB.Create(logEntry).Error
	if err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}
