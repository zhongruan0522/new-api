package controller

import (
	"net/http"
	"strconv"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetAllLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), channel, group, requestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetUserLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt("id")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := model.GetUserLogs(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

// Deprecated: SearchAllLogs 已废弃，前端未使用该接口。
func SearchAllLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// Deprecated: SearchUserLogs 已废弃，前端未使用该接口。
func SearchUserLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

func GetLogByKey(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(200, gin.H{
			"success": false,
			"message": "无效的令牌",
		})
		return
	}
	logs, err := model.GetLogByTokenId(tokenId)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    logs,
	})
}

func GetLogsStat(c *gin.Context) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")

	var statData model.Stat
	if common.DataExportEnabled {
		var qStat model.QuotaStat
		var err error
		if username != "" {
			qStat, err = model.GetQuotaStatByUsername(username, startTimestamp, endTimestamp)
		} else {
			qStat, err = model.GetAllQuotaStat(startTimestamp, endTimestamp)
		}
		if err != nil {
			common.ApiError(c, err)
			return
		}
		// 从 logs 表实时查询 RPM/TPM（最近60秒），quota_data 是小时级预聚合无法提供实时指标
		rpm, tpm, err := model.QueryRpmTpm(username, tokenName, modelName, channel, group)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		statData = model.Stat{
			Quota:        qStat.Quota,
			Rpm:          rpm,
			Tpm:          tpm,
			SuccessCount: qStat.SuccessCount,
			FailCount:    qStat.FailCount,
		}
	} else {
		stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		statData = stat
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    statData,
	})
}

func GetLogsSelfStat(c *gin.Context) {
	username := c.GetString("username")
	userId := c.GetInt("id")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")

	var statData model.Stat
	if common.DataExportEnabled {
		qStat, err := model.GetQuotaStatByUserId(userId, startTimestamp, endTimestamp)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		// 从 logs 表实时查询 RPM/TPM（最近60秒）
		rpm, tpm, err := model.QueryRpmTpm(username, tokenName, modelName, channel, group)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		statData = model.Stat{
			Quota:        qStat.Quota,
			Rpm:          rpm,
			Tpm:          tpm,
			SuccessCount: qStat.SuccessCount,
			FailCount:    qStat.FailCount,
		}
	} else {
		stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		statData = stat
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    statData,
	})
}

func DeleteHistoryLogs(c *gin.Context) {
	targetTimestamp, _ := strconv.ParseInt(c.Query("target_timestamp"), 10, 64)
	cleanStoredMedia := c.Query("clean_stored_media") == "true" || c.Query("clean_stored_media") == "1" ||
		c.Query("clean_stored_images") == "true" || c.Query("clean_stored_images") == "1"
	if targetTimestamp == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "target timestamp is required",
		})
		return
	}
	count, err := model.DeleteOldLog(c.Request.Context(), targetTimestamp, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if cleanStoredMedia {
		imgCount, err := model.DeleteOldStoredImages(c.Request.Context(), targetTimestamp, 100)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		videoCount, err := model.DeleteOldStoredVideos(c.Request.Context(), targetTimestamp, 100)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"logs":          count,
				"stored_images": imgCount,
				"stored_videos": videoCount,
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
	return
}
