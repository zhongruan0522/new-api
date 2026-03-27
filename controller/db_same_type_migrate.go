package controller

import (
	"net/http"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/service"

	"github.com/gin-gonic/gin"
)

type dbSameTypeMigrateStartRequest struct {
	TargetDSN    string `json:"target_dsn"`
	TargetLogDSN string `json:"target_log_dsn"`
	IncludeLogs  bool   `json:"include_logs"`
	Force        bool   `json:"force"`
}

func GetDBSameTypeMigrateInfo(c *gin.Context) {
	info, err := service.GetDBSameTypeMigrateInfo()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, info)
}

func StartDBSameTypeMigrate(c *gin.Context) {
	var req dbSameTypeMigrateStartRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	jobID, err := service.StartDBSameTypeMigrate(service.DBSameTypeMigrateStartParams{
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

func GetDBSameTypeMigrateJob(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "job_id 不能为空",
		})
		return
	}

	job, ok := service.GetDBSameTypeMigrateJob(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "任务不存在",
		})
		return
	}
	common.ApiSuccess(c, job)
}
