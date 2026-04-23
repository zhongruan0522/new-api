package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/i18n"
	"github.com/zhongruan0522/new-api/model"

	"github.com/gin-gonic/gin"
)

// maxUserTokens 每用户最大令牌数量（硬编码）
const maxUserTokens = 1000

func GetAllTokens(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	tokens, err := model.GetAllUserTokens(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, _ := model.CountUserTokens(userId)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tokens)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchTokens(c *gin.Context) {
	userId := c.GetInt("id")
	keyword := c.Query("keyword")
	token := c.Query("token")

	pageInfo := common.GetPageQuery(c)

	tokens, total, err := model.SearchUserTokens(userId, keyword, token, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tokens)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    token,
	})
	return
}

func GetTokenStatus(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"object":          "credit_summary",
		"total_granted":   token.RemainQuota,
		"total_used":      0, // not supported currently
		"total_available": token.RemainQuota,
		"expires_at":      expiredAt * 1000,
	})
}

func GetTokenUsage(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "No Authorization header",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Invalid Bearer token",
		})
		return
	}
	tokenKey := parts[1]

	token, err := model.GetTokenByKey(strings.TrimPrefix(tokenKey, "sk-"), false)
	if err != nil {
		common.SysError("failed to get token by key: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgTokenGetInfoFailed)
		return
	}

	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    true,
		"message": "ok",
		"data": gin.H{
			"object":               "token_usage",
			"name":                 token.Name,
			"total_granted":        token.RemainQuota + token.UsedQuota,
			"total_used":           token.UsedQuota,
			"total_available":      token.RemainQuota,
			"unlimited_quota":      token.UnlimitedQuota,
			"model_limits":         token.GetModelLimitsMap(),
			"model_limits_enabled": token.ModelLimitsEnabled,
			"expires_at":           expiredAt,
		},
	})
}

func AddToken(c *gin.Context) {
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiError(c, err)
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
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口时长必须大于等于1小时",
			})
			return
		}
		if token.WindowQuota <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口额度必须大于0",
			})
			return
		}
		if token.WindowStartHour < 0 || token.WindowStartHour > 23 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口起始小时必须在0-23之间",
			})
			return
		}
	case 3: // 时段+周期限额
		token.UnlimitedQuota = false
		if token.WindowHours < 1 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口时长必须大于等于1小时",
			})
			return
		}
		if token.WindowQuota <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口额度必须大于0",
			})
			return
		}
		if token.WindowStartHour < 0 || token.WindowStartHour > 23 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口起始小时必须在0-23之间",
			})
			return
		}
		if token.CycleDays < 1 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "周期天数必须大于等于1",
			})
			return
		}
		if token.CycleQuota <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "周期总额度必须大于0",
			})
			return
		}
	default:
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的限额类型",
		})
		return
	}

	// 检查用户令牌数量是否已达上限
	count, err := model.CountUserTokens(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if int(count) >= maxUserTokens {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("已达到最大令牌数量限制 (%d)", maxUserTokens),
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
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	err := model.DeleteTokenById(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateToken(c *gin.Context) {
	userId := c.GetInt("id")
	statusOnly := c.Query("status_only")
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiError(c, err)
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
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口时长必须大于等于1小时",
			})
			return
		}
		if token.WindowQuota <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口额度必须大于0",
			})
			return
		}
		if token.WindowStartHour < 0 || token.WindowStartHour > 23 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口起始小时必须在0-23之间",
			})
			return
		}
	case 3: // 时段+周期限额
		token.UnlimitedQuota = false
		if token.WindowHours < 1 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口时长必须大于等于1小时",
			})
			return
		}
		if token.WindowQuota <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口额度必须大于0",
			})
			return
		}
		if token.WindowStartHour < 0 || token.WindowStartHour > 23 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "窗口起始小时必须在0-23之间",
			})
			return
		}
		if token.CycleDays < 1 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "周期天数必须大于等于1",
			})
			return
		}
		if token.CycleQuota <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "周期总额度必须大于0",
			})
			return
		}
	}

	cleanToken, err := model.GetTokenByIds(token.Id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
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
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanToken,
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
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
}

func ResetTokenKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	newKey, err := model.ResetTokenKey(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"key": newKey,
		},
	})
}
