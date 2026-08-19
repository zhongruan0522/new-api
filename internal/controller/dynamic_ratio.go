package controller

import (
	"strconv"
	"strings"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/model"
	"github.com/NookMux/NookMux/internal/service"

	"github.com/gin-gonic/gin"
)

// GetDynamicRatioRules 获取规则列表
func GetDynamicRatioRules(c *gin.Context) {
	rules, err := model.GetDynamicRatioRules()
	if err != nil {
		common.SysError("failed to get dynamic ratio rules: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, rules)
}

// CreateDynamicRatioRule 创建规则
func CreateDynamicRatioRule(c *gin.Context) {
	var rule model.DynamicRatioRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if err := rule.Validate(); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.CreateDynamicRatioRule(&rule); err != nil {
		common.SysError("failed to create dynamic ratio rule: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	model.RefreshDynamicRatioCache()
	service.RecordAudit(c, model.AuditModuleDynamicRatio, model.AuditActionCreate, "新增动态倍率规则", nil, rule)
	common.ApiSuccess(c, rule)
}

// UpdateDynamicRatioRule 更新规则
func UpdateDynamicRatioRule(c *gin.Context) {
	var rule model.DynamicRatioRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if rule.Id == 0 {
		common.ApiErrorI18n(c, i18n.MsgDynamicRatioRuleIDRequired)
		return
	}
	if err := rule.Validate(); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	// 查询更新前的原始数据用于审计差异对比
	origin, err := model.GetDynamicRatioRuleById(rule.Id)
	if err != nil {
		common.SysError("failed to get dynamic ratio rule by id: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	if err := model.UpdateDynamicRatioRule(&rule); err != nil {
		common.SysError("failed to update dynamic ratio rule: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	model.RefreshDynamicRatioCache()
	service.RecordAudit(c, model.AuditModuleDynamicRatio, model.AuditActionUpdate, "修改动态倍率规则", origin, rule)
	common.ApiSuccess(c, rule)
}

// DeleteDynamicRatioRule 删除规则
func DeleteDynamicRatioRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgDynamicRatioInvalidRuleID)
		return
	}
	if err := model.DeleteDynamicRatioRule(id); err != nil {
		common.SysError("failed to delete dynamic ratio rule: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	model.RefreshDynamicRatioCache()
	service.RecordAudit(c, model.AuditModuleDynamicRatio, model.AuditActionDelete, "删除动态倍率规则 #"+strconv.FormatInt(id, 10), nil, map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

// ReorderDynamicRatioRules 重排优先级
func ReorderDynamicRatioRules(c *gin.Context) {
	var req struct {
		Ids []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if len(req.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgDynamicRatioIDListRequired)
		return
	}
	if err := model.ReorderDynamicRatioRules(req.Ids); err != nil {
		common.SysError("failed to reorder dynamic ratio rules: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	model.RefreshDynamicRatioCache()
	service.RecordAudit(c, model.AuditModuleDynamicRatio, model.AuditActionUpdate, "重排动态倍率规则", nil, map[string]interface{}{"ids": req.Ids})
	common.ApiSuccess(c, nil)
}

// SetDynamicRatioEnabled 全局开关
func SetDynamicRatioEnabled(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if err := model.UpdateOption("DynamicRatioEnabled", strconv.FormatBool(req.Enabled)); err != nil {
		common.SysError("failed to update DynamicRatioEnabled option: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModuleDynamicRatio, model.AuditActionUpdate, "设置动态倍率开关: "+strconv.FormatBool(req.Enabled), nil, map[string]interface{}{"enabled": req.Enabled})
	common.ApiSuccess(c, nil)
}

// GetDynamicRatioStatus 用户端动态倍率状态
func GetDynamicRatioStatus(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.SysError("failed to get user by id: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	if group != "" {
		if !service.GroupInUserUsableGroups(user.Group, group) {
			common.ApiErrorI18n(c, i18n.MsgDynamicRatioGroupForbidden)
			return
		}
		common.ApiSuccess(c, model.GetDynamicRatioStatus(group))
		return
	}

	usableGroups := service.GetUserUsableGroups(user.Group)
	groups := make([]string, 0, len(usableGroups)+1)
	for usableGroup := range usableGroups {
		groups = append(groups, usableGroup)
	}
	if user.Group != "" {
		groups = append(groups, user.Group)
	}

	status := model.GetDynamicRatioStatusForGroups(groups)
	common.ApiSuccess(c, status)
}
