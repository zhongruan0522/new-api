package middleware

import (
	"net/http"
	"strings"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"

	domaingroup "github.com/NookMux/NookMux/internal/domain/group"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/infra/cache"
	"github.com/gin-gonic/gin"
)

func PlaygroundRequestContext() func(c *gin.Context) {
	return func(c *gin.Context) {
		if c.GetBool("use_access_token") {
			abortWithOpenAiMessage(c, http.StatusForbidden, "暂不支持使用 access token", shared.ErrorCodeAccessDenied)
			return
		}

		playgroundRequest := &shared.PlayGroundRequest{}
		if err := httpapi.UnmarshalBodyReusable(c, playgroundRequest); err != nil {
			statusCode := http.StatusBadRequest
			errorCode := shared.ErrorCodeInvalidRequest
			if cache.IsRequestBodyTooLargeError(err) {
				statusCode = http.StatusRequestEntityTooLarge
				errorCode = shared.ErrorCodeReadRequestBodyFailed
			}
			abortWithOpenAiMessage(c, statusCode, "无效的游乐场请求: "+err.Error(), errorCode)
			return
		}

		selectedGroup := strings.TrimSpace(playgroundRequest.Group)
		if selectedGroup == "" {
			c.Next()
			return
		}

		userGroup := httpapi.GetContextKeyString(c, common.ContextKeyUserGroup)
		if userGroup == "" {
			userGroup = httpapi.GetContextKeyString(c, common.ContextKeyUsingGroup)
		}
		if !domaingroup.GroupInUserUsableGroups(userGroup, selectedGroup) && selectedGroup != userGroup {
			abortWithOpenAiMessage(c, http.StatusForbidden, "无权访问该分组", shared.ErrorCodeAccessDenied)
			return
		}

		httpapi.SetContextKey(c, common.ContextKeyUsingGroup, selectedGroup)
		httpapi.SetContextKey(c, common.ContextKeyTokenGroup, selectedGroup)
		c.Next()
	}
}
