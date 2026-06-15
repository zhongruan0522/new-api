package controller

import (
	"net/http"
	"strconv"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"

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
	upstreamRequestId := c.Query("upstream_request_id")
	ip := c.Query("ip")
	ua := c.Query("ua")
	xTitle := c.Query("x_title")
	httpReferer := c.Query("http_referer")
	logs, total, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), channel, group, requestId, upstreamRequestId, ip, ua, xTitle, httpReferer)
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
	upstreamRequestId := c.Query("upstream_request_id")
	ip := c.Query("ip")
	ua := c.Query("ua")
	xTitle := c.Query("x_title")
	httpReferer := c.Query("http_referer")
	logs, total, err := model.GetUserLogs(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId, upstreamRequestId, ip, ua, xTitle, httpReferer)
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
	channel, _ := strconv.Atoi(c.Query("channel"))
	filter := model.LogStatFilter{
		Username:          c.Query("username"),
		TokenName:         c.Query("token_name"),
		ModelName:         c.Query("model_name"),
		Channel:           channel,
		Group:             c.Query("group"),
		RequestId:         c.Query("request_id"),
		UpstreamRequestId: c.Query("upstream_request_id"),
		Ip:                c.Query("ip"),
		Ua:                c.Query("ua"),
		XTitle:            c.Query("x_title"),
		HttpReferer:       c.Query("http_referer"),
	}

	var statData model.Stat
	if common.DataExportEnabled && filter.TokenName == "" && filter.Channel == 0 && filter.Group == "" && filter.ModelName == "" && !filter.HasLogOnlyFilters() && logType == 0 {
		var qStat model.QuotaStat
		var err error
		if filter.Username != "" {
			qStat, err = model.GetQuotaStatByUsername(filter.Username, startTimestamp, endTimestamp)
		} else {
			qStat, err = model.GetAllQuotaStat(startTimestamp, endTimestamp)
		}
		if err != nil {
			common.ApiError(c, err)
			return
		}
		// 从 logs 表实时查询 RPM/TPM（最近60秒），quota_data 是小时级预聚合无法提供实时指标
		rpm, tpm, err := model.QueryRpmTpm(filter)
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
		stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, filter)
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
	userId := c.GetInt("id")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channel, _ := strconv.Atoi(c.Query("channel"))
	filter := model.LogStatFilter{
		Username:          c.GetString("username"),
		TokenName:         c.Query("token_name"),
		ModelName:         c.Query("model_name"),
		Channel:           channel,
		Group:             c.Query("group"),
		RequestId:         c.Query("request_id"),
		UpstreamRequestId: c.Query("upstream_request_id"),
		Ip:                c.Query("ip"),
		Ua:                c.Query("ua"),
		XTitle:            c.Query("x_title"),
		HttpReferer:       c.Query("http_referer"),
	}

	var statData model.Stat
	if common.DataExportEnabled && filter.TokenName == "" && filter.Channel == 0 && filter.Group == "" && filter.ModelName == "" && !filter.HasLogOnlyFilters() && logType == 0 {
		qStat, err := model.GetQuotaStatByUserId(userId, startTimestamp, endTimestamp)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		// 从 logs 表实时查询 RPM/TPM（最近60秒）
		rpm, tpm, err := model.QueryRpmTpm(filter)
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
		stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, filter)
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
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	cleanLogs := c.Query("clean_logs") == "true" || c.Query("clean_logs") == "1"
	cleanStoredImages := c.Query("clean_stored_images") == "true" || c.Query("clean_stored_images") == "1"
	cleanStoredVideos := c.Query("clean_stored_videos") == "true" || c.Query("clean_stored_videos") == "1"
	cleanAuditLogs := c.Query("clean_audit_logs") == "true" || c.Query("clean_audit_logs") == "1"

	if endTimestamp == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "end timestamp is required",
		})
		return
	}
	if !cleanLogs && !cleanStoredImages && !cleanStoredVideos && !cleanAuditLogs {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "at least one log type must be selected",
		})
		return
	}

	data := gin.H{}
	// detail 同时作为审计 after 数据，记录每个已选类型的实际删除数量，
	// 便于事后核对清理范围。
	detail := map[string]interface{}{
		"start_timestamp": startTimestamp,
		"end_timestamp":   endTimestamp,
	}
	// 链式执行：任一类型失败记录首个错误，继续执行其余类型，
	// 确保部分成功的清理也有审计记录。
	var firstErr error

	if cleanLogs {
		count, err := model.DeleteLogsInRange(c.Request.Context(), startTimestamp, endTimestamp, 100)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		data["logs"] = count
		detail["clean_logs"] = count
	}
	if cleanStoredImages {
		count, err := model.DeleteStoredImagesInRange(c.Request.Context(), startTimestamp, endTimestamp, 100)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		data["stored_images"] = count
		detail["clean_stored_images"] = count
	}
	if cleanStoredVideos {
		count, err := model.DeleteStoredVideosInRange(c.Request.Context(), startTimestamp, endTimestamp, 100)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		data["stored_videos"] = count
		detail["clean_stored_videos"] = count
	}
	if cleanAuditLogs {
		count, err := model.DeleteAuditLogsInRange(startTimestamp, endTimestamp)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		data["audit_logs"] = count
		detail["clean_audit_logs"] = count
	}

	// 清理日志属于关键运维操作，使用 forceRecord=true 强制审计，
	// 即使审计总开关或 log 模块被关闭也必须记录，避免审计链断裂。
	if firstErr != nil {
		detail["error"] = firstErr.Error()
	}
	service.RecordAudit(c, model.AuditModuleLog, model.AuditActionDelete, "清理历史日志", nil, detail, true)

	if firstErr != nil {
		common.ApiError(c, firstErr)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}
