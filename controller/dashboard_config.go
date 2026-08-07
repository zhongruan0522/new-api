package controller

import (
	"net/http"
	"strconv"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/i18n"
	"github.com/NookMux/NookMux/model"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/setting/config"
	"github.com/NookMux/NookMux/setting/dashboard_setting"

	"github.com/gin-gonic/gin"
)

// GetDashboardConfig 获取仪表板配置
func GetDashboardConfig(c *gin.Context) {
	dashboardConfig := dashboard_setting.GetDashboardConfig()
	configMap, err := config.ConfigToMap(dashboardConfig)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgDashboardConfigGetFailed) + ": " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    configMap,
	})
}

// UpdateDashboardConfig 更新仪表板配置
func UpdateDashboardConfig(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgDashboardConfigRequestInvalid) + ": " + err.Error(),
		})
		return
	}

	if len(updates) == 0 {
		common.ApiErrorI18n(c, i18n.MsgDashboardConfigNoUpdates)
		return
	}

	// 获取更新前的配置（用于审计）
	beforeConfig := dashboard_setting.GetDashboardConfig()
	beforeMap, _ := config.ConfigToMap(beforeConfig)

	// 逐个更新并校验
	updatedFields := make(map[string]interface{})
	for key, value := range updates {
		// 类型转换处理
		var finalValue interface{}
		switch v := value.(type) {
		case float64:
			// JSON 数字默认解析为 float64，需转为 int
			finalValue = int(v)
		default:
			finalValue = v
		}

		// 字段级校验
		err := dashboard_setting.ValidateDashboardConfigField(key, finalValue)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}

		// 更新到数据库
		fullKey := "dashboard_config." + key
		var valueStr string
		switch v := finalValue.(type) {
		case bool:
			if v {
				valueStr = "true"
			} else {
				valueStr = "false"
			}
		case int:
			valueStr = strconv.Itoa(v)
		case string:
			valueStr = v
		default:
			valueStr = common.Interface2String(v)
		}

		err = model.UpdateOption(fullKey, valueStr)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgDashboardConfigUpdateFailed) + ": " + err.Error(),
			})
			return
		}

		updatedFields[key] = finalValue
	}

	// 获取更新后的配置（用于审计）
	afterConfig := dashboard_setting.GetDashboardConfig()
	afterMap, _ := config.ConfigToMap(afterConfig)

	// 记录审计日志
	service.RecordAudit(
		c,
		model.AuditModuleDashboardConfig,
		model.AuditActionUpdate,
		"更新仪表板配置",
		beforeMap,
		afterMap,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": i18n.T(c, i18n.MsgDashboardConfigUpdated),
		"data":    updatedFields,
	})
}

// ResetDashboardConfig 重置仪表板配置为默认值
func ResetDashboardConfig(c *gin.Context) {
	// 获取重置前的配置（用于审计）
	beforeConfig := dashboard_setting.GetDashboardConfig()
	beforeMap, _ := config.ConfigToMap(beforeConfig)

	// 获取默认配置
	defaultConfig := &dashboard_setting.DashboardConfig{
		QuotaDataEnabled:             true,
		UserAnalyticsEnabled:         true,
		RankingsEnabled:              true,
		MediaConvertStatsEnabled:     true,
		QuotaDataTrackTokens:         true,
		QuotaDataTrackByModel:        true,
		QuotaDataTrackByUser:         true,
		ApiInfoEnabled:               true,
		UptimeKumaEnabled:            true,
		AnnouncementsEnabled:         true,
		FAQEnabled:                   true,
		QuotaDataRefreshInterval:     3600,
		UserAnalyticsRefreshInterval: 3600,
		RankingsRefreshInterval:      300,
		UptimeKumaRefreshInterval:    60,
		DefaultTimeRangeDays:         7,
		MaxTimeRangeDays:             31,
		RankingsModelLimit:           20,
		RankingsVendorLimit:          5,
		UserAnalyticsTopN:            20,
	}

	// 保存到数据库
	configMap, err := config.ConfigToMap(defaultConfig)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgDashboardConfigSerializeFailed) + ": " + err.Error(),
		})
		return
	}

	for key, value := range configMap {
		fullKey := "dashboard_config." + key
		err := model.UpdateOption(fullKey, value)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgDashboardConfigResetFailed) + ": " + err.Error(),
			})
			return
		}
	}

	// 获取重置后的配置（用于审计）
	afterConfig := dashboard_setting.GetDashboardConfig()
	afterMap, _ := config.ConfigToMap(afterConfig)

	// 记录审计日志
	service.RecordAudit(
		c,
		model.AuditModuleDashboardConfig,
		model.AuditActionUpdate,
		"重置仪表板配置为默认值",
		beforeMap,
		afterMap,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": i18n.T(c, i18n.MsgDashboardConfigReset),
		"data":    configMap,
	})
}
