package billingcontroller

import (
	"github.com/NookMux/NookMux/internal/config/operation"
	"github.com/NookMux/NookMux/internal/config/ratio"
	audit "github.com/NookMux/NookMux/internal/domain/audit"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	domaingroup "github.com/NookMux/NookMux/internal/domain/group"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/store/audit"
	"github.com/NookMux/NookMux/internal/store/option"
	"github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/gin-gonic/gin"
)

func GetPricing(c *gin.Context) {
	// Pricing is a shared cache. Clone every nested plan before applying caller
	// visibility rules so an anonymous response can never strip or retain data
	// for a concurrent authenticated request.
	pricing := pricingstore.ClonePricing(pricingstore.GetPricing())
	userId, exists := c.Get("id")
	var usableGroup map[string]string
	groupRatio := map[string]float64{}
	for s, f := range ratio.GetGroupRatioCopy() {
		groupRatio[s] = f
	}
	var group string
	if exists {
		user, err := userstore.GetUserCache(userId.(int))
		if err == nil {
			group = user.Group
			for g := range groupRatio {
				ratio, ok := ratio.GetGroupGroupRatio(group, g)
				if ok {
					groupRatio[g] = ratio
				}
			}
		}
	}

	usableGroup = domaingroup.GetUserUsableGroups(group)
	// check groupRatio contains usableGroup
	for group := range ratio.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	// enable_groups 是内部组分类信息，只对已识别身份的调用者返回；组件价格表
	// 也按同一组可见性裁剪，避免匿名响应泄露内部生效分组。
	for i := range pricing {
		if !exists {
			pricing[i].EnableGroup = nil
		}
		pricing[i].PricePlans = filterVisibleModelPricePlans(pricing[i].PricePlans, exists, usableGroup)
	}

	c.JSON(200, gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            pricingstore.GetVendors(),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": pricingstore.GetSupportedEndpointMap(),
		"auto_groups":        domaingroup.GetUserAutoGroup(group),
	})
}

func filterVisibleModelPricePlans(plans []contract.ModelPricePlan, authenticated bool, usableGroups map[string]string) []contract.ModelPricePlan {
	visiblePlans := make([]contract.ModelPricePlan, 0, len(plans))
	for _, plan := range plans {
		if plan.EffectiveGroup != "" {
			if !authenticated {
				continue
			}
			if _, usable := usableGroups[plan.EffectiveGroup]; !usable {
				continue
			}
		}
		visiblePlans = append(visiblePlans, plan)
	}
	return visiblePlans
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio.DefaultModelRatio2JSONString()
	err := optionstore.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	audit.RecordAudit(c, auditstore.AuditModuleOption, auditstore.AuditActionUpdate, "重置模型倍率", nil, nil)
	httpapi.ApiSuccessI18n(c, i18n.MsgPricingResetModelRatioSuccess, nil)
}

// ResetToolBillingRules restores tool_billing_setting.rules to the built-in
// default rule set. The dotted key is registered via config.GlobalConfig, so
// optionstore.UpdateOption takes care of both the DB write and the in-memory refresh.
func ResetToolBillingRules(c *gin.Context) {
	defaultStr := operation.DefaultToolBillingRules2JSONString()
	err := optionstore.UpdateOption("tool_billing_setting.rules", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	audit.RecordAudit(c, auditstore.AuditModuleOption, auditstore.AuditActionUpdate, "重置工具计费规则", nil, nil)
	httpapi.ApiSuccessI18n(c, i18n.MsgPricingResetToolBillingRulesSuccess, nil)
}
