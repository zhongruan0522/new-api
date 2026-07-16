package controller

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/i18n"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"
)

type Setup struct {
	Status       bool   `json:"status"`
	RootInit     bool   `json:"root_init"`
	DatabaseType string `json:"database_type"`
}

type SetupRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

func GetSetup(c *gin.Context) {
	setup := Setup{
		Status: constant.Setup,
	}
	if constant.Setup {
		c.JSON(200, gin.H{
			"success": true,
			"data":    setup,
		})
		return
	}
	setup.RootInit = model.RootUserExists()
	if common.UsingMySQL {
		setup.DatabaseType = "mysql"
	}
	if common.UsingPostgreSQL {
		setup.DatabaseType = "postgres"
	}
	if common.UsingSQLite {
		setup.DatabaseType = "sqlite"
	}
	c.JSON(200, gin.H{
		"success": true,
		"data":    setup,
	})
}

func PostSetup(c *gin.Context) {
	// Check if setup is already completed
	if constant.Setup {
		common.ApiErrorI18n(c, i18n.MsgSetupAlreadyInitialized)
		return
	}

	// Check if root user already exists
	rootExists := model.RootUserExists()

	var req SetupRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgSetupRequestInvalid)
		return
	}

	// If root doesn't exist, validate and create admin account
	if !rootExists {
		// Validate username length: max 12 characters to align with model.User validation
		if len(req.Username) > 12 {
			common.ApiErrorI18n(c, i18n.MsgSetupUsernameTooLong)
			return
		}
		// Validate password
		if req.Password != req.ConfirmPassword {
			common.ApiErrorI18n(c, i18n.MsgSetupPasswordMismatch)
			return
		}

		if len(req.Password) < 8 {
			common.ApiErrorI18n(c, i18n.MsgSetupPasswordMin)
			return
		}

		// Create root user
		hashedPassword, err := common.Password2Hash(req.Password)
		if err != nil {
			c.JSON(200, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgSetupSystemError) + ": " + err.Error(),
			})
			return
		}
		rootUser := model.User{
			Username:    req.Username,
			Password:    hashedPassword,
			Role:        common.RoleRootUser,
			Status:      common.UserStatusEnabled,
			DisplayName: "Root User",
			AccessToken: nil,
			Quota:       100000000,
		}
		err = model.DB.Create(&rootUser).Error
		if err != nil {
			c.JSON(200, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgSetupCreateAdminFailed) + ": " + err.Error(),
			})
			return
		}
	}

	// Update setup status
	constant.Setup = true

	setup := model.Setup{
		Version:       common.Version,
		InitializedAt: time.Now().Unix(),
	}
	err = model.DB.Create(&setup).Error
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgSetupInitFailed) + ": " + err.Error(),
		})
		return
	}

	// 系统初始化时无鉴权，手动设置操作人信息用于审计记录
	c.Set("username", req.Username)
	service.RecordAudit(c, model.AuditModuleSetup, model.AuditActionCreate, "系统初始化", nil, map[string]interface{}{"username": req.Username}, true)

	common.ApiSuccessI18n(c, i18n.MsgSetupInitSuccess, nil)
}
