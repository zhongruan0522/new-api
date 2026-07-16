package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting"
	"github.com/zhongruan0522/new-api/setting/config"
	"github.com/zhongruan0522/new-api/setting/model_setting"
	"github.com/zhongruan0522/new-api/setting/operation_setting"
	"github.com/zhongruan0522/new-api/setting/performance_setting"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
	"github.com/zhongruan0522/new-api/setting/system_setting"
)

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

func AllOption() ([]*Option, error) {
	var options []*Option
	var err error
	err = DB.Find(&options).Error
	return options, err
}

func InitOptionMap() {
	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)

	// 添加原有的系统配置
	common.OptionMap["FileUploadPermission"] = strconv.Itoa(common.FileUploadPermission)
	common.OptionMap["FileDownloadPermission"] = strconv.Itoa(common.FileDownloadPermission)
	common.OptionMap["ImageUploadPermission"] = strconv.Itoa(common.ImageUploadPermission)
	common.OptionMap["ImageDownloadPermission"] = strconv.Itoa(common.ImageDownloadPermission)
	common.OptionMap["PasswordLoginEnabled"] = strconv.FormatBool(common.PasswordLoginEnabled)
	common.OptionMap["PasswordRegisterEnabled"] = strconv.FormatBool(common.PasswordRegisterEnabled)
	common.OptionMap["EmailVerificationEnabled"] = strconv.FormatBool(common.EmailVerificationEnabled)
	common.OptionMap["GitHubOAuthEnabled"] = strconv.FormatBool(common.GitHubOAuthEnabled)
	common.OptionMap["LinuxDOOAuthEnabled"] = strconv.FormatBool(common.LinuxDOOAuthEnabled)
	common.OptionMap["TurnstileCheckEnabled"] = strconv.FormatBool(common.TurnstileCheckEnabled)
	common.OptionMap["RegisterEnabled"] = strconv.FormatBool(common.RegisterEnabled)
	common.OptionMap["AutomaticDisableChannelEnabled"] = strconv.FormatBool(common.AutomaticDisableChannelEnabled)
	common.OptionMap["AutomaticEnableChannelEnabled"] = strconv.FormatBool(common.AutomaticEnableChannelEnabled)
	common.OptionMap["LogConsumeEnabled"] = strconv.FormatBool(common.LogConsumeEnabled)
	common.OptionMap["DataExportEnabled"] = strconv.FormatBool(common.DataExportEnabled)
	common.OptionMap["ChannelDisableThreshold"] = strconv.FormatFloat(common.ChannelDisableThreshold, 'f', -1, 64)
	common.OptionMap["EmailDomainRestrictionEnabled"] = strconv.FormatBool(common.EmailDomainRestrictionEnabled)
	common.OptionMap["EmailAliasRestrictionEnabled"] = strconv.FormatBool(common.EmailAliasRestrictionEnabled)
	common.OptionMap["EmailDomainWhitelist"] = strings.Join(common.EmailDomainWhitelist, ",")
	common.OptionMap["SMTPServer"] = ""
	common.OptionMap["SMTPFrom"] = ""
	common.OptionMap["SMTPPort"] = strconv.Itoa(common.SMTPPort)
	common.OptionMap["SMTPAccount"] = ""
	common.OptionMap["SMTPToken"] = ""
	common.OptionMap["SMTPSSLEnabled"] = strconv.FormatBool(common.SMTPSSLEnabled)
	common.OptionMap["SMTPForceLoginAuthEnabled"] = strconv.FormatBool(common.SMTPForceLoginAuthEnabled)
	common.OptionMap["Notice"] = ""
	common.OptionMap["About"] = ""
	common.OptionMap["HomePageContent"] = ""
	common.OptionMap["Footer"] = common.Footer
	common.OptionMap["SystemName"] = common.SystemName
	common.OptionMap["Logo"] = common.Logo
	common.OptionMap["ServerAddress"] = ""
	common.OptionMap["WorkerUrl"] = system_setting.WorkerUrl
	common.OptionMap["WorkerValidKey"] = system_setting.WorkerValidKey
	common.OptionMap["WorkerAllowHttpImageRequestEnabled"] = strconv.FormatBool(system_setting.WorkerAllowHttpImageRequestEnabled)
	common.OptionMap["PayAddress"] = ""
	common.OptionMap["CustomCallbackAddress"] = ""
	common.OptionMap["EpayId"] = ""
	common.OptionMap["EpayKey"] = ""
	common.OptionMap["Price"] = strconv.FormatFloat(operation_setting.Price, 'f', -1, 64)
	common.OptionMap["MinTopUp"] = strconv.Itoa(operation_setting.MinTopUp)
	common.OptionMap["StripeMinTopUp"] = strconv.Itoa(setting.StripeMinTopUp)
	common.OptionMap["StripeApiSecret"] = setting.StripeApiSecret
	common.OptionMap["StripeWebhookSecret"] = setting.StripeWebhookSecret
	common.OptionMap["StripePriceId"] = setting.StripePriceId
	common.OptionMap["StripeUnitPrice"] = strconv.FormatFloat(setting.StripeUnitPrice, 'f', -1, 64)
	common.OptionMap["StripePromotionCodesEnabled"] = strconv.FormatBool(setting.StripePromotionCodesEnabled)
	common.OptionMap["TopupGroupRatio"] = common.TopupGroupRatio2JSONString()
	common.OptionMap["AutoGroups"] = setting.AutoGroups2JsonString()
	common.OptionMap["DefaultUseAutoGroup"] = strconv.FormatBool(setting.DefaultUseAutoGroup)
	common.OptionMap["PayMethods"] = operation_setting.PayMethods2JsonString()
	common.OptionMap["GitHubClientId"] = ""
	common.OptionMap["GitHubClientSecret"] = ""
	common.OptionMap["TurnstileSiteKey"] = ""
	common.OptionMap["TurnstileSecretKey"] = ""
	common.OptionMap["QuotaForNewUser"] = strconv.Itoa(common.QuotaForNewUser)
	common.OptionMap["QuotaForInviter"] = strconv.Itoa(common.QuotaForInviter)
	common.OptionMap["QuotaForInvitee"] = strconv.Itoa(common.QuotaForInvitee)
	common.OptionMap["QuotaRemindThreshold"] = strconv.Itoa(common.QuotaRemindThreshold)
	common.OptionMap["PreConsumedQuota"] = strconv.Itoa(common.PreConsumedQuota)
	common.OptionMap["ModelRequestRateLimitCount"] = strconv.Itoa(setting.ModelRequestRateLimitCount)
	common.OptionMap["ModelRequestRateLimitDurationMinutes"] = strconv.Itoa(setting.ModelRequestRateLimitDurationMinutes)
	common.OptionMap["ModelRequestRateLimitSuccessCount"] = strconv.Itoa(setting.ModelRequestRateLimitSuccessCount)
	common.OptionMap["ModelRequestRateLimitGroup"] = setting.ModelRequestRateLimitGroup2JSONString()
	common.OptionMap["ModelRatio"] = ratio_setting.ModelRatio2JSONString()
	common.OptionMap["ModelPrice"] = ratio_setting.ModelPrice2JSONString()
	common.OptionMap["CacheRatio"] = ratio_setting.CacheRatio2JSONString()
	common.OptionMap["CreateCacheRatio"] = ratio_setting.CreateCacheRatio2JSONString()
	common.OptionMap["ContextPricing"] = ratio_setting.ContextPricing2JSONString()
	common.OptionMap["GroupRatio"] = ratio_setting.GroupRatio2JSONString()
	common.OptionMap["GroupGroupRatio"] = ratio_setting.GroupGroupRatio2JSONString()
	common.OptionMap["UserUsableGroups"] = setting.UserUsableGroups2JSONString()
	common.OptionMap["CompletionRatio"] = ratio_setting.CompletionRatio2JSONString()
	common.OptionMap["AudioRatio"] = ratio_setting.AudioRatio2JSONString()
	common.OptionMap["AudioCompletionRatio"] = ratio_setting.AudioCompletionRatio2JSONString()
	common.OptionMap["TopUpLink"] = common.TopUpLink
	common.OptionMap["QuotaPerUnit"] = strconv.FormatFloat(common.QuotaPerUnit, 'f', -1, 64)
	common.OptionMap["RetryTimes"] = strconv.Itoa(common.RetryTimes)
	common.OptionMap["AutomaticRetryEnabled"] = strconv.FormatBool(common.AutomaticRetryEnabled)
	common.OptionMap["DataExportInterval"] = strconv.Itoa(common.DataExportInterval)
	common.OptionMap["DataExportDefaultTime"] = common.DataExportDefaultTime
	common.OptionMap["DefaultCollapseSidebar"] = strconv.FormatBool(common.DefaultCollapseSidebar)
	common.OptionMap["CheckSensitiveEnabled"] = strconv.FormatBool(setting.CheckSensitiveEnabled)

	common.OptionMap["ModelRequestRateLimitEnabled"] = strconv.FormatBool(setting.ModelRequestRateLimitEnabled)
	common.OptionMap["CheckSensitiveOnPromptEnabled"] = strconv.FormatBool(setting.CheckSensitiveOnPromptEnabled)
	common.OptionMap["StopOnSensitiveEnabled"] = strconv.FormatBool(setting.StopOnSensitiveEnabled)
	common.OptionMap["SensitiveWords"] = setting.SensitiveWordsToString()
	common.OptionMap["StreamCacheQueueLength"] = strconv.Itoa(setting.StreamCacheQueueLength)
	common.OptionMap["AutomaticDisableKeywords"] = operation_setting.AutomaticDisableKeywordsToString()
	common.OptionMap["AutomaticDisableStatusCodes"] = operation_setting.AutomaticDisableStatusCodesToString()
	common.OptionMap["AutomaticRetryStatusCodes"] = operation_setting.AutomaticRetryStatusCodesToString()
	common.OptionMap["DynamicRatioEnabled"] = strconv.FormatBool(common.DynamicRatioEnabled)

	// 自动添加所有注册的模型配置
	modelConfigs := config.GlobalConfig.ExportAllConfigs()
	for k, v := range modelConfigs {
		common.OptionMap[k] = v
	}

	common.OptionMapRWMutex.Unlock()
	loadOptionsFromDatabase()
}

func loadOptionsFromDatabase() {
	options, _ := AllOption()
	toolBillingMigrated, migratedToolBillingRules := migrateLegacyToolBillingRulesInOptions(options)

	for _, option := range options {
		err := updateOptionMap(option.Key, option.Value)
		if err != nil {
			common.SysLog("failed to update option map: " + err.Error())
		}
	}
	if toolBillingMigrated {
		persistMigratedToolBillingRules(migratedToolBillingRules)
	}

	// One-time migration: if the removed toggle "quota_setting.enable_free_model_pre_consume"
	// exists in DB and is "false", override FreeModelPreConsumedQuota to 0 so that
	// users who had disabled free-model pre-consumption don't suddenly get charged
	// after upgrade. Then persist the migrated value. Always delete the stale row
	// regardless of its value since the toggle has been removed entirely.
	for _, option := range options {
		if option.Key == "quota_setting.enable_free_model_pre_consume" {
			if option.Value == "false" {
				quotaSetting := operation_setting.GetQuotaSetting()
				quotaSetting.FreeModelPreConsumedQuota = 0
				// Sync to OptionMap so API returns the correct value
				common.OptionMap["quota_setting.free_model_pre_consumed_quota"] = "0"
				// Persist the migrated quota to DB so it survives restart
				migratedOption := Option{Key: "quota_setting.free_model_pre_consumed_quota"}
				DB.FirstOrCreate(&migratedOption, Option{Key: "quota_setting.free_model_pre_consumed_quota"})
				DB.Model(&migratedOption).Update("value", "0")
				common.SysLog("migrated quota_setting.enable_free_model_pre_consume=false -> free_model_pre_consumed_quota=0")
			}
			// Delete the stale toggle row regardless of value
			DB.Where(commonKeyCol+" = ?", "quota_setting.enable_free_model_pre_consume").Delete(&Option{})
			break
		}
	}

	// Backward-compatibility migration: AutomaticRetryEnabled was introduced as a
	// standalone toggle. On older deployments the row may not exist yet. To avoid
	// silently turning off an already-configured retry setup after upgrade, derive
	// the initial enabled state from the existing RetryTimes: if an admin had set
	// RetryTimes > 0 we treat automatic retry as enabled.
	retryEnabledExists := false
	for _, option := range options {
		if option.Key == "AutomaticRetryEnabled" {
			retryEnabledExists = true
			break
		}
	}
	if !retryEnabledExists {
		derivedEnabled := common.RetryTimes > 0
		common.AutomaticRetryEnabled = derivedEnabled
		common.OptionMap["AutomaticRetryEnabled"] = strconv.FormatBool(derivedEnabled)
		migratedOption := Option{Key: "AutomaticRetryEnabled"}
		DB.FirstOrCreate(&migratedOption, Option{Key: "AutomaticRetryEnabled"})
		DB.Model(&migratedOption).Update("value", strconv.FormatBool(derivedEnabled))
	}

}

func migrateLegacyToolBillingRulesInOptions(options []*Option) (bool, string) {
	for i := range options {
		if options[i] == nil || options[i].Key != "tool_billing_setting.rules" || options[i].Value == "" {
			continue
		}
		migrated, didMigrate, err := operation_setting.MigrateLegacyRules(options[i].Value)
		if err != nil {
			common.SysError("failed to migrate tool_billing_setting.rules: " + err.Error())
			return false, ""
		}
		if !didMigrate {
			return false, ""
		}
		options[i].Value = migrated
		return true, migrated
	}
	return false, ""
}

func persistMigratedToolBillingRules(migrated string) {
	migratedOption := Option{Key: "tool_billing_setting.rules"}
	if err := DB.FirstOrCreate(&migratedOption, Option{Key: "tool_billing_setting.rules"}).Error; err != nil {
		common.SysError("failed to create migrated tool_billing_setting.rules: " + err.Error())
		return
	}
	if err := DB.Model(&migratedOption).Update("value", migrated).Error; err != nil {
		common.SysError("failed to persist migrated tool_billing_setting.rules: " + err.Error())
		return
	}
	common.SysLog("migrated tool_billing_setting.rules to conditions format")
}

func SyncOptions(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing options from database")
		loadOptionsFromDatabase()
	}
}

func UpdateOption(key string, value string) error {
	switch key {
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
		return errors.New("option removed")
	}
	if err := validateConfigUpdate(key, value); err != nil {
		return err
	}
	// Save to database first
	option := Option{
		Key: key,
	}
	// https://gorm.io/docs/update.html#Save-All-Fields
	if err := DB.FirstOrCreate(&option, Option{Key: key}).Error; err != nil {
		return err
	}
	option.Value = value
	// Save is a combination function.
	// If save value does not contain primary key, it will execute Create,
	// otherwise it will execute Update (with all fields).
	if err := DB.Save(&option).Error; err != nil {
		return err
	}
	// Update OptionMap
	return updateOptionMap(key, value)
}

func updateOptionMap(key string, value string) (err error) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()

	// Removed legacy options: keep DB compatibility but do not load them into memory.
	// (Avoid resurrecting removed features via SyncOptions or stale DB rows.)
	switch key {
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
		delete(common.OptionMap, key)
		return nil
	}
	// 检查是否是模型配置 - 使用更规范的方式处理
	if handled, cfgErr := handleConfigUpdate(key, value); handled {
		if cfgErr != nil {
			return cfgErr
		}
		common.OptionMap[key] = value
		return nil // 已由配置系统处理
	}
	common.OptionMap[key] = value

	// 处理传统配置项...
	if strings.HasSuffix(key, "Permission") {
		intValue, _ := strconv.Atoi(value)
		switch key {
		case "FileUploadPermission":
			common.FileUploadPermission = intValue
		case "FileDownloadPermission":
			common.FileDownloadPermission = intValue
		case "ImageUploadPermission":
			common.ImageUploadPermission = intValue
		case "ImageDownloadPermission":
			common.ImageDownloadPermission = intValue
		}
	}
	if strings.HasSuffix(key, "Enabled") || key == "DefaultCollapseSidebar" || key == "DefaultUseAutoGroup" {
		boolValue := value == "true"
		switch key {
		case "PasswordRegisterEnabled":
			common.PasswordRegisterEnabled = boolValue
		case "PasswordLoginEnabled":
			common.PasswordLoginEnabled = boolValue
		case "EmailVerificationEnabled":
			common.EmailVerificationEnabled = boolValue
		case "GitHubOAuthEnabled":
			common.GitHubOAuthEnabled = boolValue
		case "LinuxDOOAuthEnabled":
			common.LinuxDOOAuthEnabled = boolValue
		case "TurnstileCheckEnabled":
			common.TurnstileCheckEnabled = boolValue
		case "RegisterEnabled":
			common.RegisterEnabled = boolValue
		case "EmailDomainRestrictionEnabled":
			common.EmailDomainRestrictionEnabled = boolValue
		case "EmailAliasRestrictionEnabled":
			common.EmailAliasRestrictionEnabled = boolValue
		case "AutomaticDisableChannelEnabled":
			common.AutomaticDisableChannelEnabled = boolValue
		case "AutomaticEnableChannelEnabled":
			common.AutomaticEnableChannelEnabled = boolValue
		case "LogConsumeEnabled":
			common.LogConsumeEnabled = boolValue
		case "DataExportEnabled":
			common.DataExportEnabled = boolValue
		case "DefaultCollapseSidebar":
			common.DefaultCollapseSidebar = boolValue
		case "CheckSensitiveEnabled":
			setting.CheckSensitiveEnabled = boolValue
		case "CheckSensitiveOnPromptEnabled":
			setting.CheckSensitiveOnPromptEnabled = boolValue
		case "ModelRequestRateLimitEnabled":
			setting.ModelRequestRateLimitEnabled = boolValue
		case "StopOnSensitiveEnabled":
			setting.StopOnSensitiveEnabled = boolValue
		case "SMTPSSLEnabled":
			common.SMTPSSLEnabled = boolValue
		case "WorkerAllowHttpImageRequestEnabled":
			system_setting.WorkerAllowHttpImageRequestEnabled = boolValue
		case "DefaultUseAutoGroup":
			setting.DefaultUseAutoGroup = boolValue
		case "DynamicRatioEnabled":
			common.DynamicRatioEnabled = boolValue
		case "AutomaticRetryEnabled":
			common.AutomaticRetryEnabled = boolValue
		}
	}
	switch key {
	case "EmailDomainWhitelist":
		common.EmailDomainWhitelist = strings.Split(value, ",")
	case "SMTPServer":
		common.SMTPServer = value
	case "SMTPPort":
		intValue, _ := strconv.Atoi(value)
		common.SMTPPort = intValue
	case "SMTPAccount":
		common.SMTPAccount = value
	case "SMTPFrom":
		common.SMTPFrom = value
	case "SMTPToken":
		common.SMTPToken = value
	case "ServerAddress":
		system_setting.ServerAddress = value
	case "WorkerUrl":
		system_setting.WorkerUrl = value
	case "WorkerValidKey":
		system_setting.WorkerValidKey = value
	case "PayAddress":
		operation_setting.PayAddress = value
	case "AutoGroups":
		err = setting.UpdateAutoGroupsByJsonString(value)
	case "CustomCallbackAddress":
		operation_setting.CustomCallbackAddress = value
	case "EpayId":
		operation_setting.EpayId = value
	case "EpayKey":
		operation_setting.EpayKey = value
	case "Price":
		operation_setting.Price, _ = strconv.ParseFloat(value, 64)
	case "MinTopUp":
		operation_setting.MinTopUp, _ = strconv.Atoi(value)
	case "StripeApiSecret":
		setting.StripeApiSecret = value
	case "StripeWebhookSecret":
		setting.StripeWebhookSecret = value
	case "StripePriceId":
		setting.StripePriceId = value
	case "StripeUnitPrice":
		setting.StripeUnitPrice, _ = strconv.ParseFloat(value, 64)
	case "StripeMinTopUp":
		setting.StripeMinTopUp, _ = strconv.Atoi(value)
	case "StripePromotionCodesEnabled":
		setting.StripePromotionCodesEnabled = value == "true"
	case "TopupGroupRatio":
		err = common.UpdateTopupGroupRatioByJSONString(value)
	case "GitHubClientId":
		common.GitHubClientId = value
	case "GitHubClientSecret":
		common.GitHubClientSecret = value
	case "LinuxDOClientId":
		common.LinuxDOClientId = value
	case "LinuxDOClientSecret":
		common.LinuxDOClientSecret = value
	case "LinuxDOMinimumTrustLevel":
		common.LinuxDOMinimumTrustLevel, _ = strconv.Atoi(value)
	case "Footer":
		common.Footer = value
	case "SystemName":
		common.SystemName = value
	case "Logo":
		common.Logo = value
	case "TurnstileSiteKey":
		common.TurnstileSiteKey = value
	case "TurnstileSecretKey":
		common.TurnstileSecretKey = value
	case "QuotaForNewUser":
		common.QuotaForNewUser, _ = strconv.Atoi(value)
	case "QuotaForInviter":
		common.QuotaForInviter, _ = strconv.Atoi(value)
	case "QuotaForInvitee":
		common.QuotaForInvitee, _ = strconv.Atoi(value)
	case "QuotaRemindThreshold":
		common.QuotaRemindThreshold, _ = strconv.Atoi(value)
	case "PreConsumedQuota":
		common.PreConsumedQuota, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitCount":
		setting.ModelRequestRateLimitCount, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitDurationMinutes":
		setting.ModelRequestRateLimitDurationMinutes, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitSuccessCount":
		setting.ModelRequestRateLimitSuccessCount, _ = strconv.Atoi(value)
	case "ModelRequestRateLimitGroup":
		err = setting.UpdateModelRequestRateLimitGroupByJSONString(value)
	case "RetryTimes":
		common.RetryTimes, _ = strconv.Atoi(value)
	case "DataExportInterval":
		common.DataExportInterval, _ = strconv.Atoi(value)
	case "DataExportDefaultTime":
		common.DataExportDefaultTime = value
	case "ModelRatio":
		err = ratio_setting.UpdateModelRatioByJSONString(value)
	case "GroupRatio":
		err = ratio_setting.UpdateGroupRatioByJSONString(value)
	case "GroupGroupRatio":
		err = ratio_setting.UpdateGroupGroupRatioByJSONString(value)
	case "UserUsableGroups":
		err = setting.UpdateUserUsableGroupsByJSONString(value)
	case "CompletionRatio":
		err = ratio_setting.UpdateCompletionRatioByJSONString(value)
	case "ModelPrice":
		err = ratio_setting.UpdateModelPriceByJSONString(value)
	case "CacheRatio":
		err = ratio_setting.UpdateCacheRatioByJSONString(value)
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(value)
	case "ContextPricing":
		err = ratio_setting.UpdateContextPricingByJSONString(value)
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(value)
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(value)
	case "TopUpLink":
		common.TopUpLink = value
	case "ChannelDisableThreshold":
		common.ChannelDisableThreshold, _ = strconv.ParseFloat(value, 64)
	case "QuotaPerUnit":
		common.QuotaPerUnit, _ = strconv.ParseFloat(value, 64)
	case "SensitiveWords":
		setting.SensitiveWordsFromString(value)
	case "AutomaticDisableKeywords":
		operation_setting.AutomaticDisableKeywordsFromString(value)
	case "AutomaticDisableStatusCodes":
		err = operation_setting.AutomaticDisableStatusCodesFromString(value)
	case "AutomaticRetryStatusCodes":
		err = operation_setting.AutomaticRetryStatusCodesFromString(value)
	case "StreamCacheQueueLength":
		setting.StreamCacheQueueLength, _ = strconv.Atoi(value)
	case "PayMethods":
		err = operation_setting.UpdatePayMethodsByJsonString(value)
	}
	return err
}

func validateConfigUpdate(key, value string) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return nil
	}

	cfg := config.GlobalConfig.Get(parts[0])
	if cfg == nil {
		return nil
	}
	if parts[0] == "minimax" {
		return model_setting.WithMiniMaxSettingsReadLock(func() error {
			return config.ValidateConfigFromMap(cfg, map[string]string{parts[1]: value})
		})
	}

	return config.ValidateConfigFromMap(cfg, map[string]string{parts[1]: value})
}

// handleConfigUpdate 处理分层配置更新，返回是否已处理
func handleConfigUpdate(key, value string) (bool, error) {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return false, nil // 不是分层配置
	}

	configName := parts[0]
	configKey := parts[1]

	// 获取配置对象
	cfg := config.GlobalConfig.Get(configName)
	if cfg == nil {
		return false, nil // 未注册的配置
	}

	// 更新配置
	configMap := map[string]string{
		configKey: value,
	}
	updateConfig := func() error {
		return config.UpdateConfigFromMap(cfg, configMap)
	}
	var err error
	if configName == "minimax" {
		err = model_setting.WithMiniMaxSettingsWriteLock(updateConfig)
	} else {
		err = updateConfig()
	}
	if err != nil {
		return true, err
	}

	// 特定配置的后处理
	if configName == "performance_setting" {
		// 同步磁盘缓存配置到 common 包
		performance_setting.UpdateAndSync()
	}
	if configName == "dashboard_config" {
		// 同步写入层聚合粒度配置到 model 包级变量，避免 model 反向依赖 dashboard_setting
		syncDashboardTrackingConfig(cfg)
	}

	return true, nil // 已处理
}

// syncDashboardTrackingConfig 从已更新的 dashboard_config 配置对象读取写入层配置，同步到 model 包级变量。
// 通过 config.ConfigToMap 反射读取，不直接依赖 dashboard_setting.DashboardConfig 类型，避免循环导入。
func syncDashboardTrackingConfig(cfg interface{}) {
	m, err := config.ConfigToMap(cfg)
	if err != nil {
		common.SysError(fmt.Sprintf("syncDashboardTrackingConfig: 解析 dashboard_config 失败: %s", err))
		return
	}
	tokens, _ := strconv.ParseBool(m["quota_data_track_tokens"])
	byModel, _ := strconv.ParseBool(m["quota_data_track_by_model"])
	byUser, _ := strconv.ParseBool(m["quota_data_track_by_user"])
	SyncQuotaDataTrackingConfig(tokens, byModel, byUser)
}
