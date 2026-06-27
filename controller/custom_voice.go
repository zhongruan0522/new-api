package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/setting/model_setting"
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
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	// 文件从 multipart form 读取。
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请上传音频文件"})
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

// CustomVoicePreviewAudioHandler 用户侧：代理转发 30 分钟内有效的试听音频。
func CustomVoicePreviewAudioHandler(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	recordId, err := strconv.ParseInt(c.Param("record_id"), 10, 64)
	if err != nil || recordId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的试听记录"})
		return
	}

	audio, err := service.GetCustomVoicePreviewAudio(userId, recordId)
	if err != nil {
		if errors.Is(err, service.ErrCustomVoiceDemoAudioExpired) {
			c.JSON(http.StatusGone, gin.H{"success": false, "message": "试听音频已过缓存期限"})
			return
		}
		if errors.Is(err, service.ErrCustomVoiceDemoAudioNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "试听音频不存在"})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.Data(http.StatusOK, audio.ContentType, audio.Data)
}

// CustomVoiceConfirmHandler 用户侧：确认定制。扣费并把记录从“试听中”转为“已创建”。
func CustomVoiceConfirmHandler(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	var req customVoiceConfirmRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
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
		"data":    model_setting.GetCustomVoiceTagsSnapshot(),
	})
}
