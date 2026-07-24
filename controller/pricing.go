package controller

import (
	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/i18n"
	"github.com/NookMux/NookMux/model"
	"github.com/NookMux/NookMux/service"
	"github.com/NookMux/NookMux/setting/operation_setting"
	"github.com/NookMux/NookMux/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	userId, exists := c.Get("id")
	var usableGroup map[string]string
	groupRatio := map[string]float64{}
	for s, f := range ratio_setting.GetGroupRatioCopy() {
		groupRatio[s] = f
	}
	var group string
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil {
			group = user.Group
			for g := range groupRatio {
				ratio, ok := ratio_setting.GetGroupGroupRatio(group, g)
				if ok {
					groupRatio[g] = ratio
				}
			}
		}
	}

	usableGroup = service.GetUserUsableGroups(group)
	// check groupRatio contains usableGroup
	for group := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	c.JSON(200, gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            model.GetVendors(),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":        service.GetUserAutoGroup(group),
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	service.RecordAudit(c, model.AuditModuleOption, model.AuditActionUpdate, "重置模型倍率", nil, nil)
	common.ApiSuccessI18n(c, i18n.MsgPricingResetModelRatioSuccess, nil)
}

// ResetToolBillingRules restores tool_billing_setting.rules to the built-in
// default rule set. The dotted key is registered via config.GlobalConfig, so
// model.UpdateOption takes care of both the DB write and the in-memory refresh.
func ResetToolBillingRules(c *gin.Context) {
	defaultStr := operation_setting.DefaultToolBillingRules2JSONString()
	err := model.UpdateOption("tool_billing_setting.rules", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	service.RecordAudit(c, model.AuditModuleOption, model.AuditActionUpdate, "重置工具计费规则", nil, nil)
	common.ApiSuccessI18n(c, i18n.MsgPricingResetToolBillingRulesSuccess, nil)
}
