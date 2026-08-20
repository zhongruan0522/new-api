package controller

import (
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/service"
	"github.com/NookMux/NookMux/internal/store/audit"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/vendor_meta"
	"github.com/gin-gonic/gin"
	"strconv"
)

// GetAllVendors 获取供应商列表（分页）
func GetAllVendors(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	vendors, err := vendormetastore.GetAllVendors(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.SysError("failed to get all vendors: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	var total int64
	dbstore.DB.Model(&vendormetastore.Vendor{}).Count(&total)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(vendors)
	common.ApiSuccess(c, pageInfo)
}

// SearchVendors 搜索供应商
func SearchVendors(c *gin.Context) {
	keyword := c.Query("keyword")
	pageInfo := common.GetPageQuery(c)
	vendors, total, err := vendormetastore.SearchVendors(keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.SysError("failed to search vendors: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(vendors)
	common.ApiSuccess(c, pageInfo)
}

// GetVendorMeta 根据 ID 获取供应商
func GetVendorMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	v, err := vendormetastore.GetVendorByID(id)
	if err != nil {
		common.SysError("failed to get vendor by id: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, v)
}

// CreateVendorMeta 新建供应商
func CreateVendorMeta(c *gin.Context) {
	var v vendormetastore.Vendor
	if err := c.ShouldBindJSON(&v); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if v.Name == "" {
		common.ApiErrorI18n(c, i18n.MsgVendorMetaNameRequired)
		return
	}
	// 创建前先检查名称
	if dup, err := vendormetastore.IsVendorNameDuplicated(0, v.Name); err != nil {
		common.SysError("failed to check vendor name duplication: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	} else if dup {
		common.ApiErrorI18n(c, i18n.MsgVendorMetaNameExists)
		return
	}

	if err := v.Insert(); err != nil {
		common.SysError("failed to insert vendor: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, auditstore.AuditModuleVendor, auditstore.AuditActionCreate, "新增供应商: "+v.Name, nil, v)
	common.ApiSuccess(c, &v)
}

// UpdateVendorMeta 更新供应商
func UpdateVendorMeta(c *gin.Context) {
	var v vendormetastore.Vendor
	if err := c.ShouldBindJSON(&v); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if v.Id == 0 {
		common.ApiErrorI18n(c, i18n.MsgVendorMetaMissingID)
		return
	}
	// 查询更新前的原始数据用于审计差异对比
	var origin vendormetastore.Vendor
	if err := dbstore.DB.First(&origin, "id = ?", v.Id).Error; err != nil {
		common.SysError("failed to get vendor origin: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	// 名称冲突检查
	if dup, err := vendormetastore.IsVendorNameDuplicated(v.Id, v.Name); err != nil {
		common.SysError("failed to check vendor name duplication: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	} else if dup {
		common.ApiErrorI18n(c, i18n.MsgVendorMetaNameExists)
		return
	}

	if err := v.Update(); err != nil {
		common.SysError("failed to update vendor: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, auditstore.AuditModuleVendor, auditstore.AuditActionUpdate, "修改供应商: "+v.Name, origin, v)
	common.ApiSuccess(c, &v)
}

// DeleteVendorMeta 删除供应商
func DeleteVendorMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := dbstore.DB.Delete(&vendormetastore.Vendor{}, id).Error; err != nil {
		common.SysError("failed to delete vendor: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, auditstore.AuditModuleVendor, auditstore.AuditActionDelete, "删除供应商 #"+strconv.Itoa(id), nil, map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}
