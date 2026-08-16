package router

import (
	"github.com/NookMux/NookMux/controller"
	"github.com/NookMux/NookMux/middleware"

	// Import oauth package to register providers via init()
	_ "github.com/NookMux/NookMux/oauth"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	anonymousRequestBodyLimit := middleware.AnonymousRequestBodyLimit()
	{
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", anonymousRequestBodyLimit, controller.PostSetup)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/status/admin_modules", middleware.AdminAuth(), controller.GetStatusAdminModules)
		// 构建版本指纹属于管理语义，仅 AdminAuth 可见，不进匿名 /api/status
		apiRouter.GET("/status/system_info", middleware.AdminAuth(), controller.GetStatusSystemInfo)
		// 面向用户的侧栏模块开关（chat/console/personal/support），登录即可见；
		// 管理段由服务端剥离，仅 admin_modules（AdminAuth）下发。
		// 响应内容随调用者角色变化，必须 no-store 防止管理员响应被共享缓存重放
		apiRouter.GET("/status/user_modules", middleware.UserAuth(), middleware.DisableCache(), controller.GetStatusUserModules)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/about", controller.GetAbout)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.HeaderNavModuleAuth("pricing"), controller.GetPricing)
		apiRouter.GET("/rankings", middleware.HeaderNavModuleAuth("rankings"), controller.GetRankings)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.ResetPassword)
		// OAuth routes - specific routes must come before :provider wildcard
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), controller.GenerateOAuthCode)
		apiRouter.POST("/oauth/email/bind", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.EmailBind)
		// Standard OAuth providers (GitHub, LinuxDO) - unified route
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimit(), controller.HandleOAuth)

		apiRouter.POST("/stripe/webhook", anonymousRequestBodyLimit, controller.StripeWebhook)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.UniversalVerify)

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.Verify2FALogin)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.GET("/logout", controller.Logout)
			userRoute.POST("/epay/notify", anonymousRequestBodyLimit, controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.GET("/self/usage_log_fields", controller.GetUsageLogFieldsVisible)
				selfRoute.PUT("/self", controller.UpdateSelf)
				selfRoute.DELETE("/self", controller.DeleteSelf)
				selfRoute.GET("/token", controller.GenerateAccessToken)
				selfRoute.GET("/passkey", controller.PasskeyStatus)
				selfRoute.POST("/passkey/register/begin", controller.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", controller.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", controller.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", controller.PasskeyVerifyFinish)
				selfRoute.PUT("/passkey/:id", controller.PasskeyUpdate)
				selfRoute.DELETE("/passkey/:id", controller.PasskeyDelete)
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
			optionRoute.GET("/value", controller.GetOptionValue)
			optionRoute.GET("/json_map", controller.GetOptionJsonMap)
			optionRoute.GET("/json_array", controller.GetOptionJsonArray)
			optionRoute.PUT("/", controller.UpdateOption)
			optionRoute.PUT("/json_map", controller.UpsertOptionJsonMapEntry)
			optionRoute.PUT("/json_array", controller.UpsertOptionJsonArrayEntry)
			optionRoute.DELETE("/json_map", controller.DeleteOptionJsonMapEntry)
			optionRoute.DELETE("/json_array", controller.DeleteOptionJsonArrayEntry)
			optionRoute.GET("/channel_affinity_cache", controller.GetChannelAffinityCacheStats)
			optionRoute.DELETE("/channel_affinity_cache", controller.ClearChannelAffinityCache)
			optionRoute.POST("/rest_model_ratio", controller.ResetModelRatio)
			optionRoute.POST("/reset_tool_billing_rules", controller.ResetToolBillingRules)
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
			channelRoute.GET("/fetch_models/:id", middleware.RootAuth(), controller.FetchUpstreamModels)
			channelRoute.POST("/fetch_models", middleware.RootAuth(), controller.FetchModels)
			channelRoute.POST("/ollama/pull", controller.OllamaPullModel)
			channelRoute.POST("/ollama/pull/stream", controller.OllamaPullModelStream)
			channelRoute.DELETE("/ollama/delete", controller.OllamaDeleteModel)
			channelRoute.GET("/ollama/version/:id", controller.OllamaVersion)
			channelRoute.POST("/batch/tag", controller.BatchSetChannelTag)
			channelRoute.GET("/tag/models", controller.GetTagModels)
			channelRoute.POST("/copy/:id", controller.CopyChannel)
			channelRoute.POST("/multi_key/manage", controller.ManageMultiKeys)
			channelRoute.POST("/test_proxy", controller.TestProxy)
			channelRoute.GET("/plan/quota/:id", controller.QueryPlanQuota)
			channelRoute.GET("/plan/glm/usage/:id", controller.QueryGlmUsage)
			channelRoute.GET("/plan/glm/risk/:id", controller.QueryRiskStatus)
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
			tokenRoute.POST("/batch/keys", controller.GetTokenKeysBatch)
			tokenRoute.POST("/:id/key", middleware.CriticalRateLimit(), controller.GetTokenKey)
			tokenRoute.PUT("/:id/key", middleware.CriticalRateLimit(), controller.ResetTokenKey)
		}

		ticketRoute := apiRouter.Group("/ticket")
		ticketRoute.Use(middleware.UserAuth())
		{
			ticketRoute.GET("/", controller.GetUserTickets)
			// 管理工单列表在路由层收紧到 AdminAuth；
			// service.ListAdminTickets 内部的角色检查保留作纵深防御
			ticketRoute.GET("/admin", middleware.AdminAuth(), controller.GetAdminTickets)
			ticketRoute.POST("/", controller.CreateTicket)
			ticketRoute.GET("/:id", controller.GetTicketDetail)
			ticketRoute.POST("/:id/reply", controller.ReplyTicket)
			ticketRoute.POST("/:id/close", controller.CloseTicket)
			ticketRoute.POST("/:id/status", controller.UpdateTicketStatus)
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
			// 完整兑换码 key 的按需查看响应必须 no-store，避免浏览器/中间代理缓存
			redemptionRoute.GET("/:id/key", middleware.DisableCache(), controller.GetRedemptionKey)
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
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminAuth(), controller.GetAllQuotaDates)
		dataRoute.GET("/users", middleware.AdminAuth(), controller.GetQuotaDataGroupByUser)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)
		dataRoute.GET("/media_convert_stats", middleware.AdminAuth(), controller.GetAllMediaConvertStats)
		dataRoute.GET("/self/media_convert_stats", middleware.UserAuth(), controller.GetUserMediaConvertStats)
		dataRoute.POST("/recalculate", middleware.AdminAuth(), controller.RecalculateQuotaData)

		dashboardRoute := apiRouter.Group("/dashboard")
		dashboardRoute.Use(middleware.AdminAuth())
		{
			dashboardRoute.GET("/config", controller.GetDashboardConfig)
			dashboardRoute.PUT("/config", controller.UpdateDashboardConfig)
			dashboardRoute.POST("/config/reset", controller.ResetDashboardConfig)
		}

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/token", middleware.TokenAuthReadOnly(), controller.GetLogByKey)
			logRoute.GET("/token/usage_log_fields", middleware.TokenAuthReadOnly(), controller.GetUsageLogFieldsVisible)
		}

		auditRoute := apiRouter.Group("/audit")
		auditRoute.Use(middleware.AdminAuth())
		{
			auditRoute.GET("/", controller.GetAuditLogs)
			auditRoute.GET("/modules", controller.GetAuditModules)
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

		dynamicRatioRoute := apiRouter.Group("/dynamic_ratio")
		{
			dynamicRatioRoute.GET("/status", middleware.UserAuth(), controller.GetDynamicRatioStatus)
			dynamicRatioRoute.GET("/rules", middleware.AdminAuth(), controller.GetDynamicRatioRules)
			dynamicRatioRoute.POST("/rules", middleware.RootAuth(), controller.CreateDynamicRatioRule)
			dynamicRatioRoute.PUT("/rules", middleware.RootAuth(), controller.UpdateDynamicRatioRule)
			dynamicRatioRoute.DELETE("/rules/:id", middleware.RootAuth(), controller.DeleteDynamicRatioRule)
			dynamicRatioRoute.PUT("/rules/reorder", middleware.RootAuth(), controller.ReorderDynamicRatioRules)
			dynamicRatioRoute.PUT("/enabled", middleware.RootAuth(), controller.SetDynamicRatioEnabled)
		}

		// 定制音色（用户侧）：上传试听 + 确认定制
		customVoiceRoute := apiRouter.Group("/custom_voice")
		customVoiceRoute.Use(middleware.UserAuth())
		{
			customVoiceRoute.GET("/tags", controller.CustomVoiceTagsHandler)
			customVoiceRoute.POST("/preview", controller.CustomVoicePreviewHandler)
			customVoiceRoute.POST("/confirm_quote", controller.CustomVoiceConfirmQuoteHandler)
			customVoiceRoute.POST("/confirm", controller.CustomVoiceConfirmHandler)
		}

		// 音色管理（管理员）：列表/新增（Admin），修改/删除（Root）
		minimaxVoiceRoute := apiRouter.Group("/minimax/voices")
		{
			minimaxVoiceRoute.GET("/", middleware.AdminAuth(), controller.GetMiniMaxVoices)
			minimaxVoiceRoute.POST("/", middleware.AdminAuth(), controller.CreateMiniMaxVoice)
			minimaxVoiceRoute.PUT("/:id", middleware.RootAuth(), controller.UpdateMiniMaxVoice)
			minimaxVoiceRoute.DELETE("/:id", middleware.RootAuth(), controller.DeleteMiniMaxVoice)
		}
	}
}
