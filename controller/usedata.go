package controller

import (
	"net/http"
	"strconv"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/setting/dashboard_setting"

	"github.com/gin-gonic/gin"
)

func isUserQuotaRangeTooLong(startTimestamp, endTimestamp int64) bool {
	cfg := dashboard_setting.GetDashboardConfig()
	maxDays := cfg.MaxTimeRangeDays
	if maxDays <= 0 {
		// 防御性兜底：配置异常（被显式置 0/负数）时保留默认 31 天行为
		maxDays = 31
	}
	maxRange := int64(maxDays) * 24 * 60 * 60
	if startTimestamp <= 0 || endTimestamp <= 0 || endTimestamp <= startTimestamp {
		return false
	}
	return endTimestamp-startTimestamp > maxRange
}

func GetAllQuotaDates(c *gin.Context) {
	dashboardConfig := dashboard_setting.GetDashboardConfig()
	if !dashboardConfig.QuotaDataEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "配额数据功能已禁用",
			"data":    []interface{}{},
		})
		return
	}

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if isUserQuotaRangeTooLong(startTimestamp, endTimestamp) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	username := c.Query("username")
	dates, err := model.GetAllQuotaDates(startTimestamp, endTimestamp, username)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func GetQuotaDataGroupByUser(c *gin.Context) {
	dashboardConfig := dashboard_setting.GetDashboardConfig()
	if !dashboardConfig.UserAnalyticsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "用户分析功能已禁用",
			"data":    []interface{}{},
		})
		return
	}

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if isUserQuotaRangeTooLong(startTimestamp, endTimestamp) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	dates, err := model.GetQuotaDataGroupByUser(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetUserQuotaDates(c *gin.Context) {
	dashboardConfig := dashboard_setting.GetDashboardConfig()
	if !dashboardConfig.QuotaDataEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "配额数据功能已禁用",
			"data":    []interface{}{},
		})
		return
	}

	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if isUserQuotaRangeTooLong(startTimestamp, endTimestamp) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	dates, err := model.GetQuotaDataByUserId(userId, startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

// GetAllMediaConvertStats 管理员查询所有用户的图片/视频转URL统计
func GetAllMediaConvertStats(c *gin.Context) {
	dashboardConfig := dashboard_setting.GetDashboardConfig()
	if !dashboardConfig.MediaConvertStatsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "图片/视频转URL统计功能已禁用",
			"data":    map[string]interface{}{"image_count": 0, "video_count": 0},
		})
		return
	}

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	stats, err := model.GetAllMediaConvertStats(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}

// RecalculateQuotaData 管理员触发重新计算指定时间范围的数据看板
func RecalculateQuotaData(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	if startTimestamp <= 0 || endTimestamp <= 0 || endTimestamp <= startTimestamp {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的时间范围",
		})
		return
	}

	err := model.RecalculateQuotaData(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	service.ClearRankingsCache()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "重新计算完成",
	})
}

// GetUserMediaConvertStats 普通用户查询自己的图片/视频转URL统计
func GetUserMediaConvertStats(c *gin.Context) {
	dashboardConfig := dashboard_setting.GetDashboardConfig()
	if !dashboardConfig.MediaConvertStatsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "图片/视频转URL统计功能已禁用",
			"data":    map[string]interface{}{"image_count": 0, "video_count": 0},
		})
		return
	}

	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	stats, err := model.GetMediaConvertStatsByUserId(userId, startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}
