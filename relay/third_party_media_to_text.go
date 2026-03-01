package relay

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type thirdPartyMultimodalConfig struct {
	modelID         string
	callAPIType     int
	systemPrompt    string
	firstUserPrompt string
}

type thirdPartyMediaToTextInput struct {
	ctx        *gin.Context
	info       *relaycommon.RelayInfo
	req        *dto.GeneralOpenAIRequest
	resolveURL relaycommon.MediaURLResolver
}

func applyThirdPartyModelMediaToText(in thirdPartyMediaToTextInput) *types.NewAPIError {
	cfg, cfgErr := loadThirdPartyMultimodalConfig()
	if cfgErr != nil {
		return cfgErr
	}

	usingGroup, groupErr := resolveThirdPartyUsingGroup(in.ctx, in.info)
	if groupErr != nil {
		return groupErr
	}

	selected, err := model.GetRandomSatisfiedChannelByAPIType(model.RandomSatisfiedChannelByAPITypeParams{
		Group:     usingGroup,
		ModelName: cfg.modelID,
		APIType:   cfg.callAPIType,
		Retry:     0,
	})
	if err != nil {
		return types.NewError(err, types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}
	if selected == nil {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("no available channel for third-party multimodal model %q in group %q", cfg.modelID, usingGroup),
			types.ErrorCodeGetChannelFailed,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	userSetting := dto.UserSetting{}
	if in.info != nil {
		userSetting = in.info.UserSetting
	}

	client, buildErr := newThirdPartyMediaTextClient(in.ctx, selected, cfg, userSetting)
	if buildErr != nil {
		return buildErr
	}

	_, convErr := relaycommon.ApplyMediaAutoConvertToText(in.req, in.resolveURL, client.Describe)
	if convErr == nil {
		return nil
	}
	if apiErr, ok := convErr.(*types.NewAPIError); ok {
		return apiErr
	}
	return types.NewError(convErr, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
}

func resolveThirdPartyUsingGroup(c *gin.Context, info *relaycommon.RelayInfo) (string, *types.NewAPIError) {
	if info != nil {
		if usingGroup := strings.TrimSpace(info.UsingGroup); usingGroup != "" {
			return usingGroup, nil
		}
	}
	if c != nil {
		if usingGroup := strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyUsingGroup)); usingGroup != "" {
			return usingGroup, nil
		}
	}
	return "", types.NewErrorWithStatusCode(errors.New("missing using group in context"), types.ErrorCodeGetChannelFailed, http.StatusInternalServerError, types.ErrOptionWithSkipRetry())
}

func loadThirdPartyMultimodalConfig() (thirdPartyMultimodalConfig, *types.NewAPIError) {
	gs := model_setting.GetGlobalSettings()
	cfg := thirdPartyMultimodalConfig{
		modelID:         strings.TrimSpace(gs.ThirdPartyMultimodalModelID),
		callAPIType:     gs.ThirdPartyMultimodalCallAPIType,
		systemPrompt:    gs.ThirdPartyMultimodalSystemPrompt,
		firstUserPrompt: gs.ThirdPartyMultimodalFirstUserPrompt,
	}
	if cfg.modelID == "" {
		return thirdPartyMultimodalConfig{}, types.NewErrorWithStatusCode(errors.New("third_party_multimodal_model_id is required for third_party_model mode"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if strings.TrimSpace(cfg.firstUserPrompt) == "" {
		return thirdPartyMultimodalConfig{}, types.NewErrorWithStatusCode(errors.New("third_party_multimodal_first_user_prompt is required for third_party_model mode"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	switch cfg.callAPIType {
	case constant.APITypeOpenAI, constant.APITypeAnthropic, constant.APITypeGemini:
		// ok
	default:
		return thirdPartyMultimodalConfig{}, types.NewErrorWithStatusCode(fmt.Errorf("unsupported third_party_multimodal_call_api_type: %d", cfg.callAPIType), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	return cfg, nil
}
