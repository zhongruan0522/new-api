package controller

import (
	"errors"
	"fmt"
	"net/http"
	"one-api/common"
	"one-api/constant"
	"one-api/dto"
	"one-api/middleware"
	"one-api/model"
	"one-api/service"
	"one-api/setting"
	"time"

	"github.com/gin-gonic/gin"
)

func Playground(c *gin.Context) {
	var openaiErr *dto.OpenAIErrorWithStatusCode

	defer func() {
		if openaiErr != nil {
			c.JSON(openaiErr.StatusCode, gin.H{
				"error": openaiErr.Error,
			})
		}
	}()

	useAccessToken := c.GetBool("use_access_token")
	if useAccessToken {
		openaiErr = service.OpenAIErrorWrapperLocal(errors.New("暂不支持使用 access token"), "access_token_not_supported", http.StatusBadRequest)
		return
	}

	playgroundRequest := &dto.PlayGroundRequest{}
	err := common.UnmarshalBodyReusable(c, playgroundRequest)
	if err != nil {
		openaiErr = service.OpenAIErrorWrapperLocal(err, "unmarshal_request_failed", http.StatusBadRequest)
		return
	}

	if playgroundRequest.Model == "" {
		openaiErr = service.OpenAIErrorWrapperLocal(errors.New("请选择模型"), "model_required", http.StatusBadRequest)
		return
	}
	c.Set("original_model", playgroundRequest.Model)
	group := playgroundRequest.Group
	userGroup := c.GetString("group")

	if group == "" {
		group = userGroup
	} else {
		if !setting.GroupInUserUsableGroups(group) && group != userGroup {
			openaiErr = service.OpenAIErrorWrapperLocal(errors.New("无权访问该分组"), "group_not_allowed", http.StatusForbidden)
			return
		}
		c.Set("group", group)
	}

	userId := c.GetInt("id")
	//c.Set("token_name", "playground-"+group)
	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("playground-%s", group),
		Group:  group,
	}
	_ = middleware.SetupContextForToken(c, tempToken)
	_, err = getChannel(c, group, playgroundRequest.Model, 0)
	if err != nil {
		openaiErr = service.OpenAIErrorWrapperLocal(err, "get_playground_channel_failed", http.StatusInternalServerError)
		return
	}
	//middleware.SetupContextForSelectedChannel(c, channel, playgroundRequest.Model)
	common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())

	// Write user context to ensure acceptUnsetRatio is available
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		openaiErr = service.OpenAIErrorWrapperLocal(err, "get_user_cache_failed", http.StatusInternalServerError)
		return
	}
	userCache.WriteContext(c)
	Relay(c)
}
