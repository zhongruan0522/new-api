package controller

import (
	"strconv"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"

	"github.com/gin-gonic/gin"
)

// GetDynamicRatioRules 获取规则列表
func GetDynamicRatioRules(c *gin.Context) {
	rules, err := model.GetDynamicRatioRules()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rules)
}

// CreateDynamicRatioRule 创建规则
func CreateDynamicRatioRule(c *gin.Context) {
	var rule model.DynamicRatioRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := rule.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.CreateDynamicRatioRule(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshDynamicRatioCache()
	common.ApiSuccess(c, rule)
}

// UpdateDynamicRatioRule 更新规则
func UpdateDynamicRatioRule(c *gin.Context) {
	var rule model.DynamicRatioRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	if rule.Id == 0 {
		common.ApiErrorMsg(c, "规则 ID 不能为空")
		return
	}
	if err := rule.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateDynamicRatioRule(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshDynamicRatioCache()
	common.ApiSuccess(c, rule)
}

// DeleteDynamicRatioRule 删除规则
func DeleteDynamicRatioRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "无效的规则 ID")
		return
	}
	if err := model.DeleteDynamicRatioRule(id); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshDynamicRatioCache()
	common.ApiSuccess(c, nil)
}

// ReorderDynamicRatioRules 重排优先级
func ReorderDynamicRatioRules(c *gin.Context) {
	var req struct {
		Ids []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "ID 列表不能为空")
		return
	}
	if err := model.ReorderDynamicRatioRules(req.Ids); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshDynamicRatioCache()
	common.ApiSuccess(c, nil)
}

// SetDynamicRatioEnabled 全局开关
func SetDynamicRatioEnabled(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateOption("DynamicRatioEnabled", strconv.FormatBool(req.Enabled)); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// GetDynamicRatioStatus 用户端动态倍率状态
func GetDynamicRatioStatus(c *gin.Context) {
	group := c.Query("group")
	if group == "" {
		// 使用用户默认 group
		userId := c.GetInt("id")
		user, err := model.GetUserById(userId, false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		group = user.Group
	} else {
		// 验证用户是否有权访问该分组
		userId := c.GetInt("id")
		user, err := model.GetUserById(userId, false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if !service.GroupInUserUsableGroups(user.Group, group) {
			common.ApiErrorMsg(c, "无权访问该分组")
			return
		}
	}

	status := model.GetDynamicRatioStatus(group)
	common.ApiSuccess(c, status)
}
