package controller

import (
	"net/http"
	"strings"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/i18n"
	"github.com/NookMux/NookMux/model"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/NookMux/NookMux/service"

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
		common.SysError("failed to get db same type migrate info: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, info)
}

func StartDBSameTypeMigrate(c *gin.Context) {
	var req dbSameTypeMigrateStartRequest
	if err := jsonx.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgInvalidParams),
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
		common.SysError("failed to start db same type migrate: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModuleDB, model.AuditActionUpdate, "启动同类型数据库迁移", nil, req)
	common.ApiSuccess(c, gin.H{"job_id": jobID})
}

func GetDBSameTypeMigrateJob(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgJobIDRequired),
		})
		return
	}

	job, ok := service.GetDBSameTypeMigrateJob(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgTaskNotFound),
		})
		return
	}
	common.ApiSuccess(c, &job)
}
