package router

import (
	"github.com/NookMux/NookMux/internal/httpapi/controller/billing"
	"github.com/NookMux/NookMux/internal/httpapi/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetDashboardRouter(router *gin.Engine) {
	apiRouter := router.Group("/")
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	apiRouter.Use(middleware.CORS())
	apiRouter.Use(middleware.TokenAuthReadOnly())
	{
		apiRouter.GET("/dashboard/billing/subscription", billingcontroller.GetSubscription)
		apiRouter.GET("/v1/dashboard/billing/subscription", billingcontroller.GetSubscription)
		apiRouter.GET("/dashboard/billing/usage", billingcontroller.GetUsage)
		apiRouter.GET("/v1/dashboard/billing/usage", billingcontroller.GetUsage)
	}
}
