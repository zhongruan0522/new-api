package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/i18n"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"
)

// 音色类型校验：只允许 preview / created。
func isValidVoiceType(t string) bool {
	return t == model.MiniMaxVoiceTypePreview || t == model.MiniMaxVoiceTypeCreated
}

// voiceListQuery 将查询参数解析为列表查询参数。
func voiceListQuery(c *gin.Context) model.MiniMaxVoiceListParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	operatorId, _ := strconv.Atoi(c.Query("operator_id"))
	startTime, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	return model.MiniMaxVoiceListParams{
		Type:       c.Query("type"),
		OperatorId: operatorId,
		VoiceId:    c.Query("voice_id"),
		StartTime:  startTime,
		EndTime:    endTime,
		Page:       page,
		PageSize:   pageSize,
	}
}

// GetMiniMaxVoices 管理员：分页查询音色记录。
func GetMiniMaxVoices(c *gin.Context) {
	params := voiceListQuery(c)
	result, err := model.ListMiniMaxVoices(params)
	if err != nil {
		common.SysError("list minimax voices failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

// miniMaxVoiceUpsertRequest 创建/更新音色请求体。
type miniMaxVoiceUpsertRequest struct {
	VoiceId    string `json:"voice_id"`
	Type       string `json:"type"`
	RedirectId string `json:"redirect_id"`
	Allowed    bool   `json:"allowed"`
	Remark     string `json:"remark"`
}

// CreateMiniMaxVoice 管理员：新建音色记录。
// 操作人 ID 记录为当前管理员，OperatorKind=admin。
func CreateMiniMaxVoice(c *gin.Context) {
	var req miniMaxVoiceUpsertRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": i18n.T(c, i18n.MsgInvalidParams)})
		return
	}
	req.VoiceId = strings.TrimSpace(req.VoiceId)
	if req.VoiceId == "" {
		common.ApiErrorI18n(c, i18n.MsgMiniMaxVoiceIDRequired)
		return
	}
	if req.Type == "" {
		req.Type = model.MiniMaxVoiceTypeCreated
	}
	if !isValidVoiceType(req.Type) {
		common.ApiErrorI18n(c, i18n.MsgMiniMaxVoiceInvalidType)
		return
	}

	// 查重：已存在则提示不合规（不暴露“重复”）。
	exists, err := model.IsMiniMaxVoiceIdExists(req.VoiceId)
	if err != nil {
		common.SysError("check minimax voice id exists failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	if exists {
		common.ApiErrorI18n(c, i18n.MsgMiniMaxVoiceInvalidID)
		return
	}

	adminId := c.GetInt("id")
	voice := &model.MiniMaxVoice{
		Type:         req.Type,
		OperatorId:   adminId,
		OperatorKind: "admin",
		VoiceId:      req.VoiceId,
		RedirectId:   strings.TrimSpace(req.RedirectId),
		Allowed:      req.Allowed,
		Remark:       req.Remark,
	}
	if err := model.InsertMiniMaxVoice(voice); err != nil {
		// 唯一约束冲突也归一为不合规。
		if isVoiceDupErr(err) {
			common.ApiErrorI18n(c, i18n.MsgMiniMaxVoiceInvalidID)
			return
		}
		common.SysError("insert minimax voice failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	service.RecordAudit(
		c,
		model.AuditModuleVoice,
		model.AuditActionCreate,
		"创建音色 "+voice.VoiceId,
		nil,
		voiceAuditMap(voice),
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    voice,
	})
}

// UpdateMiniMaxVoice Root：修改音色记录。
func UpdateMiniMaxVoice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": i18n.T(c, i18n.MsgMiniMaxVoiceInvalidID)})
		return
	}
	var req miniMaxVoiceUpsertRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": i18n.T(c, i18n.MsgInvalidParams)})
		return
	}
	if req.Type != "" && !isValidVoiceType(req.Type) {
		common.ApiErrorI18n(c, i18n.MsgMiniMaxVoiceInvalidType)
		return
	}

	before, err := model.GetMiniMaxVoiceById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": i18n.T(c, i18n.MsgMiniMaxVoiceNotFound)})
			return
		}
		common.SysError("get minimax voice by id failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	// 修改音色 ID 时需查重。
	newVoiceId := strings.TrimSpace(req.VoiceId)
	if newVoiceId != "" && newVoiceId != before.VoiceId {
		exists, derr := model.IsMiniMaxVoiceIdExists(newVoiceId)
		if derr != nil {
			common.SysError("check minimax voice id exists failed: " + derr.Error())
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
			return
		}
		if exists {
			common.ApiErrorI18n(c, i18n.MsgMiniMaxVoiceInvalidID)
			return
		}
		before.VoiceId = newVoiceId
	}
	if req.Type != "" {
		before.Type = req.Type
	}
	before.RedirectId = strings.TrimSpace(req.RedirectId)
	before.Allowed = req.Allowed
	if req.Remark != "" {
		before.Remark = req.Remark
	}
	before.UpdatedAt = time.Now().Unix()
	if err := model.UpdateMiniMaxVoice(before); err != nil {
		if isVoiceDupErr(err) {
			common.ApiErrorI18n(c, i18n.MsgMiniMaxVoiceInvalidID)
			return
		}
		common.SysError("update minimax voice failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	service.RecordAudit(
		c,
		model.AuditModuleVoice,
		model.AuditActionUpdate,
		"修改音色 "+before.VoiceId,
		nil,
		voiceAuditMap(before),
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    before,
	})
}

// DeleteMiniMaxVoice Root：删除音色记录。
func DeleteMiniMaxVoice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": i18n.T(c, i18n.MsgMiniMaxVoiceInvalidID)})
		return
	}
	before, err := model.GetMiniMaxVoiceById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": i18n.T(c, i18n.MsgMiniMaxVoiceNotFound)})
			return
		}
		common.SysError("get minimax voice by id failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	if err := model.DeleteMiniMaxVoiceById(id); err != nil {
		common.SysError("delete minimax voice by id failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(
		c,
		model.AuditModuleVoice,
		model.AuditActionDelete,
		"删除音色 "+before.VoiceId,
		voiceAuditMap(before),
		nil,
	)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func voiceAuditMap(v *model.MiniMaxVoice) map[string]interface{} {
	return map[string]interface{}{
		"id":            v.Id,
		"voice_id":      v.VoiceId,
		"type":          v.Type,
		"redirect_id":   v.RedirectId,
		"allowed":       v.Allowed,
		"operator_id":   v.OperatorId,
		"operator_kind": v.OperatorKind,
	}
}

func isVoiceDupErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint")
}
