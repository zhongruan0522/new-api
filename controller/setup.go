package controller

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/constant"
	"github.com/NookMux/NookMux/i18n"
	"github.com/NookMux/NookMux/model"
	"github.com/NookMux/NookMux/service"
)

// setupMutex 保护安装流程的 check-then-act 竞态：并发安装请求在检查
// constant.Setup / RootUserExists 与实际写入之间可能交错。
var setupMutex sync.Mutex

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
	// 安装流程全局只允许一个在途请求：check-then-act（检查 constant.Setup 与
	// RootUserExists 后再写入）没有锁保护时，并发安装请求可能交错创建
	// 重复的 root 用户或 setup 记录。
	setupMutex.Lock()
	defer setupMutex.Unlock()

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
			common.SysError("setup: failed to hash password: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgSetupSystemError)
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
			common.SysError("setup: failed to create root user: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgSetupCreateAdminFailed)
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
		common.SysError("setup: failed to create setup record: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgSetupInitFailed)
		return
	}

	// 系统初始化时无鉴权，手动设置操作人信息用于审计记录。
	// 审计元数据区分「新建 root 用户」与「复用已有 root 用户」两种初始化路径。
	c.Set("username", req.Username)
	service.RecordAudit(c, model.AuditModuleSetup, model.AuditActionCreate, "系统初始化", nil, map[string]interface{}{
		"username":          req.Username,
		"root_user_created": !rootExists,
	}, true)

	common.ApiSuccessI18n(c, i18n.MsgSetupInitSuccess, nil)
}
