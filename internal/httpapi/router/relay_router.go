package router

import (
	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/httpapi/controller/model"
	"github.com/NookMux/NookMux/internal/httpapi/controller/playground"
	"github.com/NookMux/NookMux/internal/httpapi/controller/relay"
	"github.com/NookMux/NookMux/internal/httpapi/middleware"
	"github.com/NookMux/NookMux/internal/relay"

	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/gin-gonic/gin"
)

func SetRelayRouter(router *gin.Engine) {
	router.Use(middleware.CORS())
	router.Use(middleware.DecompressRequestMiddleware())
	router.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	router.Use(middleware.StatsMiddleware())
	// MCP / tool asset routes (signed URL, no auth).
	mcpRouter := router.Group("/mcp")
	{
		mcpRouter.GET("/image/:id", relay.RelayStoredImage)
		mcpRouter.GET("/video/:id", relay.RelayStoredVideo)
	}

	// https://platform.openai.com/docs/api-reference/introduction
	modelsRouter := router.Group("/v1/models")
	modelsRouter.Use(middleware.TokenAuth())
	{
		modelsRouter.GET("", func(c *gin.Context) {
			switch {
			case c.GetHeader("x-api-key") != "" && c.GetHeader("anthropic-version") != "":
				modelcontroller.ListModels(c, constant.ChannelTypeAnthropic)
			case c.GetHeader("x-goog-api-key") != "" || c.Query("key") != "": // 单独的适配
				modelcontroller.RetrieveModel(c, constant.ChannelTypeGemini)
			default:
				modelcontroller.ListModels(c, constant.ChannelTypeOpenAI)
			}
		})

		modelsRouter.GET("/:model", func(c *gin.Context) {
			switch {
			case c.GetHeader("x-api-key") != "" && c.GetHeader("anthropic-version") != "":
				modelcontroller.RetrieveModel(c, constant.ChannelTypeAnthropic)
			default:
				modelcontroller.RetrieveModel(c, constant.ChannelTypeOpenAI)
			}
		})
	}

	geminiRouter := router.Group("/v1beta/models")
	geminiRouter.Use(middleware.TokenAuth())
	{
		geminiRouter.GET("", func(c *gin.Context) {
			modelcontroller.ListModels(c, constant.ChannelTypeGemini)
		})
	}

	geminiCompatibleRouter := router.Group("/v1beta/openai/models")
	geminiCompatibleRouter.Use(middleware.TokenAuth())
	{
		geminiCompatibleRouter.GET("", func(c *gin.Context) {
			modelcontroller.ListModels(c, constant.ChannelTypeOpenAI)
		})
	}

	playgroundRouter := router.Group("/pg")
	playgroundRouter.Use(middleware.SystemPerformanceCheck())
	playgroundRouter.Use(middleware.UserAuth(), middleware.PlaygroundRequestContext(), middleware.ModelRequestRateLimit(), middleware.Distribute())
	{
		playgroundRouter.POST("/chat/completions", playgroundcontroller.Playground)
	}

	relayV1Router := router.Group("/v1")
	relayV1Router.Use(middleware.SystemPerformanceCheck())
	relayV1Router.Use(middleware.TokenAuth())
	relayV1Router.Use(middleware.ModelRequestRateLimit())
	{
		// WebSocket 路由（统一到 Relay）
		wsRouter := relayV1Router.Group("")
		wsRouter.Use(middleware.Distribute())
		wsRouter.GET("/realtime", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIRealtime)
		})
	}
	{
		//http router
		httpRouter := relayV1Router.Group("")
		httpRouter.Use(middleware.Distribute())

		// claude related routes
		httpRouter.POST("/messages", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatClaude)
		})

		// chat related routes
		httpRouter.POST("/completions", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAI)
		})
		httpRouter.POST("/chat/completions", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAI)
		})

		// response related routes
		httpRouter.POST("/responses", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIResponses)
		})
		httpRouter.POST("/responses/compact", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIResponsesCompaction)
		})

		// image related routes
		httpRouter.POST("/edits", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIImage)
		})
		httpRouter.POST("/images/generations", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIImage)
		})
		httpRouter.POST("/images/edits", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIImage)
		})

		// embedding related routes
		httpRouter.POST("/embeddings", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatEmbedding)
		})

		// audio related routes
		httpRouter.POST("/audio/transcriptions", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIAudio)
		})
		httpRouter.POST("/audio/translations", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIAudio)
		})
		httpRouter.POST("/audio/speech", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAIAudio)
		})

		// rerank related routes
		httpRouter.POST("/rerank", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatRerank)
		})

		// gemini relay routes
		httpRouter.POST("/engines/:model/embeddings", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatGemini)
		})
		httpRouter.POST("/models/*path", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatGemini)
		})

		// other relay routes
		httpRouter.POST("/moderations", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatOpenAI)
		})

		// not implemented
		httpRouter.POST("/images/variations", relaycontroller.RelayNotImplemented)
		httpRouter.GET("/files", relaycontroller.RelayNotImplemented)
		httpRouter.POST("/files", relaycontroller.RelayNotImplemented)
		httpRouter.DELETE("/files/:id", relaycontroller.RelayNotImplemented)
		httpRouter.GET("/files/:id", relaycontroller.RelayNotImplemented)
		httpRouter.GET("/files/:id/content", relaycontroller.RelayNotImplemented)
		httpRouter.POST("/fine-tunes", relaycontroller.RelayNotImplemented)
		httpRouter.GET("/fine-tunes", relaycontroller.RelayNotImplemented)
		httpRouter.GET("/fine-tunes/:id", relaycontroller.RelayNotImplemented)
		httpRouter.POST("/fine-tunes/:id/cancel", relaycontroller.RelayNotImplemented)
		httpRouter.GET("/fine-tunes/:id/events", relaycontroller.RelayNotImplemented)
		httpRouter.DELETE("/models/:model", relaycontroller.RelayNotImplemented)
	}

	relayGeminiRouter := router.Group("/v1beta")
	relayGeminiRouter.Use(middleware.SystemPerformanceCheck())
	relayGeminiRouter.Use(middleware.TokenAuth())
	relayGeminiRouter.Use(middleware.ModelRequestRateLimit())
	relayGeminiRouter.Use(middleware.Distribute())
	{
		// Gemini API 路径格式: /v1beta/models/{model_name}:{action}
		relayGeminiRouter.POST("/models/*path", func(c *gin.Context) {
			relaycontroller.Relay(c, relayconstant.RelayFormatGemini)
		})
	}
}
