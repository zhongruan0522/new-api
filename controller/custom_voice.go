package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/service"
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
