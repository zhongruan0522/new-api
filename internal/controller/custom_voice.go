package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/model"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/service"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

// customVoiceConfirmRequest 确认定制请求体。
type customVoiceConfirmRequest struct {
	VoiceId string `json:"voice_id"`
}

// CustomVoicePreviewHandler 用户侧：上传音频文件 + 填写音色 ID 与试听模型，生成试听音频。
// 走 UserAuth，扣减当前登录用户额度。成功后写入“试听中”音色记录。
func CustomVoicePreviewHandler(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": i18n.T(c, i18n.MsgCustomVoiceNotLoggedIn)})
		return
	}

	// 文件从 multipart form 读取。
	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgCustomVoiceUploadAudioRequired)
		return
	}

	req := service.CustomVoicePreviewRequest{
		Model:                   c.PostForm("model"),
		VoiceId:                 c.PostForm("voice_id"),
		PreviewText:             c.PostForm("text"),
		NeedNoiseReduction:      c.PostForm("need_noise_reduction") == "true",
		NeedVolumeNormalization: c.PostForm("need_volume_normalization") == "true",
	}

	result, err := service.CustomVoicePreview(c, userId, req, fileHeader)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

// CustomVoiceConfirmQuoteHandler 用户侧：确认定制前查询本次应扣额度，不执行扣费。
func CustomVoiceConfirmQuoteHandler(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": i18n.T(c, i18n.MsgCustomVoiceNotLoggedIn)})
		return
	}

	var req customVoiceConfirmRequest
	if err := jsonx.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": i18n.T(c, i18n.MsgInvalidParams)})
		return
	}

	result, err := service.CustomVoiceConfirmQuote(c, userId, req.VoiceId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

// CustomVoiceConfirmHandler 用户侧：确认定制。扣费并把记录从“试听中”转为“已创建”。
func CustomVoiceConfirmHandler(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": i18n.T(c, i18n.MsgCustomVoiceNotLoggedIn)})
		return
	}

	var req customVoiceConfirmRequest
	if err := jsonx.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": i18n.T(c, i18n.MsgInvalidParams)})
		return
	}

	result, err := service.CustomVoiceConfirm(c, userId, req.VoiceId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

// CustomVoiceTagsHandler 用户侧：返回定制音色试听可用的情绪/语气词标签源值。
//
// 只暴露 redirect map 的 key（用户应输入的源标签），不暴露 value（上游真实标签），
// 避免前端直接展示上游标签后被现有语气词/情绪逻辑误删除。
func CustomVoiceTagsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    model.GetCustomVoiceTagsSnapshot(),
	})
}
