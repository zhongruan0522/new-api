package controller

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/zhongruan0522/new-api/common"
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
			"message": "缺少配置项 key",
		})
		return
	}
	if isSensitiveOptionKey(key) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "该配置项不允许读取",
		})
		return
	}
	value, ok := readOptionValue(key)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "配置项不存在",
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
			"message": "不支持的 JSON 映射配置项",
		})
		return nil, "", false
	}
	value, ok := readOptionValue(key)
	if !ok || strings.TrimSpace(value) == "" {
		value = "{}"
	}
	items := map[string]string{}
	if err := common.UnmarshalJsonStr(value, &items); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "JSON 映射配置解析失败: " + err.Error(),
		})
		return nil, "", false
	}
	return items, value, true
}

// readMiniMaxStringArrayOption 仅用于兼容旧的 JSON 数组配置读取。
// 音色白名单已迁移到数据库音色表，这里对已废弃的 key 直接返回空数组与提示，
// 防止旧前端在数组编辑器里看到陈旧数据后误操作。
func readMiniMaxStringArrayOption(c *gin.Context, key string) ([]string, string, bool) {
	if key != "minimax.voice_whitelist" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "不支持的 JSON 数组配置项",
		})
		return nil, "", false
	}
	// 已迁移至音色管理页面，统一返回空列表，避免暴露历史残留数据。
	return []string{}, "[]", true
}

func GetOptionJsonMap(c *gin.Context) {
	key := c.Query("key")
	items, _, ok := readMiniMaxStringMapOption(c, key)
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

	keys := make([]string, 0, len(items))
	for itemKey := range items {
		keys = append(keys, itemKey)
	}
	sort.Strings(keys)

	total := len(keys)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	entries := make([]optionJsonMapEntry, 0, end-start)
	for _, itemKey := range keys[start:end] {
		entries = append(entries, optionJsonMapEntry{
			Key:   itemKey,
			Value: items[itemKey],
		})
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
		"message": "该配置已迁移至音色管理页面，请通过音色管理进行维护",
	})
}

func UpsertOptionJsonArrayEntry(c *gin.Context) {
	// 同上：音色白名单已迁移，写入不再支持。
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该配置已迁移至音色管理页面，请通过音色管理进行维护",
	})
}

func DeleteOptionJsonMapEntry(c *gin.Context) {
	var req OptionJsonMapDeleteRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	if strings.TrimSpace(req.MapKey) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "映射键不能为空",
		})
		return
	}
	items, beforeValue, ok := readMiniMaxStringMapOption(c, req.Key)
	if !ok {
		return
	}
	if _, exists := items[req.MapKey]; !exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "映射项不存在",
		})
		return
	}

	delete(items, req.MapKey)
	bytes, err := common.Marshal(items)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	nextValue := string(bytes)
	if err := model_setting.ValidateMiniMaxOptionValue(req.Key, nextValue); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "MiniMax 设置失败: " + err.Error(),
		})
		return
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
			"message": "无效的参数",
		})
		return
	}

	mapKey := strings.TrimSpace(req.MapKey)
	oldMapKey := strings.TrimSpace(req.OldMapKey)
	if mapKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "映射键不能为空",
		})
		return
	}

	items, beforeValue, ok := readMiniMaxStringMapOption(c, req.Key)
	if !ok {
		return
	}

	if oldMapKey != "" && oldMapKey != mapKey {
		if _, exists := items[oldMapKey]; !exists {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "原映射项不存在",
			})
			return
		}
		if _, exists := items[mapKey]; exists {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "映射键已存在",
			})
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
	nextValue := string(bytes)
	if err := model_setting.ValidateMiniMaxOptionValue(req.Key, nextValue); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "MiniMax 设置失败: " + err.Error(),
		})
		return
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
			"message": "无效的参数",
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
			"message": "该配置已移除",
		})
		return
	}
	value, ok := optionUpdateValueToString(option.Value)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "配置值必须是字符串、布尔值或数字",
		})
		return
	}
	option.Value = value
	switch option.Key {
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GitHubClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 GitHub OAuth，请先填入 GitHub Client Id 以及 GitHub Client Secret！",
			})
			return
		}
	case "LinuxDOOAuthEnabled":
		if option.Value == "true" && common.LinuxDOClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 LinuxDO OAuth，请先填入 LinuxDO Client Id 以及 LinuxDO Client Secret！",
			})
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(common.EmailDomainWhitelist) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用邮箱域名限制，请先填入限制的邮箱域名！",
			})
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && common.TurnstileSiteKey == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Turnstile 校验，请先填入 Turnstile 校验相关配置信息！",
			})

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
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "音频倍率设置失败: " + err.Error(),
			})
			return
		}
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "音频补全倍率设置失败: " + err.Error(),
			})
			return
		}
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "缓存创建倍率设置失败: " + err.Error(),
			})
			return
		}
	case "ContextPricing":
		err = ratio_setting.UpdateContextPricingByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "分段计费设置失败: " + err.Error(),
			})
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
		// RetryTimes must be a non-negative integer capped at 10.
		retryValue, parseErr := strconv.Atoi(strings.TrimSpace(option.Value.(string)))
		if parseErr != nil || retryValue < 0 || retryValue > 10 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "重试次数必须是 0 到 10 之间的整数",
			})
			return
		}
		// When enabling retry the count must be positive; allow 0 only when retry is disabled.
		if common.AutomaticRetryEnabled && retryValue <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "自动重试已启用，重试次数必须大于 0",
			})
			return
		}
	case "AutomaticRetryEnabled":
		if option.Value != "true" && option.Value != "false" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "自动重试开关必须是布尔值",
			})
			return
		}
		// Turning retry on requires a positive RetryTimes.
		if option.Value == "true" && common.RetryTimes <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用自动重试：请先将重试次数设置为大于 0 的值",
			})
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
		err = operation_setting.ValidateToolBillingRules(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "工具计费规则设置失败: " + err.Error(),
			})
			return
		}
	case "minimax.model_redirect", "minimax.emotion_redirect",
		"minimax.tone_word_redirect":
		// 仍保留的 JSON 映射型 MiniMax 选项：保存前校验。
		if err := model_setting.ValidateMiniMaxOptionValue(option.Key, option.Value.(string)); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "MiniMax 设置失败: " + err.Error(),
			})
			return
		}
	case "minimax.emotion_pattern", "minimax.tone_word_pattern":
		if err := model_setting.ValidateMiniMaxOptionValue(option.Key, option.Value.(string)); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "MiniMax 设置失败: " + err.Error(),
			})
			return
		}
	case "minimax.voice_whitelist", "minimax.voice_redirect":
		// 已迁移到数据库音色表，拒绝写入旧 key，提示通过音色管理维护。
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "该配置已迁移至音色管理页面，请通过音色管理进行维护",
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
