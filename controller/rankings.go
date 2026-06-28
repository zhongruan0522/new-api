package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/setting/dashboard_setting"
)

func GetRankings(c *gin.Context) {
	dashboardConfig := dashboard_setting.GetDashboardConfig()
	if !dashboardConfig.RankingsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "排行榜功能已禁用",
			"data": map[string]interface{}{
				"models":  []map[string]interface{}{},
				"vendors": []map[string]interface{}{},
			},
		})
		return
	}

	result, err := service.GetRankingsSnapshot(c.DefaultQuery("period", "week"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}
