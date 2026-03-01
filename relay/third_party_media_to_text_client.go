package relay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type thirdPartyMediaTextClient struct {
	channel *model.Channel
	cfg     thirdPartyMultimodalConfig
	ctx     *gin.Context
	setting dto.UserSetting
}

func newThirdPartyMediaTextClient(parent *gin.Context, ch *model.Channel, cfg thirdPartyMultimodalConfig, userSetting dto.UserSetting) (*thirdPartyMediaTextClient, *types.NewAPIError) {
	if ch == nil {
		return nil, types.NewErrorWithStatusCode(errors.New("third-party channel is nil"), types.ErrorCodeGetChannelFailed, http.StatusInternalServerError, types.ErrOptionWithSkipRetry())
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request, _ = http.NewRequest(http.MethodPost, "/internal/third_party_multimodal", nil)
	if parent != nil && parent.Request != nil {
		ctx.Request.RemoteAddr = parent.Request.RemoteAddr
	}
	return &thirdPartyMediaTextClient{
		channel: ch,
		cfg:     cfg,
		ctx:     ctx,
		setting: userSetting,
	}, nil
}

func (c *thirdPartyMediaTextClient) Describe(kind string, resolvedURL string) (string, error) {
	normalizedKind, normalizedURL, err := normalizeThirdPartyMediaInput(kind, resolvedURL, c.cfg.callAPIType)
	if err != nil {
		return "", err
	}

	apiType, respBody, err := c.callThirdPartyMultimodalModel(normalizedKind, normalizedURL)
	if err != nil {
		return "", err
	}

	return parseThirdPartyOutputText(apiType, respBody)
}

func (c *thirdPartyMediaTextClient) callThirdPartyMultimodalModel(kind string, resolvedURL string) (int, []byte, error) {
	apiKey, _, keyErr := c.channel.GetNextEnabledKey()
	if keyErr != nil {
		return 0, nil, keyErr
	}

	internalReq := buildThirdPartyMultimodalRequest(thirdPartyMultimodalRequestArgs{
		modelID:         c.cfg.modelID,
		systemPrompt:    c.cfg.systemPrompt,
		firstUserPrompt: c.cfg.firstUserPrompt,
		kind:            kind,
		resolvedURL:     resolvedURL,
	})
	internalInfo, infoErr := buildThirdPartyRelayInfo(c.channel, apiKey, c.cfg.modelID)
	if infoErr != nil {
		return 0, nil, infoErr
	}
	internalInfo.UserSetting = c.setting

	adaptor := GetAdaptor(internalInfo.ApiType)
	if adaptor == nil {
		return 0, nil, types.NewErrorWithStatusCode(fmt.Errorf("invalid api type: %d", internalInfo.ApiType), types.ErrorCodeInvalidApiType, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(internalInfo)

	body, err := c.convertAndMarshal(adaptor, internalInfo, internalReq)
	if err != nil {
		return 0, nil, err
	}
	respBody, err := c.doRequestAndRead(adaptor, internalInfo, body)
	if err != nil {
		return 0, nil, err
	}

	return internalInfo.ApiType, respBody, nil
}

func (c *thirdPartyMediaTextClient) convertAndMarshal(adaptor relaychannel.Adaptor, info *relaycommon.RelayInfo, req *dto.GeneralOpenAIRequest) ([]byte, error) {
	c.ctx.Set("model_mapping", c.channel.GetModelMapping())
	if err := helper.ModelMappedHelper(c.ctx, info, req); err != nil {
		return nil, types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	payload, err := adaptor.ConvertOpenAIRequest(c.ctx, info, req)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeJsonMarshalFailed, types.ErrOptionWithSkipRetry())
	}
	return body, nil
}

func (c *thirdPartyMediaTextClient) doRequestAndRead(adaptor relaychannel.Adaptor, info *relaycommon.RelayInfo, body []byte) ([]byte, error) {
	respAny, err := adaptor.DoRequest(c.ctx, info, bytes.NewReader(body))
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeDoRequestFailed, types.ErrOptionWithSkipRetry())
	}

	resp, ok := respAny.(*http.Response)
	if !ok || resp == nil {
		return nil, types.NewErrorWithStatusCode(fmt.Errorf("unexpected third-party response type: %T", respAny), types.ErrorCodeBadResponse, http.StatusBadGateway, types.ErrOptionWithSkipRetry())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeReadResponseBodyFailed, types.ErrOptionWithSkipRetry())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, types.NewErrorWithStatusCode(fmt.Errorf("third-party upstream bad status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody))), types.ErrorCodeBadResponseStatusCode, http.StatusBadGateway, types.ErrOptionWithSkipRetry())
	}
	return respBody, nil
}

func normalizeThirdPartyMediaInput(kind string, resolvedURL string, callAPIType int) (string, string, error) {
	trimmedKind := strings.TrimSpace(kind)
	trimmedURL := strings.TrimSpace(resolvedURL)
	if trimmedKind == "" || trimmedURL == "" {
		return "", "", errors.New("kind and resolvedURL are required")
	}
	if trimmedKind != "image" && trimmedKind != "video" {
		return "", "", types.NewErrorWithStatusCode(fmt.Errorf("unsupported media kind: %q", trimmedKind), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if trimmedKind == "video" && callAPIType != constant.APITypeOpenAI {
		return "", "", types.NewErrorWithStatusCode(fmt.Errorf("video_url is only supported when third_party_multimodal_call_api_type=%d (OpenAI-compatible)", constant.APITypeOpenAI), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	return trimmedKind, trimmedURL, nil
}

type thirdPartyMultimodalRequestArgs struct {
	modelID         string
	systemPrompt    string
	firstUserPrompt string
	kind            string
	resolvedURL     string
}

func buildThirdPartyMultimodalRequest(args thirdPartyMultimodalRequestArgs) *dto.GeneralOpenAIRequest {
	msgs := make([]dto.Message, 0, 2)
	if strings.TrimSpace(args.systemPrompt) != "" {
		msgs = append(msgs, dto.Message{
			Role:    "system",
			Content: args.systemPrompt,
		})
	}

	user := dto.Message{Role: "user"}
	parts := []dto.MediaContent{
		{Type: dto.ContentTypeText, Text: args.firstUserPrompt},
	}
	switch strings.TrimSpace(args.kind) {
	case "video":
		parts = append(parts, dto.MediaContent{
			Type: dto.ContentTypeVideoUrl,
			VideoUrl: &dto.MessageVideoUrl{
				Url: args.resolvedURL,
			},
		})
	default:
		parts = append(parts, dto.MediaContent{
			Type: dto.ContentTypeImageURL,
			ImageUrl: &dto.MessageImageUrl{
				Url:    args.resolvedURL,
				Detail: "high",
			},
		})
	}
	user.SetMediaContent(parts)
	msgs = append(msgs, user)

	return &dto.GeneralOpenAIRequest{
		Model:    args.modelID,
		Messages: msgs,
		Stream:   false,
	}
}

func buildThirdPartyRelayInfo(ch *model.Channel, apiKey string, modelID string) (*relaycommon.RelayInfo, *types.NewAPIError) {
	if ch == nil {
		return nil, types.NewErrorWithStatusCode(errors.New("channel is nil"), types.ErrorCodeGetChannelFailed, http.StatusInternalServerError, types.ErrOptionWithSkipRetry())
	}
	apiType, ok := common.ChannelType2APIType(ch.Type)
	if !ok {
		return nil, types.NewErrorWithStatusCode(fmt.Errorf("unsupported channel type: %d", ch.Type), types.ErrorCodeInvalidApiType, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	channelSetting := ch.GetSetting()
	channelSetting.PassThroughBodyEnabled = false
	channelSetting.PassThroughHeadersEnabled = false
	channelOtherSettings := ch.GetOtherSettings()

	organization := ""
	if ch.OpenAIOrganization != nil {
		organization = strings.TrimSpace(*ch.OpenAIOrganization)
	}

	meta := &relaycommon.ChannelMeta{
		ChannelType:          ch.Type,
		ChannelId:            ch.Id,
		ChannelIsMultiKey:    ch.ChannelInfo.IsMultiKey,
		ChannelMultiKeyIndex: 0,
		ChannelBaseUrl:       ch.GetBaseURL(),
		ApiType:              apiType,
		ApiVersion:           ch.Other,
		ApiKey:               apiKey,
		Organization:         organization,
		ChannelCreateTime:    ch.CreatedTime,
		ParamOverride:        ch.GetParamOverride(),
		HeadersOverride:      ch.GetHeaderOverride(),
		UpstreamModelName:    modelID,
		IsModelMapped:        false,
		SupportStreamOptions: false,
		ChannelSetting:       channelSetting,
		ChannelOtherSettings: channelOtherSettings,
	}

	info := &relaycommon.RelayInfo{
		RequestURLPath:    "/v1/chat/completions",
		RelayMode:         relayconstant.RelayModeChatCompletions,
		RelayFormat:       types.RelayFormatOpenAI,
		IsStream:          false,
		OriginModelName:   modelID,
		ChannelMeta:       meta,
		IsClaudeBetaQuery: channelOtherSettings.ClaudeBetaQuery,
	}
	return info, nil
}
