package controller

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/i18n"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/setting"
	"github.com/zhongruan0522/new-api/setting/console_setting"
	"github.com/zhongruan0522/new-api/setting/model_setting"
	"github.com/zhongruan0522/new-api/setting/operation_setting"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetOptions(c *gin.Context) {
	var options []*model.Option
	excludeLargeOptions := c.Query("exclude_large_options") == "true"
	common.OptionMapRWMutex.Lock()
	for k, v := range common.OptionMap {
		if strings.HasSuffix(k, "Token") ||
			strings.HasSuffix(k, "Secret") ||
			strings.HasSuffix(k, "Key") ||
			strings.HasSuffix(k, "secret") ||
			strings.HasSuffix(k, "api_key") {
			continue
		}
		value := common.Interface2String(v)
		if excludeLargeOptions && model_setting.IsMiniMaxLegacyRemovedOption(k) {
			value = "{}"
		}
		options = append(options, &model.Option{
			Key:   k,
			Value: value,
		})
	}
	common.OptionMapRWMutex.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
	return
}

type optionJsonMapEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type optionJsonMapResponse struct {
	Items    []optionJsonMapEntry `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Total    int                  `json:"total"`
}

type optionJsonArrayEntry struct {
	Value string `json:"value"`
}

type optionJsonArrayResponse struct {
	Items    []optionJsonArrayEntry `json:"items"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Total    int                    `json:"total"`
}

type OptionJsonMapDeleteRequest struct {
	Key    string `json:"key"`
	MapKey string `json:"map_key"`
}

type OptionJsonMapUpsertRequest struct {
	Key       string `json:"key"`
	MapKey    string `json:"map_key"`
	OldMapKey string `json:"old_map_key"`
	Value     string `json:"value"`
}

var pricingJsonMapOptionKeys = map[string]struct{}{
	"ModelPrice":           {},
	"ModelRatio":           {},
	"CacheRatio":           {},
	"CreateCacheRatio":     {},
	"CompletionRatio":      {},
	"AudioRatio":           {},
	"AudioCompletionRatio": {},
	"ContextPricing":       {},
}

func isPricingJsonMapOptionKey(key string) bool {
	_, ok := pricingJsonMapOptionKeys[key]
	return ok
}

func isSensitiveOptionKey(key string) bool {
	return strings.HasSuffix(key, "Token") ||
		strings.HasSuffix(key, "Secret") ||
		strings.HasSuffix(key, "Key") ||
		strings.HasSuffix(key, "secret") ||
		strings.HasSuffix(key, "api_key") ||
		strings.Contains(key, "Password") ||
		strings.Contains(key, "password")
}

func readOptionValue(key string) (string, bool) {
	common.OptionMapRWMutex.RLock()
	value, ok := common.OptionMap[key]
	common.OptionMapRWMutex.RUnlock()
	return value, ok
}

func GetOptionValue(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionKeyRequired),
		})
		return
	}
	if isSensitiveOptionKey(key) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionReadForbidden),
		})
		return
	}
	value, ok := readOptionValue(key)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionNotFound),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"key":   key,
			"value": value,
		},
	})
}

func readMiniMaxStringMapOption(c *gin.Context, key string) (map[string]string, string, bool) {
	if !model_setting.IsMiniMaxStringMapOption(key) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionJSONMapUnsupported),
		})
		return nil, "", false
	}
	value, ok := readOptionValue(key)
	if !ok || strings.TrimSpace(value) == "" {
		value = "{}"
	}
	items := map[string]string{}
	if err := common.UnmarshalJsonStr(value, &items); err != nil {
		common.ApiErrorI18n(c, i18n.MsgOptionJSONMapParseFailed, map[string]any{"Error": err.Error()})
		return nil, "", false
	}
	return items, value, true
}

func readPricingJsonMapOption(c *gin.Context, key string) (map[string]json.RawMessage, string, bool) {
	if !isPricingJsonMapOptionKey(key) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionJSONMapUnsupported),
		})
		return nil, "", false
	}
	value, ok := readOptionValue(key)
	if !ok || strings.TrimSpace(value) == "" {
		value = "{}"
	}
	items := map[string]json.RawMessage{}
	if err := common.UnmarshalJsonStr(value, &items); err != nil {
		common.ApiErrorI18n(c, i18n.MsgOptionJSONMapParseFailed, map[string]any{"Error": err.Error()})
		return nil, "", false
	}
	return items, value, true
}

func validatePricingJsonMapOption(key string, value string) error {
	if key == "ContextPricing" {
		return ratio_setting.ValidateContextPricing(value)
	}
	values := map[string]float64{}
	return common.UnmarshalJsonStr(value, &values)
}

func marshalPricingJsonMapOption(key string, items map[string]json.RawMessage) (string, error) {
	bytes, err := common.Marshal(items)
	if err != nil {
		return "", err
	}
	nextValue := string(bytes)
	if err := validatePricingJsonMapOption(key, nextValue); err != nil {
		return "", err
	}
	return nextValue, nil
}

// readMiniMaxStringArrayOption 仅用于兼容旧的 JSON 数组配置读取。
// 音色白名单已迁移到数据库音色表，这里对已废弃的 key 直接返回空数组与提示，
// 防止旧前端在数组编辑器里看到陈旧数据后误操作。
func readMiniMaxStringArrayOption(c *gin.Context, key string) ([]string, string, bool) {
	if key != "minimax.voice_whitelist" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionJSONArrayUnsupported),
		})
		return nil, "", false
	}
	// 已迁移至音色管理页面，统一返回空列表，避免暴露历史残留数据。
	return []string{}, "[]", true
}

func GetOptionJsonMap(c *gin.Context) {
	key := c.Query("key")
	var entries []optionJsonMapEntry
	var total int

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	keyword := strings.TrimSpace(c.Query("keyword"))

	if model_setting.IsMiniMaxStringMapOption(key) {
		items, _, ok := readMiniMaxStringMapOption(c, key)
		if !ok {
			return
		}
		keys := make([]string, 0, len(items))
		for itemKey := range items {
			if keyword == "" || strings.Contains(itemKey, keyword) || strings.Contains(items[itemKey], keyword) {
				keys = append(keys, itemKey)
			}
		}
		sort.Strings(keys)
		total = len(keys)
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		entries = make([]optionJsonMapEntry, 0, end-start)
		for _, itemKey := range keys[start:end] {
			entries = append(entries, optionJsonMapEntry{
				Key:   itemKey,
				Value: items[itemKey],
			})
		}
	} else {
		items, _, ok := readPricingJsonMapOption(c, key)
		if !ok {
			return
		}
		keys := make([]string, 0, len(items))
		for itemKey, itemValue := range items {
			value := string(itemValue)
			if keyword == "" || strings.Contains(itemKey, keyword) || strings.Contains(value, keyword) {
				keys = append(keys, itemKey)
			}
		}
		sort.Strings(keys)
		total = len(keys)
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		entries = make([]optionJsonMapEntry, 0, end-start)
		for _, itemKey := range keys[start:end] {
			entries = append(entries, optionJsonMapEntry{
				Key:   itemKey,
				Value: string(items[itemKey]),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": optionJsonMapResponse{
			Items:    entries,
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

func GetOptionJsonArray(c *gin.Context) {
	key := c.Query("key")
	items, _, ok := readMiniMaxStringArrayOption(c, key)
	if !ok {
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	entries := make([]optionJsonArrayEntry, 0, end-start)
	for _, item := range items[start:end] {
		entries = append(entries, optionJsonArrayEntry{Value: item})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": optionJsonArrayResponse{
			Items:    entries,
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

func DeleteOptionJsonArrayEntry(c *gin.Context) {
	// 音色白名单已迁移到数据库音色表，旧的 JSON 数组增删改接口不再支持写入。
	// 保留路由以兼容旧前端，但所有写入统一返回“已迁移”提示。
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": i18n.T(c, i18n.MsgOptionMigratedToVoiceManagement),
	})
}

func UpsertOptionJsonArrayEntry(c *gin.Context) {
	// 同上：音色白名单已迁移，写入不再支持。
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": i18n.T(c, i18n.MsgOptionMigratedToVoiceManagement),
	})
}

func DeleteOptionJsonMapEntry(c *gin.Context) {
	var req OptionJsonMapDeleteRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionInvalidParams),
		})
		return
	}
	if strings.TrimSpace(req.MapKey) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionMapKeyRequired),
		})
		return
	}
	var beforeValue, nextValue string
	if model_setting.IsMiniMaxStringMapOption(req.Key) {
		items, value, ok := readMiniMaxStringMapOption(c, req.Key)
		if !ok {
			return
		}
		beforeValue = value
		if _, exists := items[req.MapKey]; !exists {
			common.ApiErrorI18n(c, i18n.MsgOptionMapItemNotFound)
			return
		}

		delete(items, req.MapKey)
		bytes, err := common.Marshal(items)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		nextValue = string(bytes)
		if err := model_setting.ValidateMiniMaxOptionValue(req.Key, nextValue); err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionMiniMaxSettingFailed, map[string]any{"Error": err.Error()})
			return
		}
	} else {
		items, value, ok := readPricingJsonMapOption(c, req.Key)
		if !ok {
			return
		}
		beforeValue = value
		if _, exists := items[req.MapKey]; !exists {
			common.ApiErrorI18n(c, i18n.MsgOptionMapItemNotFound)
			return
		}

		delete(items, req.MapKey)
		var err error
		nextValue, err = marshalPricingJsonMapOption(req.Key, items)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}

	if err := model.UpdateOption(req.Key, nextValue); err != nil {
		common.ApiError(c, err)
		return
	}
	service.RecordAudit(
		c,
		model.AuditModuleOption,
		model.AuditActionUpdate,
		"删除 JSON 映射配置项 "+req.Key+"."+req.MapKey,
		map[string]interface{}{"option": req.Key, "value": beforeValue},
		map[string]interface{}{"option": req.Key, "value": nextValue},
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func UpsertOptionJsonMapEntry(c *gin.Context) {
	var req OptionJsonMapUpsertRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionInvalidParams),
		})
		return
	}

	mapKey := strings.TrimSpace(req.MapKey)
	oldMapKey := strings.TrimSpace(req.OldMapKey)
	if mapKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionMapKeyRequired),
		})
		return
	}

	var beforeValue, nextValue string
	if model_setting.IsMiniMaxStringMapOption(req.Key) {
		items, value, ok := readMiniMaxStringMapOption(c, req.Key)
		if !ok {
			return
		}
		beforeValue = value

		if oldMapKey != "" && oldMapKey != mapKey {
			if _, exists := items[oldMapKey]; !exists {
				common.ApiErrorI18n(c, i18n.MsgOptionOriginalMapItemNotFound)
				return
			}
			if _, exists := items[mapKey]; exists {
				common.ApiErrorI18n(c, i18n.MsgOptionMapKeyExists)
				return
			}
			delete(items, oldMapKey)
		}

		items[mapKey] = strings.TrimSpace(req.Value)
		bytes, err := common.Marshal(items)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		nextValue = string(bytes)
		if err := model_setting.ValidateMiniMaxOptionValue(req.Key, nextValue); err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionMiniMaxSettingFailed, map[string]any{"Error": err.Error()})
			return
		}
	} else {
		items, value, ok := readPricingJsonMapOption(c, req.Key)
		if !ok {
			return
		}
		beforeValue = value

		if oldMapKey != "" && oldMapKey != mapKey {
			if _, exists := items[oldMapKey]; !exists {
				common.ApiErrorI18n(c, i18n.MsgOptionOriginalMapItemNotFound)
				return
			}
			if _, exists := items[mapKey]; exists {
				common.ApiErrorI18n(c, i18n.MsgOptionMapKeyExists)
				return
			}
			delete(items, oldMapKey)
		}

		rawValue := json.RawMessage(strings.TrimSpace(req.Value))
		if len(rawValue) == 0 || !json.Valid(rawValue) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgOptionMapValueJSONRequired),
			})
			return
		}
		items[mapKey] = rawValue
		var err error
		nextValue, err = marshalPricingJsonMapOption(req.Key, items)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}

	if err := model.UpdateOption(req.Key, nextValue); err != nil {
		common.ApiError(c, err)
		return
	}
	service.RecordAudit(
		c,
		model.AuditModuleOption,
		model.AuditActionUpdate,
		"修改 JSON 映射配置项 "+req.Key+"."+mapKey,
		map[string]interface{}{"option": req.Key, "value": beforeValue},
		map[string]interface{}{"option": req.Key, "value": nextValue},
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

type OptionUpdateRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func optionUpdateValueToString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case bool:
		return common.Interface2String(v), true
	case float64:
		return common.Interface2String(v), true
	case int:
		return common.Interface2String(v), true
	default:
		return "", false
	}
}

func UpdateOption(c *gin.Context) {
	var option OptionUpdateRequest
	err := common.DecodeJson(c.Request.Body, &option)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionInvalidParams),
		})
		return
	}
	switch option.Key {
	case "Chats", "ChatLink", "ChatLink2",
		"WeChatAuthEnabled", "WeChatServerAddress", "WeChatServerToken", "WeChatAccountQRCodeImageURL",
		"oidc.enabled", "oidc.client_id", "oidc.client_secret", "oidc.well_known", "oidc.authorization_endpoint", "oidc.token_endpoint", "oidc.user_info_endpoint",
		"CreemApiKey", "CreemProducts", "CreemTestMode", "CreemWebhookSecret",
		"ExposeRatioEnabled", "ImageRatio",
		"global.third_party_multimodal_model_id", "global.third_party_multimodal_call_api_type",
		"global.third_party_multimodal_system_prompt", "global.third_party_multimodal_first_user_prompt",
		"global.third_party_multimodal_user_agent", "global.third_party_multimodal_x_title",
		"global.third_party_multimodal_http_referer",
		"quota_setting.enable_free_model_pre_consume":
		// Removed legacy features: do not allow recreating these options via API.
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionRemoved),
		})
		return
	}
	value, ok := optionUpdateValueToString(option.Value)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionValueTypeInvalid),
		})
		return
	}
	option.Value = value
	switch option.Key {
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GitHubClientId == "" {
			common.ApiErrorI18n(c, i18n.MsgOptionGitHubOAuthConfigRequired)
			return
		}
	case "LinuxDOOAuthEnabled":
		if option.Value == "true" && common.LinuxDOClientId == "" {
			common.ApiErrorI18n(c, i18n.MsgOptionLinuxDOOAuthConfigRequired)
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(common.EmailDomainWhitelist) == 0 {
			common.ApiErrorI18n(c, i18n.MsgOptionEmailDomainRequired)
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && common.TurnstileSiteKey == "" {
			common.ApiErrorI18n(c, i18n.MsgOptionTurnstileConfigRequired)

			return
		}
	case "GroupRatio":
		err = ratio_setting.CheckGroupRatio(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionAudioRatioFailed, map[string]any{"Error": err.Error()})
			return
		}
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionAudioCompletionRatioFailed, map[string]any{"Error": err.Error()})
			return
		}
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionCreateCacheRatioFailed, map[string]any{"Error": err.Error()})
			return
		}
	case "ContextPricing":
		err = ratio_setting.UpdateContextPricingByJSONString(option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionContextPricingFailed, map[string]any{"Error": err.Error()})
			return
		}
	case "ModelRequestRateLimitGroup":
		err = setting.CheckModelRequestRateLimitGroup(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AutomaticDisableStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AutomaticRetryStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "RetryTimes":
		// RetryTimes must be a non-negative integer strictly less than 100.
		retryValue, parseErr := strconv.Atoi(strings.TrimSpace(option.Value.(string)))
		if parseErr != nil || retryValue < 0 || retryValue >= 100 {
			common.ApiErrorI18n(c, i18n.MsgOptionRetryTimesRange)
			return
		}
		// When enabling retry the count must be positive; allow 0 only when retry is disabled.
		if common.AutomaticRetryEnabled && retryValue <= 0 {
			common.ApiErrorI18n(c, i18n.MsgOptionRetryTimesPositiveWhenEnable)
			return
		}
	case "AutomaticRetryEnabled":
		if option.Value != "true" && option.Value != "false" {
			common.ApiErrorI18n(c, i18n.MsgOptionRetryEnabledMustBool)
			return
		}
		// Turning retry on requires a positive RetryTimes.
		if option.Value == "true" && common.RetryTimes <= 0 {
			common.ApiErrorI18n(c, i18n.MsgOptionRetryEnableNeedsPositive)
			return
		}
	case "SidebarModulesAdmin":
		// No additional validation needed; frontend manages the config.
	case "console_setting.api_info":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "ApiInfo")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.announcements":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "Announcements")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.faq":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "FAQ")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.uptime_kuma_groups":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "UptimeKumaGroups")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.usage_log_fields":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "UsageLogFields")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "tool_billing_setting.rules":
		// 旧格式（带 quality/size/model_filter/provider 字段）自动迁移为新 conditions 格式
		migrated, didMigrate, migrateErr := operation_setting.MigrateLegacyRules(option.Value.(string))
		if migrateErr != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionToolBillingRulesParseFailed, map[string]any{"Error": migrateErr.Error()})
			return
		}
		if didMigrate {
			option.Value = migrated
		}
		err = operation_setting.ValidateToolBillingRules(option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionToolBillingRulesSetFailed, map[string]any{"Error": err.Error()})
			return
		}
	case "minimax.model_redirect", "minimax.emotion_redirect",
		"minimax.tone_word_redirect":
		// 仍保留的 JSON 映射型 MiniMax 选项：保存前校验。
		if err := model_setting.ValidateMiniMaxOptionValue(option.Key, option.Value.(string)); err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionMiniMaxSettingFailed, map[string]any{"Error": err.Error()})
			return
		}
	case "minimax.emotion_pattern", "minimax.tone_word_pattern":
		if err := model_setting.ValidateMiniMaxOptionValue(option.Key, option.Value.(string)); err != nil {
			common.ApiErrorI18n(c, i18n.MsgOptionMiniMaxSettingFailed, map[string]any{"Error": err.Error()})
			return
		}
	case "minimax.voice_whitelist", "minimax.voice_redirect":
		// 已迁移到数据库音色表，拒绝写入旧 key，提示通过音色管理维护。
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgOptionMigratedToVoiceManagement),
		})
		return
	}
	// 读取旧值用于审计 before 字段。必须在 UpdateOption 之前读取，
	// 否则 OptionMap 已被新值覆盖。OptionMap 中可能不存在该 key（首次创建）。
	common.OptionMapRWMutex.RLock()
	oldValue, hasOld := common.OptionMap[option.Key]
	common.OptionMapRWMutex.RUnlock()

	err = model.UpdateOption(option.Key, option.Value.(string))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if option.Key == "DataExportInterval" {
		service.ClearRankingsCache()
	}
	// 审计系统设置变更：对敏感配置值脱敏，防止凭证泄露到审计日志。
	// 复用 GetOptions 中的敏感 key 后缀规则。
	isSensitiveKey := strings.HasSuffix(option.Key, "Token") ||
		strings.HasSuffix(option.Key, "Secret") ||
		strings.HasSuffix(option.Key, "Key") ||
		strings.HasSuffix(option.Key, "secret") ||
		strings.HasSuffix(option.Key, "api_key") ||
		strings.Contains(option.Key, "Password") ||
		strings.Contains(option.Key, "password")
	auditValue := option.Value
	if isSensitiveKey {
		auditValue = "[REDACTED]"
	}
	// 构建 before：在 UpdateOption 之前已读取 oldValue，敏感 key 同样脱敏。
	var beforeMap map[string]interface{}
	if hasOld {
		oldAuditValue := oldValue
		if isSensitiveKey {
			oldAuditValue = "[REDACTED]"
		}
		beforeMap = map[string]interface{}{"key": option.Key, "value": oldAuditValue}
	}
	// P1-3: 审计配置本身的变更必须被记录。
	// model.UpdateOption 已在上面执行，此时 audit_setting 已是最新值。
	// 如果管理员关闭了审计总开关或 option 模块，RecordAudit 会按新配置跳过。
	// 因此对 audit_setting.* 的任何变更都强制记录。
	forceRecord := strings.HasPrefix(option.Key, "audit_setting.")
	service.RecordAudit(c, model.AuditModuleOption, model.AuditActionUpdate, "修改系统设置 "+option.Key, beforeMap, map[string]interface{}{"key": option.Key, "value": auditValue}, forceRecord)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}
