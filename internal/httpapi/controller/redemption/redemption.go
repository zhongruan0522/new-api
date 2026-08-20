package redemptioncontroller

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	audit "github.com/NookMux/NookMux/internal/domain/audit"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/store/audit"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/log"
	"github.com/NookMux/NookMux/internal/store/redemption"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"unicode/utf8"
)

// maskRedemptionKeys 兑换码是可兑换凭证：列表/搜索/详情默认不返回完整 key，
// 需要完整 key 时走专门的查看接口（GetRedemptionKey）并记录操作日志。
func maskRedemptionKeys(redemptions []*redemptionstore.Redemption) {
	for _, r := range redemptions {
		r.Key = ""
	}
}

func GetAllRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := redemptionstore.GetAllRedemptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.SysError("failed to get all redemptions: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	maskRedemptionKeys(redemptions)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := redemptionstore.SearchRedemptions(keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.SysError("failed to search redemptions: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	maskRedemptionKeys(redemptions)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	redemption, err := redemptionstore.GetRedemptionById(id)
	if err != nil {
		common.SysError("failed to get redemption by id: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	redemption.Key = ""
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
}

// GetRedemptionKey 按需返回单个兑换码完整 key（AdminAuth + 专门查看动作），
// 并记录操作日志，避免列表批量泄露可兑换凭证。
func GetRedemptionKey(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	redemption, err := redemptionstore.GetRedemptionById(id)
	if err != nil {
		common.SysError("failed to get redemption by id: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	userstore.RecordLog(userId, logstore.LogTypeSystem, fmt.Sprintf("查看兑换码密钥 (兑换码ID: %d)", id))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": map[string]interface{}{
			"key": redemption.Key,
		},
	})
}

func AddRedemption(c *gin.Context) {
	redemption := redemptionstore.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if utf8.RuneCountInString(redemption.Name) == 0 || utf8.RuneCountInString(redemption.Name) > 20 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionNameLength)
		return
	}
	if redemption.Count <= 0 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountPositive)
		return
	}
	// 单次请求上限 100 条，防止大量并发写入导致数据库过载。
	// 如需创建更多兑换码，由前端分批调用本接口实现。
	if redemption.Count > 100 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountMax)
		return
	}
	if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	var keys []string
	// 使用事务保证原子性：整批成功或整批回滚，避免部分写入导致数据不一致
	err = dbstore.DB.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < redemption.Count; i++ {
			key := common.GetUUID()
			cleanRedemption := redemptionstore.Redemption{
				UserId:      c.GetInt("id"),
				Name:        redemption.Name,
				Key:         key,
				CreatedTime: common.GetTimestamp(),
				Quota:       redemption.Quota,
				ExpiredTime: redemption.ExpiredTime,
			}
			if err := tx.Create(&cleanRedemption).Error; err != nil {
				return err
			}
			keys = append(keys, key)
		}
		return nil
	})
	if err != nil {
		common.SysError("failed to batch insert redemptions: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgRedemptionCreateFailed),
			"data":    []string{},
		})
		return
	}
	audit.RecordAudit(c, auditstore.AuditModuleRedemption, auditstore.AuditActionCreate, "新增兑换码", nil, map[string]interface{}{"name": redemption.Name, "count": redemption.Count})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
}

func DeleteRedemption(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := redemptionstore.DeleteRedemptionById(id)
	if err != nil {
		common.SysError("failed to delete redemption: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	audit.RecordAudit(c, auditstore.AuditModuleRedemption, auditstore.AuditActionDelete, "删除兑换码", nil, map[string]interface{}{"id": id})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func UpdateRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	redemption := redemptionstore.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	cleanRedemption, err := redemptionstore.GetRedemptionById(redemption.Id)
	if err != nil {
		common.SysError("failed to get redemption by id: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	// 保存更新前快照用于审计差异对比
	originRedemption := *cleanRedemption
	if statusOnly == "" {
		if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		// If you add more fields, please also update redemption.Update()
		cleanRedemption.Name = redemption.Name
		cleanRedemption.Quota = redemption.Quota
		cleanRedemption.ExpiredTime = redemption.ExpiredTime
	}
	if statusOnly != "" {
		cleanRedemption.Status = redemption.Status
	}
	err = cleanRedemption.Update()
	if err != nil {
		common.SysError("failed to update redemption: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	audit.RecordAudit(c, auditstore.AuditModuleRedemption, auditstore.AuditActionUpdate, "修改兑换码: "+cleanRedemption.Name, originRedemption, cleanRedemption)
	// 与列表/详情口径一致：更新响应不回传完整 key，完整 key 只能通过
	// GetRedemptionKey 按需查看（留痕）获取。
	cleanRedemption.Key = ""
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
}

func DeleteInvalidRedemption(c *gin.Context) {
	rows, err := redemptionstore.DeleteInvalidRedemptions()
	if err != nil {
		common.SysError("failed to delete invalid redemptions: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	audit.RecordAudit(c, auditstore.AuditModuleRedemption, auditstore.AuditActionDelete, "删除无效兑换码", nil, nil)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
}

func validateExpiredTime(c *gin.Context, expired int64) (bool, string) {
	if expired != 0 && expired < common.GetTimestamp() {
		return false, i18n.T(c, i18n.MsgRedemptionExpireTimeInvalid)
	}
	return true, ""
}
