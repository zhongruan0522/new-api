package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/i18n"
	"github.com/NookMux/NookMux/model"
	"github.com/NookMux/NookMux/service"

	"github.com/gin-gonic/gin"
)

// maxUserTokens 每用户最大令牌数量（硬编码）
const maxUserTokens = 1000

// buildMaskedTokenResponse 返回 key 已脱敏的 token 副本，避免列表/详情接口泄露真实密钥。
func buildMaskedTokenResponse(token *model.Token) *model.Token {
	if token == nil {
		return nil
	}
	maskedToken := *token
	maskedToken.Key = token.GetMaskedKey()
	return &maskedToken
}

func buildMaskedTokenResponses(tokens []*model.Token) []*model.Token {
	maskedTokens := make([]*model.Token, 0, len(tokens))
	for _, token := range tokens {
		maskedTokens = append(maskedTokens, buildMaskedTokenResponse(token))
	}
	return maskedTokens
}

func GetAllTokens(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	tokens, err := model.GetAllUserTokens(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.SysError("get all user tokens failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	total, _ := model.CountUserTokens(userId)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(buildMaskedTokenResponses(tokens))
	common.ApiSuccess(c, pageInfo)
}

func SearchTokens(c *gin.Context) {
	userId := c.GetInt("id")
	keyword := c.Query("keyword")
	token := c.Query("token")
	group := c.Query("group")
	all := c.Query("all") == "true" || c.Query("all") == "1"

	// status: 0 或缺省表示不过滤；1=启用 2=禁用 3=过期 4=额度耗尽。
	// 与 model.SearchUserTokens 约定一致，> 0 时走后端等值过滤。
	status := 0
	if statusStr := c.Query("status"); statusStr != "" {
		if parsed, err := strconv.Atoi(statusStr); err == nil {
			status = parsed
		}
	}

	pageInfo := common.GetPageQuery(c)

	tokens, total, err := model.SearchUserTokens(userId, keyword, token, group, status, all, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.SysError("search user tokens failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(buildMaskedTokenResponses(tokens))
	common.ApiSuccess(c, pageInfo)
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.SysError("get token by ids failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    buildMaskedTokenResponse(token),
	})
}

func GetTokenKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	userId := c.GetInt("id")
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.SysError("get token by ids failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"key": token.GetFullKey(),
		},
	})
}

func GetTokenStatus(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.SysError("get token by ids failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	snapshot := token.GetQuotaSnapshot()
	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"object":          "credit_summary",
		"total_granted":   snapshot.TotalGranted,
		"total_used":      snapshot.TotalUsed,
		"total_available": snapshot.TotalAvailable,
		"expires_at":      expiredAt * 1000,
	})
}

func GetTokenUsage(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgTokenNoAuthHeader),
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgTokenInvalidBearer),
		})
		return
	}
	token, err := getTokenForFeedback(c)
	if err != nil {
		common.SysError("failed to get token by key: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgTokenGetInfoFailed)
		return
	}
	snapshot := token.GetQuotaSnapshot()

	// 对齐 quota_type 兼容逻辑：quota_type=0 且非无限额度视为永久限额(1)。
	// 与 GetQuotaSnapshot / TokenAuthReadOnly 中间件保持一致。
	effectiveQuotaType := token.QuotaType
	if effectiveQuotaType == 0 && !token.UnlimitedQuota {
		effectiveQuotaType = 1
	}

	// 计算下一次窗口/周期重置时间，避免前端重复实现后端锚点逻辑。
	// 未启用对应限额类型时保持为 0。
	windowNextResetTime := int64(0)
	cycleNextResetTime := int64(0)
	if effectiveQuotaType == 2 || effectiveQuotaType == 3 {
		if _, windowEnd := token.GetCurrentWindow(); windowEnd > 0 {
			windowNextResetTime = windowEnd
		}
	}
	if effectiveQuotaType == 3 {
		if _, cycleEnd := token.GetCurrentCycle(); cycleEnd > 0 {
			cycleNextResetTime = cycleEnd
		}
	}

	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    true,
		"message": i18n.T(c, i18n.MsgTokenAuthorized),
		"data": gin.H{
			"object":               "token_usage",
			"name":                 token.Name,
			"total_granted":        snapshot.TotalGranted,
			"total_used":           snapshot.TotalUsed,
			"total_available":      snapshot.TotalAvailable,
			"unlimited_quota":      token.UnlimitedQuota,
			"model_limits":         token.GetModelLimitsMap(),
			"model_limits_enabled": token.ModelLimitsEnabled,
			"expires_at":           expiredAt,
			// 以下字段为密钥用量查询页面所需，供前端按 KEY 类型展示额度与周期信息。
			"quota_type":             effectiveQuotaType,
			"created_time":           token.CreatedTime,
			"accessed_time":          token.AccessedTime,
			"window_hours":           token.WindowHours,
			"window_quota":           token.WindowQuota,
			"window_used_quota":      token.WindowUsedQuota,
			"window_start_hour":      token.WindowStartHour,
			"window_start_time":      token.WindowStartTime,
			"window_next_reset_time": windowNextResetTime,
			"cycle_days":             token.CycleDays,
			"cycle_quota":            token.CycleQuota,
			"cycle_used_quota":       token.CycleUsedQuota,
			"cycle_start_time":       token.CycleStartTime,
			"cycle_next_reset_time":  cycleNextResetTime,
		},
	})
}

func AddToken(c *gin.Context) {
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}

	// 根据 quota_type 验证额度参数
	quotaType := token.QuotaType
	// 兼容旧逻辑：如果 quota_type 未设置但 unlimited_quota 有值
	if quotaType == 0 && !token.UnlimitedQuota {
		quotaType = 1
	}

	switch quotaType {
	case 0: // 无限额度
		token.UnlimitedQuota = true
	case 1: // 永久限额
		token.UnlimitedQuota = false
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	case 2: // 时段限额
		token.UnlimitedQuota = false
		if token.WindowHours < 1 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowHoursMin)
			return
		}
		if token.WindowQuota <= 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowQuotaPositive)
			return
		}
		if token.WindowStartHour < 0 || token.WindowStartHour > 23 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowStartRange)
			return
		}
	case 3: // 时段+周期限额
		token.UnlimitedQuota = false
		if token.WindowHours < 1 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowHoursMin)
			return
		}
		if token.WindowQuota <= 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowQuotaPositive)
			return
		}
		if token.WindowStartHour < 0 || token.WindowStartHour > 23 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowStartRange)
			return
		}
		if token.CycleDays < 1 {
			common.ApiErrorI18n(c, i18n.MsgTokenCycleDaysMin)
			return
		}
		if token.CycleQuota <= 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenCycleQuotaPositive)
			return
		}
	default:
		common.ApiErrorI18n(c, i18n.MsgTokenInvalidQuotaType)
		return
	}

	// 检查用户令牌数量是否已达上限
	count, err := model.CountUserTokens(c.GetInt("id"))
	if err != nil {
		common.SysError("count user tokens failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	if int(count) >= maxUserTokens {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgTokenMaxLimitReached, map[string]any{"Max": maxUserTokens}),
		})
		return
	}
	key, err := common.GenerateKey()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTokenGenerateFailed)
		common.SysLog("failed to generate token key: " + err.Error())
		return
	}

	now := common.GetTimestamp()
	cleanToken := model.Token{
		UserId:             c.GetInt("id"),
		Name:               token.Name,
		Key:                key,
		CreatedTime:        now,
		AccessedTime:       now,
		ExpiredTime:        token.ExpiredTime,
		RemainQuota:        token.RemainQuota,
		UnlimitedQuota:     token.UnlimitedQuota,
		ModelLimitsEnabled: token.ModelLimitsEnabled,
		ModelLimits:        token.ModelLimits,
		AllowIps:           token.AllowIps,
		Group:              token.Group,
		CrossGroupRetry:    token.CrossGroupRetry,
		QuotaType:          quotaType,
		WindowHours:        token.WindowHours,
		WindowQuota:        token.WindowQuota,
		WindowStartHour:    token.WindowStartHour,
		CycleDays:          token.CycleDays,
		CycleQuota:         token.CycleQuota,
		WindowUsedQuota:    0,
		WindowStartTime:    0,
		CycleUsedQuota:     0,
		CycleStartTime:     0,
	}
	err = cleanToken.Insert()
	if err != nil {
		common.SysError("insert token failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModuleToken, model.AuditActionCreate, "新增令牌: "+cleanToken.Name, nil, map[string]interface{}{"name": cleanToken.Name, "user_id": cleanToken.UserId})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"key": key,
		},
	})
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	err := model.DeleteTokenById(id, userId)
	if err != nil {
		common.SysError("delete token by id failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModuleToken, model.AuditActionDelete, "删除令牌", nil, map[string]interface{}{"id": id})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func UpdateToken(c *gin.Context) {
	userId := c.GetInt("id")
	statusOnly := c.Query("status_only")
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}

	// 根据 quota_type 验证额度参数
	quotaType := token.QuotaType
	// 兼容旧逻辑：如果 quota_type 未设置但 unlimited_quota 有值
	if quotaType == 0 && !token.UnlimitedQuota {
		quotaType = 1
	}

	switch quotaType {
	case 0: // 无限额度
		token.UnlimitedQuota = true
	case 1: // 永久限额
		token.UnlimitedQuota = false
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	case 2: // 时段限额
		token.UnlimitedQuota = false
		if token.WindowHours < 1 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowHoursMin)
			return
		}
		if token.WindowQuota <= 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowQuotaPositive)
			return
		}
		if token.WindowStartHour < 0 || token.WindowStartHour > 23 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowStartRange)
			return
		}
	case 3: // 时段+周期限额
		token.UnlimitedQuota = false
		if token.WindowHours < 1 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowHoursMin)
			return
		}
		if token.WindowQuota <= 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowQuotaPositive)
			return
		}
		if token.WindowStartHour < 0 || token.WindowStartHour > 23 {
			common.ApiErrorI18n(c, i18n.MsgTokenWindowStartRange)
			return
		}
		if token.CycleDays < 1 {
			common.ApiErrorI18n(c, i18n.MsgTokenCycleDaysMin)
			return
		}
		if token.CycleQuota <= 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenCycleQuotaPositive)
			return
		}
	}

	cleanToken, err := model.GetTokenByIds(token.Id, userId)
	if err != nil {
		common.SysError("get token by ids failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	// 保存更新前快照用于审计差异对比
	originToken := *cleanToken
	if token.Status == common.TokenStatusEnabled {
		if cleanToken.Status == common.TokenStatusExpired && cleanToken.ExpiredTime <= common.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			common.ApiErrorI18n(c, i18n.MsgTokenExpiredCannotEnable)
			return
		}
		if cleanToken.Status == common.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			common.ApiErrorI18n(c, i18n.MsgTokenExhaustedCannotEable)
			return
		}
	}
	if statusOnly != "" {
		cleanToken.Status = token.Status
	} else {
		// If you add more fields, please also update token.Update()
		oldQuotaType := cleanToken.QuotaType
		oldWindowHours := cleanToken.WindowHours
		oldWindowStartHour := cleanToken.WindowStartHour
		oldCycleDays := cleanToken.CycleDays
		oldCycleQuota := cleanToken.CycleQuota

		cleanToken.Name = token.Name
		cleanToken.ExpiredTime = token.ExpiredTime
		cleanToken.RemainQuota = token.RemainQuota
		cleanToken.UnlimitedQuota = token.UnlimitedQuota
		cleanToken.ModelLimitsEnabled = token.ModelLimitsEnabled
		cleanToken.ModelLimits = token.ModelLimits
		cleanToken.AllowIps = token.AllowIps
		cleanToken.Group = token.Group
		cleanToken.CrossGroupRetry = token.CrossGroupRetry
		cleanToken.QuotaType = quotaType
		cleanToken.WindowHours = token.WindowHours
		cleanToken.WindowQuota = token.WindowQuota
		cleanToken.WindowStartHour = token.WindowStartHour
		cleanToken.CycleDays = token.CycleDays
		cleanToken.CycleQuota = token.CycleQuota

		// 如果 quota_type 或窗口参数变化，重置运行时状态
		if oldQuotaType != quotaType ||
			oldWindowHours != token.WindowHours ||
			oldWindowStartHour != token.WindowStartHour {
			cleanToken.WindowUsedQuota = 0
			cleanToken.WindowStartTime = 0
		}
		if oldQuotaType != quotaType || (quotaType == 3 && (oldCycleDays != token.CycleDays || oldCycleQuota != token.CycleQuota)) {
			cleanToken.CycleUsedQuota = 0
			cleanToken.CycleStartTime = 0
		}
	}
	err = cleanToken.Update()
	if err != nil {
		common.SysError("update token failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModuleToken, model.AuditActionUpdate, "修改令牌: "+cleanToken.Name, originToken, cleanToken)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    buildMaskedTokenResponse(cleanToken),
	})
}

type TokenBatch struct {
	Ids []int `json:"ids"`
}

func DeleteTokenBatch(c *gin.Context) {
	tokenBatch := TokenBatch{}
	if err := c.ShouldBindJSON(&tokenBatch); err != nil || len(tokenBatch.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	userId := c.GetInt("id")
	count, err := model.BatchDeleteTokens(tokenBatch.Ids, userId)
	if err != nil {
		common.SysError("batch delete tokens failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModuleToken, model.AuditActionDelete, "批量删除令牌", nil, map[string]interface{}{"ids": tokenBatch.Ids})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
}

func GetTokenKeysBatch(c *gin.Context) {
	tokenBatch := TokenBatch{}
	if err := c.ShouldBindJSON(&tokenBatch); err != nil || len(tokenBatch.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	userId := c.GetInt("id")
	keys := make(map[int]string, len(tokenBatch.Ids))
	for _, id := range tokenBatch.Ids {
		token, err := model.GetTokenByIds(id, userId)
		if err != nil {
			common.SysError("get token by ids failed: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
			return
		}
		keys[id] = token.GetFullKey()
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"keys": keys,
		},
	})
}

func ResetTokenKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	userId := c.GetInt("id")
	newKey, err := model.ResetTokenKey(id, userId)
	if err != nil {
		common.SysError("reset token key failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	service.RecordAudit(c, model.AuditModuleToken, model.AuditActionUpdate, "重置令牌密钥", nil, map[string]interface{}{"id": id})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"key": newKey,
		},
	})
}
