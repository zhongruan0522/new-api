package rankingscontroller

import (
	"net/http"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/dashboard"
	rankings "github.com/NookMux/NookMux/internal/domain/rankings"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/gin-gonic/gin"
)

func GetRankings(c *gin.Context) {
	dashboardConfig := dashboard.GetDashboardConfig()
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

	result, err := rankings.GetRankingsSnapshot(c.DefaultQuery("period", "week"))
	if err != nil {
		common.SysError("failed to get rankings snapshot: " + err.Error())
		httpapi.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}
