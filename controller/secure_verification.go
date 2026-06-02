package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	passkeysvc "github.com/zhongruan0522/new-api/service/passkey"
	"github.com/zhongruan0522/new-api/setting/system_setting"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	// SecureVerificationSessionKey 安全验证的 session key
	SecureVerificationSessionKey = "secure_verified_at"
	// SecureVerificationUserIDSessionKey 绑定安全验证通过的用户，避免验证态被其他身份复用
	SecureVerificationUserIDSessionKey = "secure_verified_user_id"
	// SecureVerificationTimeout 验证有效期（秒）
	SecureVerificationTimeout = 300 // 5分钟
)

type UniversalVerifyRequest struct {
	Method string `json:"method"` // "2fa" 或 "passkey"
	Code   string `json:"code,omitempty"`
}

type VerificationStatusResponse struct {
	Verified  bool  `json:"verified"`
	ExpiresAt int64 `json:"expires_at,omitempty"`
}

// UniversalVerify 通用验证接口
// 支持 2FA 和 Passkey 验证，验证成功后在 session 中记录时间戳
func UniversalVerify(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req UniversalVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, fmt.Errorf("参数错误: %v", err))
		return
	}

	// 获取用户信息
	user := &model.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		common.ApiError(c, fmt.Errorf("获取用户信息失败: %v", err))
		return
	}

	if user.Status != common.UserStatusEnabled {
		common.ApiError(c, fmt.Errorf("该用户已被禁用"))
		return
	}

	// 检查用户的验证方式
	twoFA, _ := model.GetTwoFAByUserId(userId)
	has2FA := twoFA != nil && twoFA.IsEnabled

	passkey, passkeyErr := model.GetPasskeyByUserID(userId)
	hasPasskey := passkeyErr == nil && passkey != nil

	if !has2FA && !hasPasskey {
		common.ApiError(c, fmt.Errorf("用户未启用2FA或Passkey"))
		return
	}

	// 根据验证方式进行验证
	var verified bool
	var verifyMethod string

	switch req.Method {
	case "2fa":
		if !has2FA {
			common.ApiError(c, fmt.Errorf("用户未启用2FA"))
			return
		}
		if req.Code == "" {
			common.ApiError(c, fmt.Errorf("验证码不能为空"))
			return
		}
		verified = validateTwoFactorAuth(twoFA, req.Code)
		verifyMethod = "2FA"

	case "passkey":
		if !hasPasskey {
			common.ApiError(c, fmt.Errorf("用户未启用Passkey"))
			return
		}
		if ok, _ := hasSecureVerificationForUser(c, userId); !ok {
			common.ApiError(c, fmt.Errorf("请先完成 Passkey 验证"))
			return
		}
		verifyMethod = "Passkey"
		verified = true

	default:
		common.ApiError(c, fmt.Errorf("不支持的验证方式: %s", req.Method))
		return
	}

	if !verified {
		common.ApiError(c, fmt.Errorf("验证失败，请检查验证码"))
		return
	}

	// 验证成功，在 session 中记录时间戳并绑定当前用户
	now, err := setSecureVerificationSession(c, userId)
	if err != nil {
		common.ApiError(c, fmt.Errorf("保存验证状态失败: %v", err))
		return
	}

	// 记录日志
	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("通用安全验证成功 (验证方式: %s)", verifyMethod))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证成功",
		"data": gin.H{
			"verified":   true,
			"expires_at": now + SecureVerificationTimeout,
		},
	})
}

// PasskeyVerifyAndSetSession Passkey 验证完成后设置 session
// 这是一个辅助函数，供 PasskeyVerifyFinish 调用
func PasskeyVerifyAndSetSession(c *gin.Context, userId int) (int64, error) {
	return setSecureVerificationSession(c, userId)
}

func setSecureVerificationSession(c *gin.Context, userId int) (int64, error) {
	session := sessions.Default(c)
	now := time.Now().Unix()
	session.Set(SecureVerificationSessionKey, now)
	session.Set(SecureVerificationUserIDSessionKey, userId)
	return now, session.Save()
}

func sessionInt(raw interface{}) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	default:
		return 0, false
	}
}

func hasSecureVerificationForUser(c *gin.Context, userId int) (bool, int64) {
	session := sessions.Default(c)
	verifiedAt, ok := session.Get(SecureVerificationSessionKey).(int64)
	if !ok {
		return false, 0
	}
	verifiedUserId, ok := sessionInt(session.Get(SecureVerificationUserIDSessionKey))
	if !ok || verifiedUserId != userId {
		return false, 0
	}
	if time.Now().Unix()-verifiedAt >= SecureVerificationTimeout {
		return false, 0
	}
	return true, verifiedAt
}

// PasskeyVerifyForSecure 用于安全验证的 Passkey 验证流程
// 整合了 begin 和 finish 流程
func PasskeyVerifyForSecure(c *gin.Context) {
	if !system_setting.GetPasskeySettings().Enabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未启用 Passkey 登录",
		})
		return
	}

	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	user := &model.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		common.ApiError(c, fmt.Errorf("获取用户信息失败: %v", err))
		return
	}

	if user.Status != common.UserStatusEnabled {
		common.ApiError(c, fmt.Errorf("该用户已被禁用"))
		return
	}

	credential, err := model.GetPasskeyByUserID(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该用户尚未绑定 Passkey",
		})
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(user, credential)
	sessionData, err := passkeysvc.PopSessionData(c, passkeysvc.VerifySessionKey)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	_, err = wa.FinishLogin(waUser, *sessionData, c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 更新凭证的最后使用时间
	usedAt := time.Now()
	credential.LastUsedAt = &usedAt
	if err := model.UpsertPasskeyCredential(credential); err != nil {
		common.ApiError(c, err)
		return
	}

	// 验证成功，设置 session
	now, err := PasskeyVerifyAndSetSession(c, userId)
	if err != nil {
		common.ApiError(c, fmt.Errorf("保存验证状态失败: %v", err))
		return
	}

	// 记录日志
	model.RecordLog(userId, model.LogTypeSystem, "Passkey 安全验证成功")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Passkey 验证成功",
		"data": gin.H{
			"verified":   true,
			"expires_at": now + SecureVerificationTimeout,
		},
	})
}
