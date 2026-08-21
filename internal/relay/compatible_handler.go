package relay

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/operation"
	"github.com/NookMux/NookMux/internal/config/ratio"
	billing "github.com/NookMux/NookMux/internal/domain/billing"
	domainchannel "github.com/NookMux/NookMux/internal/domain/channel"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/infra/cache"
	"github.com/NookMux/NookMux/internal/infra/log"
	media "github.com/NookMux/NookMux/internal/infra/media"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/internal/relay/helper"
	"github.com/NookMux/NookMux/internal/store/channel"
	"github.com/NookMux/NookMux/internal/store/log"
	"github.com/NookMux/NookMux/internal/store/stored_media"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strings"
	"time"
)

func TextHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *shared.NookMuxError) {
	info.InitChannelMeta(c)

	textReq, ok := info.Request.(*shared.GeneralOpenAIRequest)
	if !ok {
		return shared.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected shared.GeneralOpenAIRequest, got %T", info.Request), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(textReq)
	if err != nil {
		return shared.NewError(fmt.Errorf("failed to copy request to GeneralOpenAIRequest: %w", err), shared.ErrorCodeInvalidRequest, shared.ErrOptionWithSkipRetry())
	}

	if request.WebSearchOptions != nil {
		c.Set("chat_completion_web_search_context_size", request.WebSearchOptions.SearchContextSize)
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return shared.NewError(err, shared.ErrorCodeChannelModelMappedError, shared.ErrOptionWithSkipRetry())
	}

	mediaMode, modeOK := info.ChannelOtherSettings.ParseImageAutoConvertToURLMode()
	if !modeOK {
		return shared.NewErrorWithStatusCode(fmt.Errorf("invalid image_auto_convert_to_url_mode: %q", info.ChannelOtherSettings.ImageAutoConvertToURLMode), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
	}

	// Channel-level multimodal handling for text-only upstream models.
	if mediaMode != shared.ImageAutoConvertToURLModeOff {
		storedURLBySHA := make(map[string]string)
		imageMaxBytes := int64(shared.MaxImageUploadMB) * 1024 * 1024
		videoMaxBytes := int64(shared.MaxVideoUploadMB) * 1024 * 1024
		imagePoolMaxBytes := int64(shared.StoredImagePoolMB) * 1024 * 1024
		videoPoolMaxBytes := int64(shared.StoredVideoPoolMB) * 1024 * 1024
		// 跟踪本次请求中新增存储的图片/视频数量（去重命中不计入）
		newImageCount := 0
		newVideoCount := 0

		resolveURL := func(rawURL string, mediaContentType string) (string, error) {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				return "", nil
			}
			if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
				return rawURL, nil
			}

			mimeType, b64, err := media.DecodeBase64FileData(rawURL)
			if err != nil {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("decode media data failed: %w", err), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}
			mimeType = strings.TrimSpace(mimeType)
			if mimeType == "" {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("invalid media mime type: %q", mimeType), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}
			lowerMime := strings.ToLower(mimeType)
			isImage := mediaContentType == shared.ContentTypeImageURL
			isVideo := mediaContentType == shared.ContentTypeVideoUrl
			if isImage && !strings.HasPrefix(lowerMime, "image/") {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("invalid image mime type: %q", mimeType), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}
			if isVideo && !strings.HasPrefix(lowerMime, "video/") {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("invalid video mime type: %q", mimeType), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}
			if !isImage && !isVideo {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("unsupported media content type: %q", mediaContentType), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}

			b64 = strings.TrimSpace(b64)
			data, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("decode media base64 failed: %w", err), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}
			if len(data) == 0 {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("media data is empty"), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}

			if isImage && imageMaxBytes > 0 && int64(len(data)) > imageMaxBytes {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("image size %d exceeds limit %d bytes", len(data), imageMaxBytes), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}
			if isVideo && videoMaxBytes > 0 && int64(len(data)) > videoMaxBytes {
				return "", shared.NewErrorWithStatusCode(fmt.Errorf("video size %d exceeds limit %d bytes", len(data), videoMaxBytes), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
			}

			sha := hex.EncodeToString(common.Sha256Raw(data))
			cacheKey := mediaContentType + ":" + sha
			if existing, ok := storedURLBySHA[cacheKey]; ok {
				return existing, nil
			}

			if isImage {
				// Cross-request dedupe: same user + same sha -> reuse existing asset URL.
				if existing, err := storedmediastore.GetStoredImageByUserAndSha(c.Request.Context(), info.UserId, sha); err == nil && existing != nil && existing.Id != "" {
					u := buildStoredImageURL(c, existing.Id)
					storedURLBySHA[cacheKey] = u
					return u, nil
				} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return "", shared.NewError(fmt.Errorf("query stored image failed: %w", err), shared.ErrorCodeQueryDataError, shared.ErrOptionWithSkipRetry())
				}

				img := &storedmediastore.StoredImage{
					UserId:    info.UserId,
					ChannelId: info.ChannelId,
					MimeType:  mimeType,
					SizeBytes: len(data),
					Sha256:    sha,
					Data:      storedmediastore.LargeBlob(data),
				}
				if err := img.Insert(c.Request.Context()); err != nil {
					return "", shared.NewError(fmt.Errorf("store image failed: %w", err), shared.ErrorCodeUpdateDataError, shared.ErrOptionWithSkipRetry())
				}
				if _, err := storedmediastore.EnsureStoredImagesPoolLimit(c.Request.Context(), imagePoolMaxBytes, 100); err != nil {
					return "", shared.NewError(fmt.Errorf("enforce stored image pool limit failed: %w", err), shared.ErrorCodeUpdateDataError, shared.ErrOptionWithSkipRetry())
				}

				newImageCount++
				u := buildStoredImageURL(c, img.Id)
				storedURLBySHA[cacheKey] = u
				return u, nil
			}

			if existing, err := storedmediastore.GetStoredVideoByUserAndSha(c.Request.Context(), info.UserId, sha); err == nil && existing != nil && existing.Id != "" {
				u := buildStoredVideoURL(c, existing.Id)
				storedURLBySHA[cacheKey] = u
				return u, nil
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return "", shared.NewError(fmt.Errorf("query stored video failed: %w", err), shared.ErrorCodeQueryDataError, shared.ErrOptionWithSkipRetry())
			}

			v := &storedmediastore.StoredVideo{
				UserId:    info.UserId,
				ChannelId: info.ChannelId,
				MimeType:  mimeType,
				SizeBytes: len(data),
				Sha256:    sha,
				Data:      storedmediastore.LargeBlob(data),
			}
			if err := v.Insert(c.Request.Context()); err != nil {
				return "", shared.NewError(fmt.Errorf("store video failed: %w", err), shared.ErrorCodeUpdateDataError, shared.ErrOptionWithSkipRetry())
			}
			if _, err := storedmediastore.EnsureStoredVideosPoolLimit(c.Request.Context(), videoPoolMaxBytes, 50); err != nil {
				return "", shared.NewError(fmt.Errorf("enforce stored video pool limit failed: %w", err), shared.ErrorCodeUpdateDataError, shared.ErrOptionWithSkipRetry())
			}

			newVideoCount++
			u := buildStoredVideoURL(c, v.Id)
			storedURLBySHA[cacheKey] = u
			return u, nil
		}

		if mediaMode != shared.ImageAutoConvertToURLModeMCP {
			return shared.NewErrorWithStatusCode(fmt.Errorf("unsupported image_auto_convert_to_url_mode: %s", mediaMode), shared.ErrorCodeInvalidRequest, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
		}

		_, convErr := relaycommon.ApplyImageAutoConvertToURL(request, resolveURL)
		if convErr != nil {
			return shared.NewError(convErr, shared.ErrorCodeInvalidRequest, shared.ErrOptionWithSkipRetry())
		}

		// 异步更新用户的多模态适配转换计数
		if newImageCount > 0 || newVideoCount > 0 {
			go userstore.IncrementMediaConvertedCount(info.UserId, newImageCount, newVideoCount)
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
		if shared.ForceStreamOption {
			request.StreamOptions = &shared.StreamOptions{
				IncludeUsage: true,
			}
		}
	}

	info.ShouldIncludeUsage = includeUsage

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return shared.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), shared.ErrorCodeInvalidApiType, shared.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	passThroughBody := info.ChannelSetting.PassThroughBodyEnabled
	// Media handling rewrites the structured request; pass-through body would bypass it.
	if mediaMode != shared.ImageAutoConvertToURLModeOff {
		passThroughBody = false
	}

	var requestBody io.Reader

	if passThroughBody {
		storage, err := httpapi.GetBodyStorage(c)
		if err != nil {
			return shared.NewErrorWithStatusCode(err, shared.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, shared.ErrOptionWithSkipRetry())
		}
		if common.DebugEnabled {
			body, _ := storage.Bytes()
			println("requestBody: ", string(body))
			_, _ = storage.Seek(0, io.SeekStart)
		}
		info.UpstreamRequestBodySize = storage.Size()
		requestBody = cache.ReaderOnly(storage)
	} else {
		convertedRequest, err := adaptor.ConvertOpenAIRequest(c, info, request)
		if err != nil {
			return shared.NewError(err, shared.ErrorCodeConvertRequestFailed, shared.ErrOptionWithSkipRetry())
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

		jsonData, err := jsonx.Marshal(convertedRequest)
		if err != nil {
			return shared.NewError(err, shared.ErrorCodeJsonMarshalFailed, shared.ErrOptionWithSkipRetry())
		}

		// remove disabled fields for OpenAI API
		jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings)
		if err != nil {
			return shared.NewError(err, shared.ErrorCodeConvertRequestFailed, shared.ErrOptionWithSkipRetry())
		}

		// apply OpenRouter provider routing preferences (channel overrides client)
		jsonData, err = relaycommon.ApplyProviderRouting(jsonData, info)
		if err != nil {
			return shared.NewError(err, shared.ErrorCodeConvertRequestFailed, shared.ErrOptionWithSkipRetry())
		}

		// apply param override
		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			if err != nil {
				return shared.NewError(err, shared.ErrorCodeChannelParamOverrideInvalid, shared.ErrOptionWithSkipRetry())
			}
		}

		log.LogDebug(c, fmt.Sprintf("text request body: %s", string(jsonData)))

		body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
		if err != nil {
			return shared.NewError(err, shared.ErrorCodeConvertRequestFailed, shared.ErrOptionWithSkipRetry())
		}
		defer closer.Close()
		jsonData = nil
		info.UpstreamRequestBodySize = size
		requestBody = body
	}

	// 通用思维强度解析：adaptor 未设置时从原始请求兜底提取
	relaycommon.EnsureReasoningEffort(info, info.Request)

	var httpResp *http.Response
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return shared.NewOpenAIError(err, shared.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	if resp != nil {
		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode != http.StatusOK {
			newApiErr := helper.RelayErrorHandler(c.Request.Context(), httpResp, false)
			// reset status code 重置状态码
			helper.ResetStatusCode(newApiErr, statusCodeMappingStr)
			return newApiErr
		}
	}

	usage, newApiErr := adaptor.DoResponse(c, httpResp, info)
	if newApiErr != nil {
		// reset status code 重置状态码
		helper.ResetStatusCode(newApiErr, statusCodeMappingStr)
		return newApiErr
	}

	var containAudioTokens = usage.(*shared.Usage).CompletionTokenDetails.AudioTokens > 0 || usage.(*shared.Usage).PromptTokensDetails.AudioTokens > 0
	var containsAudioRatios = ratio.ContainsAudioRatio(info.OriginModelName) || ratio.ContainsAudioCompletionRatio(info.OriginModelName)

	if containAudioTokens && containsAudioRatios {
		if apiErr := billing.PostAudioConsumeQuota(c, info, usage.(*shared.Usage), ""); apiErr != nil {
			return apiErr
		}
	} else {
		if apiErr := postConsumeQuota(c, info, usage.(*shared.Usage)); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func postConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *shared.Usage, extraContent ...string) *shared.NookMuxError {
	originUsage := usage
	if usage == nil {
		usage = &shared.Usage{
			PromptTokens:     relayInfo.GetEstimatePromptTokens(),
			CompletionTokens: 0,
			TotalTokens:      relayInfo.GetEstimatePromptTokens(),
		}
		extraContent = append(extraContent, "上游无计费信息")
	}

	if originUsage != nil {
		domainchannel.ObserveChannelAffinityUsageCacheFromContext(ctx, usage)
	}

	adminRejectReason := httpapi.GetContextKeyString(ctx, common.ContextKeyAdminRejectReason)

	useTimeMs := time.Since(relayInfo.StartTime).Milliseconds()
	promptTokens := usage.PromptTokens
	cacheTokens := usage.PromptTokensDetails.CachedTokens
	audioTokens := usage.PromptTokensDetails.AudioTokens
	completionTokens := usage.CompletionTokens
	cachedCreationTokens := usage.PromptTokensDetails.CachedCreationTokens

	modelName := relayInfo.OriginModelName

	tokenName := ctx.GetString("token_name")
	completionRatio := relayInfo.PriceData.CompletionRatio
	cacheRatio := relayInfo.PriceData.CacheRatio
	modelRatio := relayInfo.PriceData.ModelRatio
	groupRatio := relayInfo.PriceData.GroupRatioInfo.GroupRatio
	modelPrice := relayInfo.PriceData.ModelPrice
	cachedCreationRatio := relayInfo.PriceData.CacheCreationRatio
	isClaudeUsageSemantic := relayInfo.FinalRequestRelayFormat == relayconstant.RelayFormatClaude

	if _, enabled, err := billing.ApplyContextPricingForUsage(modelName, billing.BuildContextPricingUsage(usage, isClaudeUsageSemantic), &relayInfo.PriceData); enabled {
		if err != nil {
			log.LogError(ctx, "context pricing failed: "+err.Error())
			extraContent = append(extraContent, "分段计费匹配失败: "+err.Error())
		} else {
			completionRatio = relayInfo.PriceData.CompletionRatio
			cacheRatio = relayInfo.PriceData.CacheRatio
			modelRatio = relayInfo.PriceData.ModelRatio
			modelPrice = relayInfo.PriceData.ModelPrice
			cachedCreationRatio = relayInfo.PriceData.CacheCreationRatio
		}
	}

	// Convert values to decimal for precise calculation
	dPromptTokens := decimal.NewFromInt(int64(promptTokens))
	dCacheTokens := decimal.NewFromInt(int64(cacheTokens))
	dAudioTokens := decimal.NewFromInt(int64(audioTokens))
	dCompletionTokens := decimal.NewFromInt(int64(completionTokens))
	dCachedCreationTokens := decimal.NewFromInt(int64(cachedCreationTokens))
	dCompletionRatio := decimal.NewFromFloat(completionRatio)
	dCacheRatio := decimal.NewFromFloat(cacheRatio)
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
		if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[shared.BuildInToolWebSearchPreview]; exists && webSearchTool.CallCount > 0 {
			// 优先使用可配置价格，回退到硬编码常量
			if pricePerCall, ok := operation.GetToolBillingPrice("web_search", map[string]string{"model": modelName, "provider": "openai"}); ok {
				webSearchPrice = pricePerCall
				dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
					Mul(decimal.NewFromInt(int64(webSearchTool.CallCount))).
					Mul(dGroupRatio).Mul(dQuotaPerUnit)
			} else {
				webSearchPrice = operation.GetWebSearchPricePerThousand(modelName, webSearchTool.SearchContextSize)
				dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
					Mul(decimal.NewFromInt(int64(webSearchTool.CallCount))).
					Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
			}
			extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 %d 次，上下文大小 %s，调用花费 %s",
				webSearchTool.CallCount, webSearchTool.SearchContextSize, dWebSearchQuota.String()))
		}
	} else if strings.HasSuffix(modelName, "search-preview") {
		// search-preview 模型不支持 response api
		searchContextSize := ctx.GetString("chat_completion_web_search_context_size")
		if searchContextSize == "" {
			searchContextSize = "medium"
		}
		if pricePerCall, ok := operation.GetToolBillingPrice("web_search", map[string]string{"model": modelName, "provider": "openai"}); ok {
			webSearchPrice = pricePerCall
			dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
				Mul(dGroupRatio).Mul(dQuotaPerUnit)
		} else {
			webSearchPrice = operation.GetWebSearchPricePerThousand(modelName, searchContextSize)
			dWebSearchQuota = decimal.NewFromFloat(webSearchPrice).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
		}
		extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 1 次，上下文大小 %s，调用花费 %s",
			searchContextSize, dWebSearchQuota.String()))
	}
	// claude web search tool 计费
	var dClaudeWebSearchQuota decimal.Decimal
	var claudeWebSearchPrice float64
	claudeWebSearchCallCount := ctx.GetInt("claude_web_search_requests")
	if claudeWebSearchCallCount > 0 {
		if pricePerCall, ok := operation.GetToolBillingPrice("web_search", map[string]string{"model": modelName, "provider": "claude"}); ok {
			claudeWebSearchPrice = pricePerCall
			dClaudeWebSearchQuota = decimal.NewFromFloat(claudeWebSearchPrice).
				Mul(decimal.NewFromInt(int64(claudeWebSearchCallCount))).
				Mul(dGroupRatio).Mul(dQuotaPerUnit)
		} else {
			claudeWebSearchPrice = operation.GetClaudeWebSearchPricePerThousand()
			dClaudeWebSearchQuota = decimal.NewFromFloat(claudeWebSearchPrice).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit).Mul(decimal.NewFromInt(int64(claudeWebSearchCallCount)))
		}
		extraContent = append(extraContent, fmt.Sprintf("Claude Web Search 调用 %d 次，调用花费 %s",
			claudeWebSearchCallCount, dClaudeWebSearchQuota.String()))
	}
	// gemini web search tool 计费
	var dGeminiWebSearchQuota decimal.Decimal
	var geminiWebSearchPrice float64
	geminiWebSearchCallCount := ctx.GetInt("gemini_web_search_requests")
	if geminiWebSearchCallCount > 0 {
		if pricePerCall, ok := operation.GetToolBillingPrice("web_search", map[string]string{"model": modelName, "provider": "gemini"}); ok {
			geminiWebSearchPrice = pricePerCall
			dGeminiWebSearchQuota = decimal.NewFromFloat(geminiWebSearchPrice).
				Mul(decimal.NewFromInt(int64(geminiWebSearchCallCount))).
				Mul(dGroupRatio).Mul(dQuotaPerUnit)
		}
		if !dGeminiWebSearchQuota.IsZero() {
			extraContent = append(extraContent, fmt.Sprintf("Gemini Web Search 调用 %d 次，调用花费 %s",
				geminiWebSearchCallCount, dGeminiWebSearchQuota.String()))
		}
	}
	// file search tool 计费
	var dFileSearchQuota decimal.Decimal
	var fileSearchPrice float64
	if relayInfo.ResponsesUsageInfo != nil {
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[shared.BuildInToolFileSearch]; exists && fileSearchTool.CallCount > 0 {
			fileSearchPrice = operation.GetFileSearchPricePerThousand()
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
		if pricePerCall, ok := operation.GetToolBillingPrice("image_generation", map[string]string{
			"model":    modelName,
			"provider": "openai",
			"quality":  ctx.GetString("image_generation_call_quality"),
			"size":     ctx.GetString("image_generation_call_size"),
		}); ok {
			imageGenerationCallPrice = pricePerCall
		} else {
			imageGenerationCallPrice = operation.GetGPTImage1PriceOnceCall(ctx.GetString("image_generation_call_quality"), ctx.GetString("image_generation_call_size"))
		}
		dImageGenerationCallQuota = decimal.NewFromFloat(imageGenerationCallPrice).Mul(dGroupRatio).Mul(dQuotaPerUnit)
		extraContent = append(extraContent, fmt.Sprintf("Image Generation Call 花费 %s", dImageGenerationCallQuota.String()))
	}

	var quotaCalculateDecimal decimal.Decimal

	var audioInputQuota decimal.Decimal
	var audioInputPrice float64
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

		// 减去 Gemini audio tokens
		if !dAudioTokens.IsZero() {
			audioInputPrice = operation.GetGeminiInputAudioPricePerMillionTokens(modelName)
			if audioInputPrice > 0 {
				// 重新计算 base tokens
				baseTokens = baseTokens.Sub(dAudioTokens)
				audioInputQuota = decimal.NewFromFloat(audioInputPrice).Div(decimal.NewFromInt(1000000)).Mul(dAudioTokens).Mul(dGroupRatio).Mul(dQuotaPerUnit)
				extraContent = append(extraContent, fmt.Sprintf("Audio Input 花费 %s", audioInputQuota.String()))
			}
		}
		promptQuota := baseTokens.Add(cachedTokensWithRatio).
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
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dClaudeWebSearchQuota)
	quotaCalculateDecimal = quotaCalculateDecimal.Add(dGeminiWebSearchQuota)
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
	// 未发生规范转换时，totalTokens == 0 代表上游缺失计费信息，需要交给外层重试。
	// 发生规范转换时，可能是转换导致的 token 统计异常，继续记录消费日志但不触发重试。
	logType := 0 // 0 表示使用默认的 LogTypeConsume
	if totalTokens == 0 {
		if apiErr := billing.NewEmptyUsageRetryError(ctx, relayInfo); apiErr != nil {
			return apiErr
		}
		// 上游没有返回 token 信息（可能是超时或错误），但如果有工具调用费用，仍需扣费
		toolQuota := dWebSearchQuota.Add(dClaudeWebSearchQuota).Add(dGeminiWebSearchQuota).
			Add(dFileSearchQuota).Add(dImageGenerationCallQuota).Add(audioInputQuota)
		if toolQuota.GreaterThan(decimal.Zero) {
			quota = int(toolQuota.Round(0).IntPart())
			extraContent = append(extraContent, "上游没有返回计费信息，但工具调用费用仍需扣除")
			userstore.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, quota)
			channelstore.UpdateChannelUsedQuota(relayInfo.ChannelId, quota)
		} else {
			quota = 0
			extraContent = append(extraContent, "上游没有返回计费信息，无法扣费（可能是上游超时）")
		}
		log.LogError(ctx, fmt.Sprintf("total tokens is 0, cannot consume quota, userId %d, channelId %d, "+
			"tokenId %d, model %s， pre-consumed quota %d", relayInfo.UserId, relayInfo.ChannelId, relayInfo.TokenId, modelName, relayInfo.FinalPreConsumedQuota))
	} else {
		if !ratio.IsZero() && quota == 0 {
			quota = 1
		}
		userstore.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, quota)
		channelstore.UpdateChannelUsedQuota(relayInfo.ChannelId, quota)
	}

	if err := billing.SettleBilling(ctx, relayInfo, quota); err != nil {
		log.LogError(ctx, "error settling billing: "+err.Error())
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
	// 计算原始分组倍率（不含动态倍率），用于日志记录
	originalGroupRatio := groupRatio
	dynamicRatio := relayInfo.PriceData.GroupRatioInfo.DynamicRatio
	if dynamicRatio > 0 {
		originalGroupRatio = groupRatio / dynamicRatio
	}
	other := billing.GenerateTextOtherInfo(ctx, relayInfo, modelRatio, originalGroupRatio, completionRatio, cacheTokens, cacheRatio, modelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio, dynamicRatio)
	if adminRejectReason != "" {
		other["reject_reason"] = adminRejectReason
	}
	// For chat-based calls to the Claude model, tagging is required. Using Claude's rendering logs, the two approaches handle input rendering differently.
	if isClaudeUsageSemantic {
		other["claude"] = true
		other["usage_semantic"] = "anthropic"
	}
	if cachedCreationTokens != 0 {
		other["cache_creation_tokens"] = cachedCreationTokens
		other["cache_creation_ratio"] = cachedCreationRatio
	}
	if !dWebSearchQuota.IsZero() {
		if relayInfo.ResponsesUsageInfo != nil {
			if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[shared.BuildInToolWebSearchPreview]; exists {
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
	} else if !dGeminiWebSearchQuota.IsZero() {
		other["web_search"] = true
		other["web_search_call_count"] = geminiWebSearchCallCount
		other["web_search_price"] = geminiWebSearchPrice
	}
	if !dFileSearchQuota.IsZero() && relayInfo.ResponsesUsageInfo != nil {
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[shared.BuildInToolFileSearch]; exists {
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
	// 共享流式日志指标，确保 OpenAI 兼容与 Claude 消费日志展示一致。
	billing.AppendStreamMetrics(other, relayInfo, useTimeMs, completionTokens)
	logstore.RecordConsumeLog(ctx, relayInfo.UserId, logstore.RecordConsumeLogParams{
		ChannelId:        relayInfo.ChannelId,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		ModelName:        logModel,
		TokenName:        tokenName,
		Quota:            quota,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeMs:        int(useTimeMs),
		IsStream:         relayInfo.IsStream,
		Group:            relayInfo.UsingGroup,
		Other:            other,
		LogType:          logType,
	})
	return nil
}
