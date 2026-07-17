package controller

import (
	"strconv"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/i18n"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"

	"github.com/gin-gonic/gin"
)

// GetPrefillGroups 获取预填组列表，可通过 ?type=xxx 过滤
func GetPrefillGroups(c *gin.Context) {
	groupType := c.Query("type")
	groups, err := model.GetAllPrefillGroups(groupType)
	if err != nil {
		common.SysError("failed to get prefill groups: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, groups)
}

// CreatePrefillGroup 创建新的预填组
func CreatePrefillGroup(c *gin.Context) {
	var g model.PrefillGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if g.Name == "" || g.Type == "" {
		common.ApiErrorI18n(c, i18n.MsgPrefillGroupNameTypeRequired)
		return
	}
	// 创建前检查名称
	if dup, err := model.IsPrefillGroupNameDuplicated(0, g.Name); err != nil {
		common.SysError("failed to check prefill group name duplication: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	} else if dup {
		common.ApiErrorI18n(c, i18n.MsgPrefillGroupNameExists)
		return
	}

	if err := g.Insert(); err != nil {
		common.SysError("failed to insert prefill group: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModulePrefillGroup, model.AuditActionCreate, "新增预填充分组: "+g.Name, nil, g)
	common.ApiSuccess(c, &g)
}

// UpdatePrefillGroup 更新预填组
func UpdatePrefillGroup(c *gin.Context) {
	var g model.PrefillGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if g.Id == 0 {
		common.ApiErrorI18n(c, i18n.MsgPrefillGroupMissingID)
		return
	}
	// 查询更新前的原始数据用于审计差异对比
	var origin model.PrefillGroup
	if err := model.DB.First(&origin, "id = ?", g.Id).Error; err != nil {
		common.SysError("failed to get prefill group by id: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	// 名称冲突检查
	if dup, err := model.IsPrefillGroupNameDuplicated(g.Id, g.Name); err != nil {
		common.SysError("failed to check prefill group name duplication: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	} else if dup {
		common.ApiErrorI18n(c, i18n.MsgPrefillGroupNameExists)
		return
	}

	if err := g.Update(); err != nil {
		common.SysError("failed to update prefill group: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModulePrefillGroup, model.AuditActionUpdate, "修改预填充分组: "+g.Name, origin, g)
	common.ApiSuccess(c, &g)
}

// DeletePrefillGroup 删除预填组
func DeletePrefillGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.DeletePrefillGroupByID(id); err != nil {
		common.SysError("failed to delete prefill group: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModulePrefillGroup, model.AuditActionDelete, "删除预填充分组 #"+strconv.Itoa(id), nil, map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}
