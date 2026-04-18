package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/zhongruan0522/new-api/i18n"
)

// I18n middleware detects and sets the language preference for the request
func I18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// GetLanguage returns the current language from gin context
func GetLanguage(c *gin.Context) string {
	return i18n.DefaultLang
}
