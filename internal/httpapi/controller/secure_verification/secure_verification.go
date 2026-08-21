package secureverificationcontroller

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/system"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/i18n"
	passkeysvc "github.com/NookMux/NookMux/internal/infra/passkey"
	"github.com/NookMux/NookMux/internal/infra/security"
	"github.com/NookMux/NookMux/internal/store/log"
	"github.com/NookMux/NookMux/internal/store/passkey"
	"github.com/NookMux/NookMux/internal/store/twofa"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const (
	// SecureVerificationSessionKey means the user has fully passed secure verification.
	SecureVerificationSessionKey = "secure_verified_at"
	// SecureVerificationUserIDSessionKey 绑定安全验证通过的用户，避免验证态被其他身份复用
	SecureVerificationUserIDSessionKey = "secure_verified_user_id"
	SecureVerificationMethodSessionKey = "secure_verified_method"
	SecureVerificationMethod2FA        = "2fa"
	SecureVerificationMethodPasskey    = "passkey"
	// PasskeyReadySessionKey means WebAuthn finished and /api/verify can finalize step-up verification.
	PasskeyReadySessionKey = "secure_passkey_ready_at"
	// SecureVerificationTimeout 验证有效期（秒）
	SecureVerificationTimeout = 300 // 5分钟
	// PasskeyReadyTimeout passkey ready 标记有效期（秒）
	PasskeyReadyTimeout = 60
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
			"message": i18n.T(c, i18n.MsgNotLoggedIn),
		})
		return
	}

	var req UniversalVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}

	// 获取用户信息
	user := &userstore.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeyUserInfoFailed)
		return
	}

	if user.Status != common.UserStatusEnabled {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeyUserDisabled)
		return
	}

	// 检查用户的验证方式
	twoFA, _ := twofastore.GetTwoFAByUserId(userId)
	has2FA := twoFA != nil && twoFA.IsEnabled

	passkey, passkeyErr := passkeystore.GetPasskeyByUserID(userId)
	hasPasskey := passkeyErr == nil && passkey != nil

	if !has2FA && !hasPasskey {
		httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyNoMethod)
		return
	}

	// 根据验证方式进行验证
	var verified bool
	var verifyMethod string
	var err error

	switch req.Method {
	case "2fa":
		if !has2FA {
			httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyTwoFANotEnabled)
			return
		}
		if req.Code == "" {
			httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyCodeRequired)
			return
		}
		verified = validateTwoFactorAuth(twoFA, req.Code)
		verifyMethod = "2FA"

	case "passkey":
		if !hasPasskey {
			httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyPasskeyNotEnabled)
			return
		}
		verified, err = consumePasskeyReady(c, userId)
		if err != nil {
			httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyInvalidPasskeyState)
			return
		}
		if !verified {
			httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyPasskeyRequired)
			return
		}
		verifyMethod = "Passkey"

	default:
		httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyUnsupportedMethod, map[string]any{"Method": req.Method})
		return
	}

	if !verified {
		httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyFailed)
		return
	}

	// 验证成功，在 session 中记录时间戳并绑定当前用户
	now, err := setSecureVerificationSession(c, userId, req.Method)
	if err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeySaveVerifyFailed)
		return
	}

	// 记录日志
	userstore.RecordLog(userId, logstore.LogTypeSystem, fmt.Sprintf("通用安全验证成功 (验证方式: %s)", verifyMethod))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": i18n.T(c, i18n.MsgSecureVerifySuccess),
		"data": gin.H{
			"verified":   true,
			"expires_at": now + SecureVerificationTimeout,
		},
	})
}

// PasskeyVerifyAndMarkReadySession writes a short-lived marker consumed by /api/verify.
func PasskeyVerifyAndMarkReadySession(c *gin.Context, userId int) (int64, error) {
	session := sessions.Default(c)
	now := time.Now().Unix()
	session.Set(PasskeyReadySessionKey, now)
	session.Set(SecureVerificationUserIDSessionKey, userId)
	session.Delete(SecureVerificationSessionKey)
	session.Delete(SecureVerificationMethodSessionKey)
	return now, session.Save()
}

func setSecureVerificationSession(c *gin.Context, userId int, method string) (int64, error) {
	session := sessions.Default(c)
	now := time.Now().Unix()
	session.Delete(PasskeyReadySessionKey)
	session.Set(SecureVerificationSessionKey, now)
	session.Set(SecureVerificationUserIDSessionKey, userId)
	session.Set(SecureVerificationMethodSessionKey, method)
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

// HasSecureVerificationMethodForUser 报告指定用户是否已通过指定方式（2fa/passkey）的安全验证。
func HasSecureVerificationMethodForUser(c *gin.Context, userId int, method string) bool {
	ok, _ := hasSecureVerificationForUser(c, userId)
	if !ok {
		return false
	}
	session := sessions.Default(c)
	verifiedMethod, ok := session.Get(SecureVerificationMethodSessionKey).(string)
	return ok && verifiedMethod == method
}

// validateTwoFactorAuth 统一的2FA验证函数
func validateTwoFactorAuth(twoFA *twofastore.TwoFA, code string) bool {
	// 尝试验证TOTP
	if cleanCode, err := security.ValidateNumericCode(code); err == nil {
		if isValid, _ := twoFA.ValidateTOTPAndUpdateUsage(cleanCode); isValid {
			return true
		}
	}

	// 尝试验证备用码
	if isValid, err := twoFA.ValidateBackupCodeAndUpdateUsage(code); err == nil && isValid {
		return true
	}

	return false
}

func consumePasskeyReady(c *gin.Context, userId int) (bool, error) {
	session := sessions.Default(c)
	readyAtRaw := session.Get(PasskeyReadySessionKey)
	if readyAtRaw == nil {
		return false, nil
	}

	readyAt, ok := readyAtRaw.(int64)
	if !ok {
		session.Delete(PasskeyReadySessionKey)
		_ = session.Save()
		return false, fmt.Errorf("%s", i18n.T(c, i18n.MsgSecureVerifyInvalidPasskeyState))
	}

	verifiedUserId, ok := sessionInt(session.Get(SecureVerificationUserIDSessionKey))
	if !ok || verifiedUserId != userId {
		session.Delete(PasskeyReadySessionKey)
		_ = session.Save()
		return false, nil
	}

	session.Delete(PasskeyReadySessionKey)
	if err := session.Save(); err != nil {
		return false, err
	}
	if time.Now().Unix()-readyAt >= PasskeyReadyTimeout {
		return false, nil
	}
	return true, nil
}

// PasskeyVerifyForSecure 用于安全验证的 Passkey 验证流程
// 整合了 begin 和 finish 流程
func PasskeyVerifyForSecure(c *gin.Context) {
	if !system.GetPasskeySettings().Enabled {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeyLoginDisabled)
		return
	}

	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgNotLoggedIn),
		})
		return
	}

	user := &userstore.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeyUserInfoFailed)
		return
	}

	if user.Status != common.UserStatusEnabled {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeyUserDisabled)
		return
	}

	credential, err := passkeystore.GetPasskeyByUserID(userId)
	if err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeyNotBound)
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyFailed)
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(user, credential)
	sessionData, err := passkeysvc.PopSessionData(c, passkeysvc.VerifySessionKey)
	if err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeyInvalidSession)
		return
	}

	_, err = wa.FinishLogin(waUser, *sessionData, c.Request)
	if err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgSecureVerifyFailed)
		return
	}

	// 更新凭证的最后使用时间
	usedAt := time.Now()
	credential.LastUsedAt = &usedAt
	if err := passkeystore.UpsertPasskeyCredential(credential); err != nil {
		common.SysError("upsert passkey credential failed: " + err.Error())
		httpapi.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	// 验证成功，设置 session
	_, err = PasskeyVerifyAndMarkReadySession(c, userId)
	if err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgPasskeySaveVerifyFailed)
		return
	}

	// 记录日志
	userstore.RecordLog(userId, logstore.LogTypeSystem, "Passkey 安全验证成功")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": i18n.T(c, i18n.MsgPasskeyVerifyOK),
	})
}
