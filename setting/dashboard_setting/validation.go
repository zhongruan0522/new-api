package dashboard_setting

import (
	"errors"
	"fmt"
)

// ValidateDashboardConfig 校验仪表板配置的有效性
func ValidateDashboardConfig(config *DashboardConfig) error {
	// 刷新间隔校验：60秒 - 86400秒（1分钟 - 1天）
	if config.QuotaDataRefreshInterval < 60 || config.QuotaDataRefreshInterval > 86400 {
		return errors.New("配额数据刷新间隔必须在 60-86400 秒之间")
	}
	if config.UserAnalyticsRefreshInterval < 60 || config.UserAnalyticsRefreshInterval > 86400 {
		return errors.New("用户分析刷新间隔必须在 60-86400 秒之间")
	}
	if config.RankingsRefreshInterval < 60 || config.RankingsRefreshInterval > 86400 {
		return errors.New("排行榜刷新间隔必须在 60-86400 秒之间")
	}
	if config.UptimeKumaRefreshInterval < 30 || config.UptimeKumaRefreshInterval > 3600 {
		return errors.New("Uptime Kuma 刷新间隔必须在 30-3600 秒之间")
	}

	// 时间范围校验：1-365天
	if config.MaxTimeRangeDays < 1 || config.MaxTimeRangeDays > 365 {
		return errors.New("最大时间范围必须在 1-365 天之间")
	}
	if config.DefaultTimeRangeDays < 1 || config.DefaultTimeRangeDays > config.MaxTimeRangeDays {
		return fmt.Errorf("默认时间范围必须在 1 到 %d 天之间", config.MaxTimeRangeDays)
	}

	// 数据上限校验：1-100
	if config.RankingsModelLimit < 1 || config.RankingsModelLimit > 100 {
		return errors.New("排行榜模型数量必须在 1-100 之间")
	}
	if config.RankingsVendorLimit < 1 || config.RankingsVendorLimit > 50 {
		return errors.New("排行榜供应商数量必须在 1-50 之间")
	}
	if config.UserAnalyticsTopN < 1 || config.UserAnalyticsTopN > 100 {
		return errors.New("用户排行榜数量必须在 1-100 之间")
	}

	return nil
}

// ValidateDashboardConfigField 校验单个配置字段
// 用于 UpdateOption 场景，仅校验被修改的字段
func ValidateDashboardConfigField(field string, value interface{}) error {
	switch field {
	case "quota_data_refresh_interval":
		if v, ok := value.(int); ok {
			if v < 60 || v > 86400 {
				return errors.New("配额数据刷新间隔必须在 60-86400 秒之间")
			}
		}
	case "user_analytics_refresh_interval":
		if v, ok := value.(int); ok {
			if v < 60 || v > 86400 {
				return errors.New("用户分析刷新间隔必须在 60-86400 秒之间")
			}
		}
	case "rankings_refresh_interval":
		if v, ok := value.(int); ok {
			if v < 60 || v > 86400 {
				return errors.New("排行榜刷新间隔必须在 60-86400 秒之间")
			}
		}
	case "uptime_kuma_refresh_interval":
		if v, ok := value.(int); ok {
			if v < 30 || v > 3600 {
				return errors.New("Uptime Kuma 刷新间隔必须在 30-3600 秒之间")
			}
		}
	case "max_time_range_days":
		if v, ok := value.(int); ok {
			if v < 1 || v > 365 {
				return errors.New("最大时间范围必须在 1-365 天之间")
			}
		}
	case "default_time_range_days":
		if v, ok := value.(int); ok {
			if v < 1 || v > 365 {
				return errors.New("默认时间范围必须在 1-365 天之间")
			}
			// 需要检查是否超过 max_time_range_days
			config := GetDashboardConfig()
			if v > config.MaxTimeRangeDays {
				return fmt.Errorf("默认时间范围不能大于最大时间范围（%d 天）", config.MaxTimeRangeDays)
			}
		}
	case "rankings_model_limit":
		if v, ok := value.(int); ok {
			if v < 1 || v > 100 {
				return errors.New("排行榜模型数量必须在 1-100 之间")
			}
		}
	case "rankings_vendor_limit":
		if v, ok := value.(int); ok {
			if v < 1 || v > 50 {
				return errors.New("排行榜供应商数量必须在 1-50 之间")
			}
		}
	case "user_analytics_top_n":
		if v, ok := value.(int); ok {
			if v < 1 || v > 100 {
				return errors.New("用户排行榜数量必须在 1-100 之间")
			}
		}
	}

	return nil
}
