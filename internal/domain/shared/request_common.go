package shared

import (
	"github.com/gin-gonic/gin"
)

type Request interface {
	GetTokenCountMeta() *TokenCountMeta
	IsStream(c *gin.Context) bool
	SetModelName(modelName string)
}

type BaseRequest struct {
}

func (b *BaseRequest) GetTokenCountMeta() *TokenCountMeta {
	return &TokenCountMeta{
		TokenType: TokenTypeTokenizer,
	}
}
func (b *BaseRequest) IsStream(c *gin.Context) bool {
	return false
}
func (b *BaseRequest) SetModelName(modelName string) {}
