package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"one-api/common"
	"one-api/constant"
	"one-api/dto"
	"one-api/middleware"
	"one-api/model"
	"one-api/relay"
	relaycommon "one-api/relay/common"
	"one-api/relay/helper"
	"one-api/service"
	"one-api/types"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/gopool"

	"github.com/gin-gonic/gin"
)

func testChannel(channel *model.Channel, testModel string) (err error, newAPIError *types.NewAPIError) {
	tik := time.Now()
	if channel.Type == constant.ChannelTypeMidjourney {
		return errors.New("midjourney channel test is not supported"), nil
	}
	if channel.Type == constant.ChannelTypeMidjourneyPlus {
		return errors.New("midjourney plus channel test is not supported"), nil
	}
	if channel.Type == constant.ChannelTypeSunoAPI {
		return errors.New("suno channel test is not supported"), nil
	}
	if channel.Type == constant.ChannelTypeKling {
		return errors.New("kling channel test is not supported"), nil
	}
	if channel.Type == constant.ChannelTypeJimeng {
		return errors.New("jimeng channel test is not supported"), nil
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	requestPath := "/v1/chat/completions"

	// 先判断是否为 Embedding 模型
	if strings.Contains(strings.ToLower(testModel), "embedding") ||
		strings.HasPrefix(testModel, "m3e") || // m3e 系列模型
		strings.Contains(testModel, "bge-") || // bge 系列模型
		strings.Contains(testModel, "embed") ||
		channel.Type == constant.ChannelTypeMokaAI { // 其他 embedding 模型
		requestPath = "/v1/embeddings" // 修改请求路径
	}

	c.Request = &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: requestPath}, // 使用动态路径
		Body:   nil,
		Header: make(http.Header),
	}

	if testModel == "" {
		if channel.TestModel != nil && *channel.TestModel != "" {
			testModel = *channel.TestModel
		} else {
			if len(channel.GetModels()) > 0 {
				testModel = channel.GetModels()[0]
			} else {
				testModel = "gpt-4o-mini"
			}
		}
	}

	cache, err := model.GetUserCache(1)
	if err != nil {
		return err, nil
	}
	cache.WriteContext(c)

	c.Request.Header.Set("Authorization", "Bearer "+channel.Key)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("channel", channel.Type)
	c.Set("base_url", channel.GetBaseURL())
	group, _ := model.GetUserGroup(1, false)
	c.Set("group", group)

	middleware.SetupContextForSelectedChannel(c, channel, testModel)

	info := relaycommon.GenRelayInfo(c)

	err = helper.ModelMappedHelper(c, info, nil)
	if err != nil {
		return err, types.NewError(err, types.ErrorCodeChannelModelMappedError)
	}
	testModel = info.UpstreamModelName

	apiType, _ := common.ChannelType2APIType(channel.Type)
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		return fmt.Errorf("invalid api type: %d, adaptor is nil", apiType), types.NewError(fmt.Errorf("invalid api type: %d, adaptor is nil", apiType), types.ErrorCodeInvalidApiType)
	}

	request := buildTestRequest(testModel)
	// 创建一个用于日志的 info 副本，移除 ApiKey
	logInfo := *info
	logInfo.ApiKey = ""
	common.SysLog(fmt.Sprintf("testing channel %d with model %s , info %+v ", channel.Id, testModel, logInfo))

	priceData, err := helper.ModelPriceHelper(c, info, 0, int(request.MaxTokens))
	if err != nil {
		return err, types.NewError(err, types.ErrorCodeModelPriceError)
	}

	adaptor.Init(info)

	convertedRequest, err := adaptor.ConvertOpenAIRequest(c, info, request)
	if err != nil {
		return err, types.NewError(err, types.ErrorCodeConvertRequestFailed)
	}
	jsonData, err := json.Marshal(convertedRequest)
	if err != nil {
		return err, types.NewError(err, types.ErrorCodeJsonMarshalFailed)
	}
	requestBody := bytes.NewBuffer(jsonData)
	c.Request.Body = io.NopCloser(requestBody)
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return err, types.NewError(err, types.ErrorCodeDoRequestFailed)
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK {
			err := service.RelayErrorHandler(httpResp, true)
			return err, types.NewError(err, types.ErrorCodeBadResponse)
		}
	}
	usageA, respErr := adaptor.DoResponse(c, httpResp, info)
	if respErr != nil {
		return respErr, respErr
	}
	if usageA == nil {
		return errors.New("usage is nil"), types.NewError(errors.New("usage is nil"), types.ErrorCodeBadResponseBody)
	}
	usage := usageA.(*dto.Usage)
	result := w.Result()
	respBody, err := io.ReadAll(result.Body)
	if err != nil {
		return err, types.NewError(err, types.ErrorCodeReadResponseBodyFailed)
	}
	info.PromptTokens = usage.PromptTokens

	quota := 0
	if !priceData.UsePrice {
		quota = usage.PromptTokens + int(math.Round(float64(usage.CompletionTokens)*priceData.CompletionRatio))
		quota = int(math.Round(float64(quota) * priceData.ModelRatio))
		if priceData.ModelRatio != 0 && quota <= 0 {
			quota = 1
		}
	} else {
		quota = int(priceData.ModelPrice * common.QuotaPerUnit)
	}
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	consumedTime := float64(milliseconds) / 1000.0
	other := service.GenerateTextOtherInfo(c, info, priceData.ModelRatio, priceData.GroupRatioInfo.GroupRatio, priceData.CompletionRatio,
		usage.PromptTokensDetails.CachedTokens, priceData.CacheRatio, priceData.ModelPrice, priceData.GroupRatioInfo.GroupSpecialRatio)
	model.RecordConsumeLog(c, 1, channel.Id, usage.PromptTokens, usage.CompletionTokens, info.OriginModelName, "模型测试",
		quota, "模型测试", 0, quota, int(consumedTime), false, info.UsingGroup, other)
	common.SysLog(fmt.Sprintf("testing channel #%d, response: \n%s", channel.Id, string(respBody)))
	return nil, nil
}

func buildTestRequest(model string) *dto.GeneralOpenAIRequest {
	testRequest := &dto.GeneralOpenAIRequest{
		Model:  "", // this will be set later
		Stream: false,
	}

	// 先判断是否为 Embedding 模型
	if strings.Contains(strings.ToLower(model), "embedding") || // 其他 embedding 模型
		strings.HasPrefix(model, "m3e") || // m3e 系列模型
		strings.Contains(model, "bge-") {
		testRequest.Model = model
		// Embedding 请求
		testRequest.Input = []string{"hello world"}
		return testRequest
	}
	// 并非Embedding 模型
	if strings.HasPrefix(model, "o") {
		testRequest.MaxCompletionTokens = 10
	} else if strings.Contains(model, "thinking") {
		if !strings.Contains(model, "claude") {
			testRequest.MaxTokens = 50
		}
	} else if strings.Contains(model, "gemini") {
		testRequest.MaxTokens = 3000
	} else {
		testRequest.MaxTokens = 10
	}

	testMessage := dto.Message{
		Role:    "user",
		Content: "hi",
	}
	testRequest.Model = model
	testRequest.Messages = append(testRequest.Messages, testMessage)
	return testRequest
}

func TestChannel(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	testModel := c.Query("model")
	tik := time.Now()
	_, newAPIError := testChannel(channel, testModel)
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	go channel.UpdateResponseTime(milliseconds)
	consumedTime := float64(milliseconds) / 1000.0
	if newAPIError != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": newAPIError.Error(),
			"time":    consumedTime,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"time":    consumedTime,
	})
	return
}

var testAllChannelsLock sync.Mutex
var testAllChannelsRunning bool = false

func testAllChannels(notify bool) error {

	testAllChannelsLock.Lock()
	if testAllChannelsRunning {
		testAllChannelsLock.Unlock()
		return errors.New("测试已在运行中")
	}
	testAllChannelsRunning = true
	testAllChannelsLock.Unlock()
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		return err
	}
	var disableThreshold = int64(common.ChannelDisableThreshold * 1000)
	if disableThreshold == 0 {
		disableThreshold = 10000000 // a impossible value
	}
	gopool.Go(func() {
		// 使用 defer 确保无论如何都会重置运行状态，防止死锁
		defer func() {
			testAllChannelsLock.Lock()
			testAllChannelsRunning = false
			testAllChannelsLock.Unlock()
		}()

		for _, channel := range channels {
			isChannelEnabled := channel.Status == common.ChannelStatusEnabled
			tik := time.Now()
			err, newAPIError := testChannel(channel, "")
			tok := time.Now()
			milliseconds := tok.Sub(tik).Milliseconds()

			shouldBanChannel := false

			// request error disables the channel
			if err != nil {
				shouldBanChannel = service.ShouldDisableChannel(channel.Type, newAPIError)
			}

			if milliseconds > disableThreshold {
				err = errors.New(fmt.Sprintf("响应时间 %.2fs 超过阈值 %.2fs", float64(milliseconds)/1000.0, float64(disableThreshold)/1000.0))
				shouldBanChannel = true
			}

			// disable channel
			if isChannelEnabled && shouldBanChannel && channel.GetAutoBan() {
				service.DisableChannel(channel.Id, channel.Name, err.Error())
			}

			// enable channel
			if !isChannelEnabled && service.ShouldEnableChannel(err, newAPIError, channel.Status) {
				service.EnableChannel(channel.Id, channel.Name)
			}

			channel.UpdateResponseTime(milliseconds)
			time.Sleep(common.RequestInterval)
		}

		if notify {
			service.NotifyRootUser(dto.NotifyTypeChannelTest, "通道测试完成", "所有通道测试已完成")
		}
	})
	return nil
}

func TestAllChannels(c *gin.Context) {
	err := testAllChannels(true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func AutomaticallyTestChannels(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Minute)
		common.SysLog("testing all channels")
		_ = testAllChannels(false)
		common.SysLog("channel test finished")
	}
}
