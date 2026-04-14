package router

import (
	"github.com/zhongruan0522/new-api/controller"
	"github.com/zhongruan0522/new-api/middleware"

	// Import oauth package to register providers via init()
	_ "github.com/zhongruan0522/new-api/oauth"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", controller.PostSetup)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/about", controller.GetAbout)
		//apiRouter.GET("/midjourney", controller.GetMidjourney)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.TryUserAuth(), controller.GetPricing)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), controller.ResetPassword)
		// OAuth routes - specific routes must come before :provider wildcard
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), controller.GenerateOAuthCode)
		apiRouter.GET("/oauth/email/bind", middleware.CriticalRateLimit(), controller.EmailBind)
		// Standard OAuth providers (GitHub, LinuxDO) - unified route
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimit(), controller.HandleOAuth)

		apiRouter.POST("/stripe/webhook", controller.StripeWebhook)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.UniversalVerify)

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), controller.Verify2FALogin)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.GET("/logout", controller.Logout)
			userRoute.POST("/epay/notify", controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)
			userRoute.GET("/groups", controller.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.PUT("/self", controller.UpdateSelf)
				selfRoute.DELETE("/self", controller.DeleteSelf)
				selfRoute.GET("/token", controller.GenerateAccessToken)
				selfRoute.GET("/passkey", controller.PasskeyStatus)
				selfRoute.POST("/passkey/register/begin", controller.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", controller.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", controller.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", controller.PasskeyVerifyFinish)
				selfRoute.DELETE("/passkey", controller.PasskeyDelete)
				selfRoute.GET("/aff", controller.GetAffCode)
				selfRoute.GET("/topup/info", controller.GetTopUpInfo)
				selfRoute.GET("/topup/self", controller.GetUserTopUps)
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), controller.TopUp)
				selfRoute.POST("/pay", middleware.CriticalRateLimit(), controller.RequestEpay)
				selfRoute.POST("/amount", controller.RequestAmount)
				selfRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.RequestStripePay)
				selfRoute.POST("/stripe/amount", controller.RequestStripeAmount)
				selfRoute.POST("/aff_transfer", controller.TransferAffQuota)
				selfRoute.PUT("/setting", controller.UpdateUserSetting)

				// 2FA routes
				selfRoute.GET("/2fa/status", controller.Get2FAStatus)
				selfRoute.POST("/2fa/setup", controller.Setup2FA)
				selfRoute.POST("/2fa/enable", controller.Enable2FA)
				selfRoute.POST("/2fa/disable", controller.Disable2FA)
				selfRoute.POST("/2fa/backup_codes", controller.RegenerateBackupCodes)

				// Check-in routes
				selfRoute.GET("/checkin", controller.GetCheckinStatus)
				selfRoute.POST("/checkin", middleware.TurnstileCheck(), controller.DoCheckin)

			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth())
			{
				adminRoute.GET("/", controller.GetAllUsers)
				adminRoute.GET("/topup", controller.GetAllTopUps)
				adminRoute.POST("/topup/complete", controller.AdminCompleteTopUp)
				adminRoute.GET("/search", controller.SearchUsers)
				adminRoute.GET("/:id", controller.GetUser)
				adminRoute.POST("/", controller.CreateUser)
				adminRoute.POST("/manage", controller.ManageUser)
				adminRoute.PUT("/", controller.UpdateUser)
				adminRoute.DELETE("/:id", controller.DeleteUser)
				adminRoute.DELETE("/:id/reset_passkey", controller.AdminResetPasskey)

				// Admin 2FA routes
				adminRoute.GET("/2fa/stats", controller.Admin2FAStats)
				adminRoute.DELETE("/:id/2fa", controller.AdminDisable2FA)
			}
		}

		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.RootAuth())
		{
			optionRoute.GET("/", controller.GetOptions)
			optionRoute.PUT("/", controller.UpdateOption)
			optionRoute.GET("/channel_affinity_cache", controller.GetChannelAffinityCacheStats)
			optionRoute.DELETE("/channel_affinity_cache", controller.ClearChannelAffinityCache)
			optionRoute.POST("/rest_model_ratio", controller.ResetModelRatio)
			optionRoute.POST("/migrate_console_setting", controller.MigrateConsoleSetting) // 用于迁移检测的旧键，下个版本会删除
		}

		dbRoute := apiRouter.Group("/db")
		dbRoute.Use(middleware.RootAuth())
		{
			dbRoute.GET("/pre_migrate/info", controller.GetDBPreMigrateInfo)
			dbRoute.POST("/pre_migrate", middleware.CriticalRateLimit(), controller.StartDBPreMigrate)
			dbRoute.GET("/pre_migrate/:id", controller.GetDBPreMigrateJob)
			dbRoute.GET("/same_type_migrate/info", controller.GetDBSameTypeMigrateInfo)
			dbRoute.POST("/same_type_migrate", middleware.CriticalRateLimit(), controller.StartDBSameTypeMigrate)
			dbRoute.GET("/same_type_migrate/:id", controller.GetDBSameTypeMigrateJob)
		}

		performanceRoute := apiRouter.Group("/performance")
		performanceRoute.Use(middleware.RootAuth())
		{
			performanceRoute.GET("/stats", controller.GetPerformanceStats)
			performanceRoute.DELETE("/disk_cache", controller.ClearDiskCache)
			performanceRoute.POST("/reset_stats", controller.ResetPerformanceStats)
			performanceRoute.POST("/gc", controller.ForceGC)
		}
		channelRoute := apiRouter.Group("/channel")
		channelRoute.Use(middleware.AdminAuth())
		{
			channelRoute.GET("/", controller.GetAllChannels)
			channelRoute.GET("/search", controller.SearchChannels)
			channelRoute.GET("/models", controller.ChannelListModels)
			channelRoute.GET("/models_enabled", controller.EnabledListModels)
			channelRoute.GET("/:id", controller.GetChannel)
			channelRoute.POST("/:id/key", middleware.RootAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.SecureVerificationRequired(), controller.GetChannelKey)
			channelRoute.GET("/test", controller.TestAllChannels)
			channelRoute.GET("/test/:id", controller.TestChannel)
			channelRoute.GET("/update_balance", controller.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", controller.UpdateChannelBalance)
			channelRoute.POST("/", controller.AddChannel)
			channelRoute.PUT("/", controller.UpdateChannel)
			channelRoute.DELETE("/disabled", controller.DeleteDisabledChannel)
			channelRoute.POST("/tag/disabled", controller.DisableTagChannels)
			channelRoute.POST("/tag/enabled", controller.EnableTagChannels)
			channelRoute.PUT("/tag", controller.EditTagChannels)
			channelRoute.DELETE("/:id", controller.DeleteChannel)
			channelRoute.POST("/batch", controller.DeleteChannelBatch)
			channelRoute.POST("/fix", controller.FixChannelsAbilities)
			channelRoute.GET("/fetch_models/:id", controller.FetchUpstreamModels)
			channelRoute.POST("/fetch_models", controller.FetchModels)
			channelRoute.POST("/ollama/pull", controller.OllamaPullModel)
			channelRoute.POST("/ollama/pull/stream", controller.OllamaPullModelStream)
			channelRoute.DELETE("/ollama/delete", controller.OllamaDeleteModel)
			channelRoute.GET("/ollama/version/:id", controller.OllamaVersion)
			channelRoute.POST("/batch/tag", controller.BatchSetChannelTag)
			channelRoute.GET("/tag/models", controller.GetTagModels)
			channelRoute.POST("/copy/:id", controller.CopyChannel)
			channelRoute.POST("/multi_key/manage", controller.ManageMultiKeys)
			channelRoute.GET("/plan/quota/:id", controller.QueryPlanQuota)
			channelRoute.GET("/plan/glm/usage/:id", controller.QueryGlmUsage)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", middleware.SearchRateLimit(), controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
			tokenRoute.POST("/batch", controller.DeleteTokenBatch)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				tokenUsageRoute.GET("/", controller.GetTokenUsage)
			}
		}

		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.AdminAuth())
		{
			redemptionRoute.GET("/", controller.GetAllRedemptions)
			redemptionRoute.GET("/search", controller.SearchRedemptions)
			redemptionRoute.GET("/:id", controller.GetRedemption)
			redemptionRoute.POST("/", controller.AddRedemption)
			redemptionRoute.PUT("/", controller.UpdateRedemption)
			redemptionRoute.DELETE("/invalid", controller.DeleteInvalidRedemption)
			redemptionRoute.DELETE("/:id", controller.DeleteRedemption)
		}
		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", middleware.AdminAuth(), controller.GetAllLogs)
		logRoute.DELETE("/", middleware.AdminAuth(), controller.DeleteHistoryLogs)
		logRoute.GET("/stat", middleware.AdminAuth(), controller.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), controller.GetLogsSelfStat)
		logRoute.GET("/channel_affinity_usage_cache", middleware.AdminAuth(), controller.GetChannelAffinityUsageCacheStats)
		logRoute.GET("/search", middleware.AdminAuth(), controller.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), middleware.SearchRateLimit(), controller.SearchUserLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminAuth(), controller.GetAllQuotaDates)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)
		dataRoute.GET("/region_stats", middleware.AdminAuth(), controller.GetAllRegionStats)
		dataRoute.GET("/self/region_stats", middleware.UserAuth(), controller.GetUserRegionStats)
		dataRoute.GET("/model_rank", middleware.AdminAuth(), controller.GetAllModelRank)
		dataRoute.GET("/self/model_rank", middleware.UserAuth(), controller.GetUserModelRank)

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/token", middleware.TokenAuthReadOnly(), controller.GetLogByKey)
		}

		storedMediaRoute := apiRouter.Group("/stored_media")
		{
			storedMediaRoute.GET("/", middleware.AdminAuth(), controller.GetAllStoredMedia)
			storedMediaRoute.GET("/self", middleware.UserAuth(), controller.GetSelfStoredMedia)
			storedMediaRoute.GET("/:media_type/:id", middleware.UserAuth(), controller.GetStoredMediaDetail)
			storedMediaRoute.DELETE("/:media_type/:id", middleware.UserAuth(), controller.DeleteStoredMedia)
			storedMediaRoute.POST("/batch", middleware.UserAuth(), controller.DeleteStoredMediaBatch)
		}
		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.AdminAuth())
		{
			groupRoute.GET("/", controller.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		prefillGroupRoute.Use(middleware.AdminAuth())
		{
			prefillGroupRoute.GET("/", controller.GetPrefillGroups)
			prefillGroupRoute.POST("/", controller.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", controller.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", controller.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/self", middleware.UserAuth(), controller.GetUserMidjourney)
		mjRoute.GET("/", middleware.AdminAuth(), controller.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/self", middleware.UserAuth(), controller.GetUserTask)
			taskRoute.GET("/", middleware.AdminAuth(), controller.GetAllTask)
		}

		vendorRoute := apiRouter.Group("/vendors")
		vendorRoute.Use(middleware.AdminAuth())
		{
			vendorRoute.GET("/", controller.GetAllVendors)
			vendorRoute.GET("/search", controller.SearchVendors)
			vendorRoute.GET("/:id", controller.GetVendorMeta)
			vendorRoute.POST("/", controller.CreateVendorMeta)
			vendorRoute.PUT("/", controller.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", controller.DeleteVendorMeta)
		}

		modelsRoute := apiRouter.Group("/models")
		modelsRoute.Use(middleware.AdminAuth())
		{
			modelsRoute.GET("/sync_upstream/preview", controller.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", controller.SyncUpstreamModels)
			modelsRoute.GET("/missing", controller.GetMissingModels)
			modelsRoute.GET("/", controller.GetAllModelsMeta)
			modelsRoute.GET("/search", controller.SearchModelsMeta)
			modelsRoute.GET("/:id", controller.GetModelMeta)
			modelsRoute.POST("/", controller.CreateModelMeta)
			modelsRoute.PUT("/", controller.UpdateModelMeta)
			modelsRoute.DELETE("/:id", controller.DeleteModelMeta)
		}
	}
}
