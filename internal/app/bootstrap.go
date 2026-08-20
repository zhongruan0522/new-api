package app

import (
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/dashboard"
	_ "github.com/NookMux/NookMux/internal/config/performance"
	"github.com/NookMux/NookMux/internal/config/ratio"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/infra/log"
	"github.com/NookMux/NookMux/internal/service"
	"github.com/NookMux/NookMux/internal/store/db/migrate"
	"github.com/NookMux/NookMux/internal/store/option"
	"github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/joho/godotenv"
	"strings"
)

func Bootstrap() error {
	// Initialize resources here if needed
	// This is a placeholder function for future resource initialization
	err := godotenv.Load(".env")
	if err != nil {
		if common.DebugEnabled {
			common.SysLog("No .env file found, using default environment variables. If needed, please create a .env file and set the relevant variables.")
		}
	}

	// 加载环境变量
	common.InitEnv()

	log.SetupLogger()

	// Initialize model settings
	ratio.InitRatioSettings()

	service.InitHttpClient()

	service.InitTokenEncoders()

	// Initialize SQL Database
	err = dbmigrate.InitDB()
	if err != nil {
		common.FatalLog("failed to initialize database: " + err.Error())
		return err
	}

	optionstore.CheckSetup()

	// Initialize options, should after dbmigrate.InitDB()
	optionstore.InitOptionMap()

	// 迁移 console_setting 面板开关到 dashboard_config（一次性，幂等）
	// 失败仅记日志不阻断启动，console_setting 仍可作为 fallback 直到双源统一
	if err := dashboard.MigrateFromConsoleSetting(); err != nil {
		common.SysError("dashboard config migration failed: " + err.Error())
	}

	// 清理旧的磁盘缓存文件
	common.CleanupOldCacheFiles()

	// 初始化模型
	pricingstore.GetPricing()

	// Initialize SQL Database
	err = dbmigrate.InitLogDB()
	if err != nil {
		return err
	}

	// Initialize Redis
	err = common.InitRedisClient()
	if err != nil {
		return err
	}

	// 启动系统监控
	common.StartSystemMonitor()

	// Initialize i18n
	err = i18n.Init()
	if err != nil {
		common.SysError("failed to initialize i18n: " + err.Error())
		// Don't return error, i18n is not critical
	} else {
		common.SysLog("i18n initialized with languages: " + strings.Join(i18n.SupportedLanguages(), ", "))
	}

	return nil
}
