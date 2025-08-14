package dto

import (
	"github.com/gin-gonic/gin"
	"one-api/types"
)

type Request interface {
	GetTokenCountMeta() *types.TokenCountMeta
	IsStream(c *gin.Context) bool
}
