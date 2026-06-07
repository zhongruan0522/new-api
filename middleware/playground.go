package middleware

import (
	"net/http"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/dto"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/types"

	"github.com/gin-gonic/gin"
)

func PlaygroundRequestContext() func(c *gin.Context) {
	return func(c *gin.Context) {
		if c.GetBool("use_access_token") {
			abortWithOpenAiMessage(c, http.StatusForbidden, "暂不支持使用 access token", types.ErrorCodeAccessDenied)
			return
		}

		playgroundRequest := &dto.PlayGroundRequest{}
		if err := common.UnmarshalBodyReusable(c, playgroundRequest); err != nil {
			statusCode := http.StatusBadRequest
			errorCode := types.ErrorCodeInvalidRequest
			if common.IsRequestBodyTooLargeError(err) {
				statusCode = http.StatusRequestEntityTooLarge
				errorCode = types.ErrorCodeReadRequestBodyFailed
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
			abortWithOpenAiMessage(c, http.StatusForbidden, "无权访问该分组", types.ErrorCodeAccessDenied)
			return
		}

		common.SetContextKey(c, constant.ContextKeyUsingGroup, selectedGroup)
		common.SetContextKey(c, constant.ContextKeyTokenGroup, selectedGroup)
		c.Next()
	}
}
