package controller

import (
	"net/http"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/service"

	"github.com/gin-gonic/gin"
)

type dbPreMigrateStartRequest struct {
	TargetDSN    string `json:"target_dsn"`
	TargetLogDSN string `json:"target_log_dsn"`
	IncludeLogs  bool   `json:"include_logs"`
	Force        bool   `json:"force"`
}

func GetDBPreMigrateInfo(c *gin.Context) {
	info, err := service.GetDBPreMigrateInfo()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, info)
}

func StartDBPreMigrate(c *gin.Context) {
	var req dbPreMigrateStartRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	jobID, err := service.StartDBPreMigrate(service.DBPreMigrateStartParams{
		TargetDSN:    strings.TrimSpace(req.TargetDSN),
		TargetLogDSN: strings.TrimSpace(req.TargetLogDSN),
		IncludeLogs:  req.IncludeLogs,
		Force:        req.Force,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"job_id": jobID})
}

func GetDBPreMigrateJob(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "job_id 不能为空",
		})
		return
	}

	job, ok := service.GetDBPreMigrateJob(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "任务不存在",
		})
		return
	}
	common.ApiSuccess(c, job)
}
