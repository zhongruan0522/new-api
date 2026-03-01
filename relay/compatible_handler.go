package relay

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/shopspring/decimal"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TextHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	textReq, ok := info.Request.(*dto.GeneralOpenAIRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.GeneralOpenAIRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(textReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to GeneralOpenAIRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	if request.WebSearchOptions != nil {
		c.Set("chat_completion_web_search_context_size", request.WebSearchOptions.SearchContextSize)
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	mediaMode, modeOK := info.ChannelOtherSettings.ParseImageAutoConvertToURLMode()
	if !modeOK {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid image_auto_convert_to_url_mode: %q", info.ChannelOtherSettings.ImageAutoConvertToURLMode), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	// Channel-level multimodal handling for text-only upstream models.
	if mediaMode != dto.ImageAutoConvertToURLModeOff {
		storedURLBySHA := make(map[string]string)
		imageMaxBytes := int64(constant.MaxImageUploadMB) * 1024 * 1024
		videoMaxBytes := int64(constant.MaxVideoUploadMB) * 1024 * 1024
		imagePoolMaxBytes := int64(constant.StoredImagePoolMB) * 1024 * 1024
		videoPoolMaxBytes := int64(constant.StoredVideoPoolMB) * 1024 * 1024

		resolveURL := func(rawURL string, mediaContentType string) (string, error) {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				return "", nil
			}
			if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
				return rawURL, nil
			}

			mimeType, b64, err := service.DecodeBase64FileData(rawURL)
			if err != nil {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("decode media data failed: %w", err), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			mimeType = strings.TrimSpace(mimeType)
			if mimeType == "" {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("invalid media mime type: %q", mimeType), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			lowerMime := strings.ToLower(mimeType)
			isImage := mediaContentType == dto.ContentTypeImageURL
			isVideo := mediaContentType == dto.ContentTypeVideoUrl
			if isImage && !strings.HasPrefix(lowerMime, "image/") {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("invalid image mime type: %q", mimeType), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			if isVideo && !strings.HasPrefix(lowerMime, "video/") {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("invalid video mime type: %q", mimeType), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			if !isImage && !isVideo {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("unsupported media content type: %q", mediaContentType), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}

			b64 = strings.TrimSpace(b64)
			data, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("decode media base64 failed: %w", err), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			if len(data) == 0 {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("media data is empty"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}

			if isImage && imageMaxBytes > 0 && int64(len(data)) > imageMaxBytes {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("image size %d exceeds limit %d bytes", len(data), imageMaxBytes), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			if isVideo && videoMaxBytes > 0 && int64(len(data)) > videoMaxBytes {
				return "", types.NewErrorWithStatusCode(fmt.Errorf("video size %d exceeds limit %d bytes", len(data), videoMaxBytes), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}

			sha := hex.EncodeToString(common.Sha256Raw(data))
			cacheKey := mediaContentType + ":" + sha
			if existing, ok := storedURLBySHA[cacheKey]; ok {
				return existing, nil
			}

			if isImage {
				// Cross-request dedupe: same user + same sha -> reuse existing asset URL.
				if existing, err := model.GetStoredImageByUserAndSha(c.Request.Context(), info.UserId, sha); err == nil && existing != nil && existing.Id != "" {
					u := buildStoredImageURL(c, existing.Id)
					storedURLBySHA[cacheKey] = u
					return u, nil
				} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return "", types.NewError(fmt.Errorf("query stored image failed: %w", err), types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
				}

				img := &model.StoredImage{
					UserId:    info.UserId,
					ChannelId: info.ChannelId,
					MimeType:  mimeType,
					SizeBytes: len(data),
					Sha256:    sha,
					Data:      model.LargeBlob(data),
				}
				if err := img.Insert(c.Request.Context()); err != nil {
					return "", types.NewError(fmt.Errorf("store image failed: %w", err), types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
				}
				if _, err := model.EnsureStoredImagesPoolLimit(c.Request.Context(), imagePoolMaxBytes, 100); err != nil {
					return "", types.NewError(fmt.Errorf("enforce stored image pool limit failed: %w", err), types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
				}

				u := buildStoredImageURL(c, img.Id)
				storedURLBySHA[cacheKey] = u
				return u, nil
			}

			if existing, err := model.GetStoredVideoByUserAndSha(c.Request.Context(), info.UserId, sha); err == nil && existing != nil && existing.Id != "" {
				u := buildStoredVideoURL(c, existing.Id)
				storedURLBySHA[cacheKey] = u
				return u, nil
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return "", types.NewError(fmt.Errorf("query stored video failed: %w", err), types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
			}

			v := &model.StoredVideo{
				UserId:    info.UserId,
				ChannelId: info.ChannelId,
				MimeType:  mimeType,
				SizeBytes: len(data),
				Sha256:    sha,
				Data:      model.LargeBlob(data),
			}
			if err := v.Insert(c.Request.Context()); err != nil {
				return "", types.NewError(fmt.Errorf("store video failed: %w", err), types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
			}
			if _, err := model.EnsureStoredVideosPoolLimit(c.Request.Context(), videoPoolMaxBytes, 50); err != nil {
				return "", types.NewError(fmt.Errorf("enforce stored video pool limit failed: %w", err), types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
			}

			u := buildStoredVideoURL(c, v.Id)
			storedURLBySHA[cacheKey] = u
			return u, nil
		}

		switch mediaMode {
		case dto.ImageAutoConvertToURLModeMCP:
			_, convErr := relaycommon.ApplyImageAutoConvertToURL(request, resolveURL)
			if convErr != nil {
				return types.NewError(convErr, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
			}
		case dto.ImageAutoConvertToURLModeThirdPartyModel:
			if err := applyThirdPartyModelMediaToText(thirdPartyMediaToTextInput{
				ctx:        c,
				info:       info,
				req:        request,
				resolveURL: resolveURL,
			}); err != nil {
				return err
			}
		default:
			return types.NewErrorWithStatusCode(fmt.Errorf("unsupported image_auto_convert_to_url_mode: %s", mediaMode), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
	}

	includeUsage := true
	// 判断用户是否需要返回使用情况
	if request.StreamOptions != nil {
		includeUsage = request.StreamOptions.IncludeUsage
	}

	// 如果不支持StreamOptions，将StreamOptions设置为nil
	if !info.SupportStreamOptions || !request.Stream {
		request.StreamOptions = nil
	} else {
		// 如果支持StreamOptions，且请求中没有设置StreamOptions，根据配置文件设置StreamOptions
		if constant.ForceStreamOption {
			request.StreamOptions = &dto.StreamOptions{
				IncludeUsage: true,
			}
		}
	}

	info.ShouldIncludeUsage = includeUsage

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	passThroughGlobal := model_setting.GetGlobalSettings().PassThroughRequestEnabled
	passThroughBody := info.ChannelSetting.PassThroughBodyEnabled
	// Media handling rewrites the structured request; pass-through body would bypass it.
	if mediaMode != dto.ImageAutoConvertToURLModeOff {
		passThroughGlobal = false
		passThroughBody = false
	}
	if info.RelayMode == relayconstant.RelayModeChatCompletions &&
		!passThroughGlobal &&
		!passThroughBody &&
		service.ShouldChatCompletionsUseResponsesGlobal(info.ChannelId, info.ChannelType, info.OriginModelName) {
		applySystemPromptIfNeeded(c, info, request)
		usage, newApiErr := chatCompletionsViaResponses(c, info, adaptor, request)
		if newApiErr != nil {
			return newApiErr
		}

		var containAudioTokens = usage.CompletionTokenDetails.AudioTokens > 0 || usage.PromptTokensDetails.AudioTokens > 0
		var containsAudioRatios = ratio_setting.ContainsAudioRatio(info.OriginModelName) || ratio_setting.ContainsAudioCompletionRatio(info.OriginModelName)

		if containAudioTokens && containsAudioRatios {
			service.PostAudioConsumeQuota(c, info, usage, "")
		} else {
			postConsumeQuota(c, info, usage)
		}
		return nil
	}

	var requestBody io.Reader

	if passThroughGlobal || passThroughBody {
		body, err := common.GetRequestBody(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		if common.DebugEnabled {
			println("requestBody: ", string(body))
		}
		requestBody = bytes.NewBuffer(body)
	} else {
		convertedRequest, err := adaptor.ConvertOpenAIRequest(c, info, request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

		if info.ChannelSetting.SystemPrompt != "" {
			// 如果有系统提示，则将其添加到请求中
			request, ok := convertedRequest.(*dto.GeneralOpenAIRequest)
			if ok {
				containSystemPrompt := false
				for _, message := range request.Messages {
					if message.Role == request.GetSystemRoleName() {
						containSystemPrompt = true
						break
					}
				}
				if !containSystemPrompt {
					// 如果没有系统提示，则添加系统提示
					systemMessage := dto.Message{
						Role:    request.GetSystemRoleName(),
						Content: info.ChannelSetting.SystemPrompt,
					}
					request.Messages = append([]dto.Message{systemMessage}, request.Messages...)
				} else if info.ChannelSetting.SystemPromptOverride {
					common.SetContextKey(c, constant.ContextKeySystemPromptOverride, true)
					// 如果有系统提示，且允许覆盖，则拼接到前面
					for i, message := range request.Messages {
						if message.Role == request.GetSystemRoleName() {
							if message.IsStringContent() {
								request.Messages[i].SetStringContent(info.ChannelSetting.SystemPrompt + "\n" + message.StringContent())
							} else {
								contents := message.ParseContent()
								contents = append([]dto.MediaContent{
									{
										Type: dto.ContentTypeText,
										Text: info.ChannelSetting.SystemPrompt,
									},
								}, contents...)
								request.Messages[i].Content = contents
							}
							break
						}
					}
				}
			}
		}

		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return types.NewError(err, types.ErrorCodeJsonMarshalFailed, types.ErrOptionWithSkipRetry())
		}

		// remove disabled fields for OpenAI API
		jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		// apply param override
		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverride(jsonData, info.ParamOverride, relaycommon.BuildParamOverrideContext(info))
			if err != nil {
				return types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
			}
		}

		logger.LogDebug(c, fmt.Sprintf("text request body: %s", string(jsonData)))

		requestBody = bytes.NewBuffer(jsonData)
	}

	var httpResp *http.Response
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	if resp != nil {
		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode != http.StatusOK {
			newApiErr := service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			// reset status code 重置状态码
			service.ResetStatusCode(newApiErr, statusCodeMappingStr)
			return newApiErr
		}
	}

	usage, newApiErr := adaptor.DoResponse(c, httpResp, info)
	if newApiErr != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newApiErr, statusCodeMappingStr)
		return newApiErr
	}

	var containAudioTokens = usage.(*dto.Usage).CompletionTokenDetails.AudioTokens > 0 || usage.(*dto.Usage).PromptTokensDetails.AudioTokens > 0
	var containsAudioRatios = ratio_setting.ContainsAudioRatio(info.OriginModelName) || ratio_setting.ContainsAudioCompletionRatio(info.OriginModelName)

	if containAudioTokens && containsAudioRatios {
		service.PostAudioConsumeQuota(c, info, usage.(*dto.Usage), "")
	} else {
		postConsumeQuota(c, info, usage.(*dto.Usage))
	}
	return nil
}

func postConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, extraContent ...string) {
	originUsage := usage
	if usage == nil {
		usage = &dto.Usage{
			PromptTokens:     relayInfo.GetEstimatePromptTokens(),
			CompletionTokens: 0,
			TotalTokens:      relayInfo.GetEstimatePromptTokens(),
		}
		extraContent = append(extraContent, "上游无计费信息")
	}

	if originUsage != nil {
		service.ObserveChannelAffinityUsageCacheFromContext(ctx, usage)
	}

	adminRejectReason := common.GetContextKeyString(ctx, constant.ContextKeyAdminRejectReason)

	useTimeSeconds := time.Now().Unix() - relayInfo.StartTime.Unix()
	promptTokens := usage.PromptTokens
	cacheTokens := usage.PromptTokensDetails.CachedTokens
	imageTokens := usage.PromptTokensDetails.ImageTokens
	audioTokens := usage.PromptTokensDetails.AudioTokens
	completionTokens := usage.CompletionTokens
	cachedCreationTokens := usage.PromptTokensDetails.CachedCreationTokens

	modelName := relayInfo.OriginModelName

	tokenName := ctx.GetString("token_name")
	completionRatio := relayInfo.PriceData.CompletionRatio
	cacheRatio := relayInfo.PriceData.CacheRatio
	imageRatio := relayInfo.PriceData.ImageRatio
	modelRatio := relayInfo.PriceData.ModelRatio
	groupRatio := relayInfo.PriceData.GroupRatioInfo.GroupRatio
	modelPrice := relayInfo.PriceData.ModelPrice
	cachedCreationRatio := relayInfo.PriceData.CacheCreationRatio

	// Convert values to decimal for precise calculation
	dPromptTokens := decimal.NewFromInt(int64(promptTokens))
	dCacheTokens := decimal.NewFromInt(int64(cacheTokens))
	dImageTokens := decimal.NewFromInt(int64(imageTokens))
	dAudioTokens := decimal.NewFromInt(int64(audioTokens))
	dCompletionTokens := decimal.NewFromInt(int64(completionTokens))
	dCachedCreationTokens := decimal.NewFromInt(int64(cachedCreationTokens))
	dCompletionRatio := decimal.NewFromFloat(completionRatio)
	dCacheRatio := decimal.NewFromFloat(cacheRatio)
	dImageRatio := decimal.NewFromFloat(imageRatio)
	dModelRatio := decimal.NewFromFloat(modelRatio)
	dGroupRatio := decimal.NewFromFloat(groupRatio)
	dModelPrice := decimal.NewFromFloat(modelPrice)
	dCachedCreationRatio := decimal.NewFromFloat(cachedCreationRatio)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)

	ratio := dModelRatio.Mul(dGroupRatio)

	// openai web search 工具计费
	var dWebSearchQuota decimal.Decimal
	var webSearchPrice float64
	// response api 格式工具计费
	if relayInfo.ResponsesUsageInfo != nil {
		if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool.CallCount > 0 {
			// 计算 web search 调用的配额 (配额 = 价格 * 调用次数 / 1000 * 分组倍率)
			webSearchPrice = operation_setting.GetWebSearchPricePerThousand(modelName, webSearchTool.SearchContextSize)
			dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
				Mul(decimal.NewFromInt(int64(webSearchTool.CallCount))).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
			extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 %d 次，上下文大小 %s，调用花费 %s",
				webSearchTool.CallCount, webSearchTool.SearchContextSize, dWebSearchQuota.String()))
		}
	} else if strings.HasSuffix(modelName, "search-preview") {
		// search-preview 模型不支持 response api
		searchContextSize := ctx.GetString("chat_completion_web_search_context_size")
		if searchContextSize == "" {
			searchContextSize = "medium"
		}
		webSearchPrice = operation_setting.GetWebSearchPricePerThousand(modelName, searchContextSize)
		dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
			Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
		extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 1 次，上下文大小 %s，调用花费 %s",
			searchContextSize, dWebSearchQuota.String()))
	}
	// claude web search tool 计费
	var dClaudeWebSearchQuota decimal.Decimal
	var claudeWebSearchPrice float64
	claudeWebSearchCallCount := ctx.GetInt("claude_web_search_requests")
	if claudeWebSearchCallCount > 0 {
		claudeWebSearchPrice = operation_setting.GetClaudeWebSearchPricePerThousand()
		dClaudeWebSearchQuota = decimal.NewFromFloat(claudeWebSearchPrice).
			Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit).Mul(decimal.NewFromInt(int64(claudeWebSearchCallCount)))
		extraContent = append(extraContent, fmt.Sprintf("Claude Web Search 调用 %d 次，调用花费 %s",
			claudeWebSearchCallCount, dClaudeWebSearchQuota.String()))
	}
	// file search tool 计费
	var dFileSearchQuota decimal.Decimal
	var fileSearchPrice float64
	if relayInfo.ResponsesUsageInfo != nil {
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolFileSearch]; exists && fileSearchTool.CallCount > 0 {
			fileSearchPrice = operation_setting.GetFileSearchPricePerThousand()
			dFileSearchQuota = decimal.NewFromFloat(fileSearchPrice).
				Mul(decimal.NewFromInt(int64(fileSearchTool.CallCount))).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
			extraContent = append(extraContent, fmt.Sprintf("File Search 调用 %d 次，调用花费 %s",
				fileSearchTool.CallCount, dFileSearchQuota.String()))
		}
	}
	var dImageGenerationCallQuota decimal.Decimal
	var imageGenerationCallPrice float64
	if ctx.GetBool("image_generation_call") {
		imageGenerationCallPrice = operation_setting.GetGPTImage1PriceOnceCall(ctx.GetString("image_generation_call_quality"), ctx.GetString("image_generation_call_size"))
		dImageGenerationCallQuota = decimal.NewFromFloat(imageGenerationCallPrice).Mul(dGroupRatio).Mul(dQuotaPerUnit)
		extraContent = append(extraContent, fmt.Sprintf("Image Generation Call 花费 %s", dImageGenerationCallQuota.String()))
	}

	var quotaCalculateDecimal decimal.Decimal

	var audioInputQuota decimal.Decimal
	var audioInputPrice float64
	isClaudeUsageSemantic := relayInfo.FinalRequestRelayFormat == types.RelayFormatClaude
	if !relayInfo.PriceData.UsePrice {
		baseTokens := dPromptTokens
		// 减去 cached tokens
		// Anthropic API 的 input_tokens 已经不包含缓存 tokens，不需要减去
		// OpenAI/OpenRouter 等 API 的 prompt_tokens 包含缓存 tokens，需要减去
		var cachedTokensWithRatio decimal.Decimal
		if !dCacheTokens.IsZero() {
			if !isClaudeUsageSemantic {
				baseTokens = baseTokens.Sub(dCacheTokens)
			}
			cachedTokensWithRatio = dCacheTokens.Mul(dCacheRatio)
		}
		var dCachedCreationTokensWithRatio decimal.Decimal
		if !dCachedCreationTokens.IsZero() {
			if !isClaudeUsageSemantic {
				baseTokens = baseTokens.Sub(dCachedCreationTokens)
			}
			dCachedCreationTokensWithRatio = dCachedCreationTokens.Mul(dCachedCreationRatio)
		}

		// 减去 image tokens
		var imageTokensWithRatio decimal.Decimal
		if !dImageTokens.IsZero() {
			baseTokens = baseTokens.Sub(dImageTokens)
			imageTokensWithRatio = dImageTokens.Mul(dImageRatio)
		}

		// 减去 Gemini audio tokens
		if !dAudioTokens.IsZero() {
			audioInputPrice = operation_setting.GetGeminiInputAudioPricePerMillionTokens(modelName)
			if audioInputPrice > 0 {
				// 重新计算 base tokens
				baseTokens = baseTokens.Sub(dAudioTokens)
				audioInputQuota = decimal.NewFromFloat(audioInputPrice).Div(decimal.NewFromInt(1000000)).Mul(dAudioTokens).Mul(dGroupRatio).Mul(dQuotaPerUnit)
				extraContent = append(extraContent, fmt.Sprintf("Audio Input 花费 %s", audioInputQuota.String()))
			}
		}
		promptQuota := baseTokens.Add(cachedTokensWithRatio).
			Add(imageTokensWithRatio).
			Add(dCachedCreationTokensWithRatio)

		completionQuota := dCompletionTokens.Mul(dCompletionRatio)

		quotaCalculateDecimal = promptQuota.Add(completionQuota).Mul(ratio)

		if !ratio.IsZero() && quotaCalculateDecimal.LessThanOrEqual(decimal.Zero) {
			quotaCalculateDecimal = decimal.NewFromInt(1)
		}
	} else {
		quotaCalculateDecimal = dModelPrice.Mul(dQuotaPerUnit).Mul(dGroupRatio)
	}
	// 添加 responses tools call 调用的配额
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dWebSearchQuota)
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dFileSearchQuota)
	// 添加 audio input 独立计费
	quotaCalculateDecimal = quotaCalculateDecimal.Add(audioInputQuota)
	// 添加 image generation call 计费
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dImageGenerationCallQuota)

	if len(relayInfo.PriceData.OtherRatios) > 0 {
		for key, otherRatio := range relayInfo.PriceData.OtherRatios {
			dOtherRatio := decimal.NewFromFloat(otherRatio)
			quotaCalculateDecimal = quotaCalculateDecimal.Mul(dOtherRatio)
			extraContent = append(extraContent, fmt.Sprintf("其他倍率 %s: %f", key, otherRatio))
		}
	}

	quota := int(quotaCalculateDecimal.Round(0).IntPart())
	totalTokens := promptTokens + completionTokens

	//var logContent string

	// record all the consume log even if quota is 0
	if totalTokens == 0 {
		// in this case, must be some error happened
		// we cannot just return, because we may have to return the pre-consumed quota
		quota = 0
		extraContent = append(extraContent, "上游没有返回计费信息，无法扣费（可能是上游超时）")
		logger.LogError(ctx, fmt.Sprintf("total tokens is 0, cannot consume quota, userId %d, channelId %d, "+
			"tokenId %d, model %s， pre-consumed quota %d", relayInfo.UserId, relayInfo.ChannelId, relayInfo.TokenId, modelName, relayInfo.FinalPreConsumedQuota))
	} else {
		if !ratio.IsZero() && quota == 0 {
			quota = 1
		}
		model.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, quota)
		model.UpdateChannelUsedQuota(relayInfo.ChannelId, quota)
	}

	if err := service.SettleBilling(ctx, relayInfo, quota); err != nil {
		logger.LogError(ctx, "error settling billing: "+err.Error())
	}

	logModel := modelName
	if strings.HasPrefix(logModel, "gpt-4-gizmo") {
		logModel = "gpt-4-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", modelName))
	}
	if strings.HasPrefix(logModel, "gpt-4o-gizmo") {
		logModel = "gpt-4o-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", modelName))
	}
	logContent := strings.Join(extraContent, ", ")
	other := service.GenerateTextOtherInfo(ctx, relayInfo, modelRatio, groupRatio, completionRatio, cacheTokens, cacheRatio, modelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
	if adminRejectReason != "" {
		other["reject_reason"] = adminRejectReason
	}
	// For chat-based calls to the Claude model, tagging is required. Using Claude's rendering logs, the two approaches handle input rendering differently.
	if isClaudeUsageSemantic {
		other["claude"] = true
		other["usage_semantic"] = "anthropic"
	}
	if imageTokens != 0 {
		other["image"] = true
		other["image_ratio"] = imageRatio
		other["image_output"] = imageTokens
	}
	if cachedCreationTokens != 0 {
		other["cache_creation_tokens"] = cachedCreationTokens
		other["cache_creation_ratio"] = cachedCreationRatio
	}
	if !dWebSearchQuota.IsZero() {
		if relayInfo.ResponsesUsageInfo != nil {
			if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists {
				other["web_search"] = true
				other["web_search_call_count"] = webSearchTool.CallCount
				other["web_search_price"] = webSearchPrice
			}
		} else if strings.HasSuffix(modelName, "search-preview") {
			other["web_search"] = true
			other["web_search_call_count"] = 1
			other["web_search_price"] = webSearchPrice
		}
	} else if !dClaudeWebSearchQuota.IsZero() {
		other["web_search"] = true
		other["web_search_call_count"] = claudeWebSearchCallCount
		other["web_search_price"] = claudeWebSearchPrice
	}
	if !dFileSearchQuota.IsZero() && relayInfo.ResponsesUsageInfo != nil {
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolFileSearch]; exists {
			other["file_search"] = true
			other["file_search_call_count"] = fileSearchTool.CallCount
			other["file_search_price"] = fileSearchPrice
		}
	}
	if !audioInputQuota.IsZero() {
		other["audio_input_seperate_price"] = true
		other["audio_input_token_count"] = audioTokens
		other["audio_input_price"] = audioInputPrice
	}
	if !dImageGenerationCallQuota.IsZero() {
		other["image_generation_call"] = true
		other["image_generation_call_price"] = imageGenerationCallPrice
	}
	model.RecordConsumeLog(ctx, relayInfo.UserId, model.RecordConsumeLogParams{
		ChannelId:        relayInfo.ChannelId,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		ModelName:        logModel,
		TokenName:        tokenName,
		Quota:            quota,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeSeconds:   int(useTimeSeconds),
		IsStream:         relayInfo.IsStream,
		Group:            relayInfo.UsingGroup,
		Other:            other,
	})
}
