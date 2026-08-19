package dashboard

import (
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/console"
	"github.com/NookMux/NookMux/internal/config/manager"
	"github.com/NookMux/NookMux/internal/model"
)

// MigrateFromConsoleSetting 从 console_setting 迁移面板开关到 dashboard_config
// 仅在首次启动时执行，通过 dashboard_config.migrated 标记防止重复迁移
func MigrateFromConsoleSetting() error {
	// 检查是否已迁移
	migratedValue := common.OptionMap["dashboard_config.migrated"]
	if migratedValue == "true" {
		common.SysLog("仪表板配置已迁移，跳过")
		return nil
	}

	common.SysLog("开始迁移 console_setting 面板配置到 dashboard_config...")

	cs := console.GetConsoleSetting()
	dc := GetDashboardConfig()

	// 复制面板开关
	dc.ApiInfoEnabled = cs.ApiInfoEnabled
	dc.UptimeKumaEnabled = cs.UptimeKumaEnabled
	dc.AnnouncementsEnabled = cs.AnnouncementsEnabled
	dc.FAQEnabled = cs.FAQEnabled

	// 保存到数据库
	configMap, err := manager.ConfigToMap(dc)
	if err != nil {
		common.SysError("配置序列化失败: " + err.Error())
		return err
	}

	for key, value := range configMap {
		fullKey := "dashboard_config." + key
		err := model.UpdateOption(fullKey, value)
		if err != nil {
			common.SysError("更新配置失败 " + fullKey + ": " + err.Error())
			return err
		}
	}

	// 标记已迁移
	err = model.UpdateOption("dashboard_config.migrated", "true")
	if err != nil {
		common.SysError("标记迁移状态失败: " + err.Error())
		return err
	}

	common.SysLog("仪表板配置迁移完成")
	return nil
}

// ShouldMigrate 检查是否需要执行迁移
// 判断依据：dashboard_config.migrated 不存在或为 false
func ShouldMigrate() bool {
	migratedValue := common.OptionMap["dashboard_config.migrated"]
	return migratedValue != "true"
}
