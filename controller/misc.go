package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/i18n"
	"github.com/zhongruan0522/new-api/middleware"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/setting"
	"github.com/zhongruan0522/new-api/setting/console_setting"
	"github.com/zhongruan0522/new-api/setting/dashboard_setting"
	"github.com/zhongruan0522/new-api/setting/operation_setting"
	"github.com/zhongruan0522/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func TestStatus(c *gin.Context) {
	err := model.PingDB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgMiscDBConnectionFailed),
		})
		return
	}
	// 获取HTTP统计信息
	httpStats := middleware.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Server is running",
		"http_stats": httpStats,
	})
	return
}

func GetStatus(c *gin.Context) {

	// 面板启用开关的单一权威源是 dashboard_config（#112）。
	// console_setting 仍用于读取内容数据（ApiInfo/Announcements/FAQ）。
	dc := dashboard_setting.GetDashboardConfig()
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	passkeySetting := system_setting.GetPasskeySettings()
	legalSetting := system_setting.GetLegalSettings()

	data := gin.H{
		"email_verification":          common.EmailVerificationEnabled,
		"github_oauth":                common.GitHubOAuthEnabled,
		"github_client_id":            common.GitHubClientId,
		"linuxdo_oauth":               common.LinuxDOOAuthEnabled,
		"linuxdo_client_id":           common.LinuxDOClientId,
		"linuxdo_minimum_trust_level": common.LinuxDOMinimumTrustLevel,
		"system_name":                 common.SystemName,
		"logo":                        common.Logo,
		"footer_html":                 common.Footer,
		"server_address":              system_setting.ServerAddress,
		"turnstile_check":             common.TurnstileCheckEnabled,
		"turnstile_site_key":          common.TurnstileSiteKey,
		"docs_link":                   operation_setting.GetGeneralSetting().DocsLink,
		"quota_per_unit":              common.QuotaPerUnit,
		"enable_batch_update":         common.BatchUpdateEnabled,
		"enable_data_export":          common.DataExportEnabled,
		"data_export_interval":        common.DataExportInterval,
		"data_export_default_time":    common.DataExportDefaultTime,
		"default_collapse_sidebar":    common.DefaultCollapseSidebar,
		"default_use_auto_group":      setting.DefaultUseAutoGroup,

		"price":             operation_setting.Price,
		"stripe_unit_price": setting.StripeUnitPrice,

		// 面板启用开关（单一配置源：dashboard_config）
		"api_info_enabled":      dc.ApiInfoEnabled,
		"uptime_kuma_enabled":   dc.UptimeKumaEnabled,
		"announcements_enabled": dc.AnnouncementsEnabled,
		"faq_enabled":           dc.FAQEnabled,

		// 模块管理配置
		"HeaderNavModules":          common.OptionMap["HeaderNavModules"],
		"SidebarModulesAdmin":       common.OptionMap["SidebarModulesAdmin"],
		"passkey_login":             passkeySetting.Enabled,
		"passkey_display_name":      passkeySetting.RPDisplayName,
		"passkey_rp_id":             passkeySetting.RPID,
		"passkey_origins":           passkeySetting.Origins,
		"passkey_allow_insecure":    passkeySetting.AllowInsecureOrigin,
		"passkey_user_verification": passkeySetting.UserVerification,
		"passkey_attachment":        passkeySetting.AttachmentPreference,
		"setup":                     constant.Setup,
		"user_agreement_enabled":    legalSetting.UserAgreement != "",
		"privacy_policy_enabled":    legalSetting.PrivacyPolicy != "",
		"checkin_enabled":           operation_setting.GetCheckinSetting().Enabled,
		"version":                   common.Version,
		"_qn":                       "new-api",
	}

	// 根据启用状态注入可选内容（开关读 dashboard_config，内容数据读 console_setting）
	if dc.ApiInfoEnabled {
		data["api_info"] = console_setting.GetApiInfo()
	}
	if dc.AnnouncementsEnabled {
		data["announcements"] = console_setting.GetAnnouncements()
	}
	if dc.FAQEnabled {
		data["faq"] = console_setting.GetFAQ()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
	return
}

func GetNotice(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["Notice"],
	})
	return
}

func GetAbout(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["About"],
	})
	return
}

func GetUserAgreement(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    system_setting.GetLegalSettings().UserAgreement,
	})
	return
}

func GetPrivacyPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    system_setting.GetLegalSettings().PrivacyPolicy,
	})
	return
}

func GetMidjourney(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["Midjourney"],
	})
	return
}

func GetHomePageContent(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["HomePageContent"],
	})
	return
}

func SendEmailVerification(c *gin.Context) {
	email := c.Query("email")
	if err := common.Validate.Var(email, "required,email"); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		common.ApiErrorI18n(c, i18n.MsgSettingEmailInvalid)
		return
	}
	localPart := parts[0]
	domainPart := parts[1]
	if common.EmailDomainRestrictionEnabled {
		allowed := false
		for _, domain := range common.EmailDomainWhitelist {
			if domainPart == domain {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "The administrator has enabled the email domain name whitelist, and your email address is not allowed due to special symbols or it's not in the whitelist.",
			})
			return
		}
	}
	if common.EmailAliasRestrictionEnabled {
		containsSpecialSymbols := strings.Contains(localPart, "+") || strings.Contains(localPart, ".")
		if containsSpecialSymbols {
			common.ApiErrorI18n(c, i18n.MsgMiscEmailAliasRejected)
			return
		}
	}

	if model.IsEmailAlreadyTaken(email) {
		common.ApiErrorI18n(c, i18n.MsgMiscEmailTaken)
		return
	}
	code := common.GenerateVerificationCode(6)
	common.RegisterVerificationCodeWithKey(email, code, common.EmailVerificationPurpose)
	subject := fmt.Sprintf("%s邮箱验证邮件", common.SystemName)
	content := fmt.Sprintf("<p>您好，你正在进行%s邮箱验证。</p>"+
		"<p>您的验证码为: <strong>%s</strong></p>"+
		"<p>验证码 %d 分钟内有效，如果不是本人操作，请忽略。</p>", common.SystemName, code, common.VerificationValidMinutes)
	err := common.SendEmail(subject, email, content)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to send email verification to %s: %v", email, err))
		common.ApiErrorI18n(c, i18n.MsgMiscEmailSendFailed)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func SendPasswordResetEmail(c *gin.Context) {
	email := c.Query("email")
	if err := common.Validate.Var(email, "required,email"); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if model.IsEmailAlreadyTaken(email) {
		code := common.GenerateVerificationCode(0)
		common.RegisterVerificationCodeWithKey(email, code, common.PasswordResetPurpose)
		link := fmt.Sprintf("%s/user/reset?email=%s&token=%s", system_setting.ServerAddress, email, code)
		subject := fmt.Sprintf("%s密码重置", common.SystemName)
		content := fmt.Sprintf("<p>您好，你正在进行%s密码重置。</p>"+
			"<p>点击 <a href='%s'>此处</a> 进行密码重置。</p>"+
			"<p>如果链接无法点击，请尝试点击下面的链接或将其复制到浏览器中打开：<br> %s </p>"+
			"<p>重置链接 %d 分钟内有效，如果不是本人操作，请忽略。</p>", common.SystemName, link, link, common.VerificationValidMinutes)
		if err := common.SendEmail(subject, email, content); err != nil {
			common.SysError(fmt.Sprintf("failed to send password reset email to %s: %v", email, err))
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

type PasswordResetRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func ResetPassword(c *gin.Context) {
	var req PasswordResetRequest
	err := common.DecodeJson(c.Request.Body, &req)
	if err != nil || req.Email == "" || req.Token == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if !common.VerifyCodeWithKey(req.Email, req.Token, common.PasswordResetPurpose) {
		common.ApiErrorI18n(c, i18n.MsgMiscPasswordResetLinkInvalid)
		return
	}
	password := common.GenerateVerificationCode(12)
	err = model.ResetUserPasswordByEmail(req.Email, password)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.DeleteKey(req.Email, common.PasswordResetPurpose)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    password,
	})
	return
}

// GetUsageLogFieldsVisible 公开接口：返回当前用户角色下使用日志详情弹窗的字段可见性配置。
// 普通用户和管理员都可访问，区别在于 isAdmin 由中间件注入。
func GetUsageLogFieldsVisible(c *gin.Context) {
	role := c.GetInt("role")
	isAdmin := role >= common.RoleAdminUser

	// 总开关关闭时，返回空字段列表，前端据此隐藏详情按钮
	if !console_setting.IsUsageLogDetailsEnabled(isAdmin) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"enabled": false,
				"fields":  []string{},
			},
		})
		return
	}

	fieldsMap, err := console_setting.GetUsageLogFieldsVisible()
	if err != nil {
		common.SysError("failed to parse usage_log_fields setting: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgMiscUsageLogFieldsParseFailed)
		return
	}

	// 返回当前角色可见的字段 key 列表
	visibleFields := make([]string, 0, len(fieldsMap))
	for key, cfg := range fieldsMap {
		if isAdmin && cfg.Admin {
			visibleFields = append(visibleFields, key)
		} else if !isAdmin && cfg.User {
			visibleFields = append(visibleFields, key)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled": true,
			"fields":  visibleFields,
		},
	})
	return
}
