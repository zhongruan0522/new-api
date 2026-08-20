package dbmigratecontroller

import (
	"github.com/NookMux/NookMux/internal/common"
	audit "github.com/NookMux/NookMux/internal/domain/audit"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/store/audit"
	"github.com/NookMux/NookMux/internal/store/db/migrate"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type dbPreMigrateStartRequest struct {
	TargetDSN    string `json:"target_dsn"`
	TargetLogDSN string `json:"target_log_dsn"`
	IncludeLogs  bool   `json:"include_logs"`
	Force        bool   `json:"force"`
}

func GetDBPreMigrateInfo(c *gin.Context) {
	info, err := dbmigrate.GetDBPreMigrateInfo()
	if err != nil {
		common.SysError("failed to get db pre migrate info: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, info)
}

func StartDBPreMigrate(c *gin.Context) {
	var req dbPreMigrateStartRequest
	if err := jsonx.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgInvalidParams),
		})
		return
	}

	jobID, err := dbmigrate.StartDBPreMigrate(dbmigrate.DBPreMigrateStartParams{
		TargetDSN:    strings.TrimSpace(req.TargetDSN),
		TargetLogDSN: strings.TrimSpace(req.TargetLogDSN),
		IncludeLogs:  req.IncludeLogs,
		Force:        req.Force,
	})
	if err != nil {
		common.SysError("failed to start db pre migrate: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	audit.RecordAudit(c, auditstore.AuditModuleDB, auditstore.AuditActionUpdate, "启动数据库预迁移", nil, req)
	common.ApiSuccess(c, gin.H{"job_id": jobID})
}

func GetDBPreMigrateJob(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgJobIDRequired),
		})
		return
	}

	job, ok := dbmigrate.GetDBPreMigrateJob(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgTaskNotFound),
		})
		return
	}
	common.ApiSuccess(c, &job)
}
