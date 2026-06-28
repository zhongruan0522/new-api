package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"
)

func GetChannelAffinityCacheStats(c *gin.Context) {
	stats := service.GetChannelAffinityCacheStats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}

func ClearChannelAffinityCache(c *gin.Context) {
	all := strings.TrimSpace(c.Query("all"))
	ruleName := strings.TrimSpace(c.Query("rule_name"))

	if all == "true" {
		deleted := service.ClearChannelAffinityCacheAll()
		service.RecordAudit(c, model.AuditModuleOption, model.AuditActionDelete, "清空渠道亲和缓存", nil, nil)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"deleted": deleted,
			},
		})
		return
	}

	if ruleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "缺少参数：rule_name，或使用 all=true 清空全部",
		})
		return
	}

	deleted, err := service.ClearChannelAffinityCacheByRuleName(ruleName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	service.RecordAudit(c, model.AuditModuleOption, model.AuditActionDelete, "清空渠道亲和缓存: "+ruleName, nil, map[string]interface{}{"rule_name": ruleName})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"deleted": deleted,
		},
	})
}

func GetChannelAffinityUsageCacheStats(c *gin.Context) {
	ruleName := strings.TrimSpace(c.Query("rule_name"))
	usingGroup := strings.TrimSpace(c.Query("using_group"))
	modelName := strings.TrimSpace(c.Query("model_name"))
	keyFp := strings.TrimSpace(c.Query("key_fp"))

	if ruleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "missing param: rule_name",
		})
		return
	}
	if keyFp == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "missing param: key_fp",
		})
		return
	}

	stats := service.GetChannelAffinityUsageCacheStats(ruleName, usingGroup, modelName, keyFp)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}
