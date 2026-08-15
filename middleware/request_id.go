package middleware

import (
	"context"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/constant"
	"github.com/gin-gonic/gin"
)

func RequestId() func(c *gin.Context) {
	return func(c *gin.Context) {
		id := common.GetTimeString() + common.GetRandomString(8)
		c.Set(common.RequestIdKey, id)
		ctx := context.WithValue(c.Request.Context(), constant.ContextKeyRequestId, id)
		c.Request = c.Request.WithContext(ctx)
		c.Header(common.RequestIdKey, id)
		c.Next()
	}
}
