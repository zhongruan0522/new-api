package middleware

import (
	"net/http"
	"strings"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/constant"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/service"

	"github.com/gin-gonic/gin"
)

func PlaygroundRequestContext() func(c *gin.Context) {
	return func(c *gin.Context) {
		if c.GetBool("use_access_token") {
			abortWithOpenAiMessage(c, http.StatusForbidden, "暂不支持使用 access token", shared.ErrorCodeAccessDenied)
			return
		}

		playgroundRequest := &shared.PlayGroundRequest{}
		if err := common.UnmarshalBodyReusable(c, playgroundRequest); err != nil {
			statusCode := http.StatusBadRequest
			errorCode := shared.ErrorCodeInvalidRequest
			if common.IsRequestBodyTooLargeError(err) {
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

		userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
		if userGroup == "" {
			userGroup = common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
		}
		if !service.GroupInUserUsableGroups(userGroup, selectedGroup) && selectedGroup != userGroup {
			abortWithOpenAiMessage(c, http.StatusForbidden, "无权访问该分组", shared.ErrorCodeAccessDenied)
			return
		}

		common.SetContextKey(c, constant.ContextKeyUsingGroup, selectedGroup)
		common.SetContextKey(c, constant.ContextKeyTokenGroup, selectedGroup)
		c.Next()
	}
}
