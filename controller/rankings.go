package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/i18n"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/setting/dashboard_setting"
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
