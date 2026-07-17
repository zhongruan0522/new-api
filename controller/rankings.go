package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/i18n"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/setting/dashboard_setting"
)

func GetRankings(c *gin.Context) {
	dashboardConfig := dashboard_setting.GetDashboardConfig()
	if !dashboardConfig.RankingsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": i18n.T(c, i18n.MsgRankingsDisabled),
			"data": map[string]interface{}{
				"models":  []map[string]interface{}{},
				"vendors": []map[string]interface{}{},
			},
		})
		return
	}

	result, err := service.GetRankingsSnapshot(c.DefaultQuery("period", "week"))
	if err != nil {
		common.SysError("failed to get rankings snapshot: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}
