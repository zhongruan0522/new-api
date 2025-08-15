package dto

import (
	"github.com/gin-gonic/gin"
	"one-api/types"
)

type Request interface {
	GetTokenCountMeta() *types.TokenCountMeta
	IsStream(c *gin.Context) bool
}

type BaseRequest struct {
}

func (b *BaseRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
	}
}

func (b *BaseRequest) IsStream(c *gin.Context) bool {
	return false
}
