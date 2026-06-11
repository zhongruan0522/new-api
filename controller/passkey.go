package controller

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	passkeysvc "github.com/zhongruan0522/new-api/service/passkey"
	"github.com/zhongruan0522/new-api/setting/system_setting"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	webauthnlib "github.com/go-webauthn/webauthn/webauthn"
)

type PasskeyRegisterRequest struct {
	DeviceName string `json:"device_name"`
}

const (
	passkeyRegistrationDeviceNameSessionKey = "passkey_registration_device_name"
	maxPasskeyDeviceNameLength              = 255
)

type PasskeyListItem struct {
	ID             int        `json:"id"`
	DeviceName     string     `json:"device_name"`
	Attachment     string     `json:"attachment"`
	BackupEligible bool       `json:"backup_eligible"`
	BackupState    bool       `json:"backup_state"`
	LastUsedAt     *time.Time `json:"last_used_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func normalizePasskeyDeviceName(deviceName string) string {
	deviceName = strings.TrimSpace(deviceName)
	runes := []rune(deviceName)
	if len(runes) > maxPasskeyDeviceNameLength {
		deviceName = string(runes[:maxPasskeyDeviceNameLength])
	}
	return deviceName
}

func PasskeyRegisterBegin(c *gin.Context) {
	if !system_setting.GetPasskeySettings().Enabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未启用 Passkey 登录",
		})
		return
	}

	user, err := getSessionUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	if !requirePasskeyRegistrationVerification(c, user.Id) {
		return
	}

	settings := system_setting.GetPasskeySettings()
	maxPasskeys := settings.MaxPasskeysPerUser
	if maxPasskeys < 1 {
		maxPasskeys = 1
	}

	count, err := model.CountPasskeysByUserID(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if count >= int64(maxPasskeys) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("已达到 Passkey 数量上限 (%d 个)", maxPasskeys),
		})
		return
	}

	credentials, err := model.GetPasskeysByUserID(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(user, nil)
	var options []webauthnlib.RegistrationOption
	if len(credentials) > 0 {
		excludeList := make([]protocol.CredentialDescriptor, 0, len(credentials))
		for _, cred := range credentials {
			webAuthnCredential := cred.ToWebAuthnCredential()
			excludeList = append(excludeList, webAuthnCredential.Descriptor())
		}
		options = append(options, webauthnlib.WithExclusions(excludeList))
	}

	creation, sessionData, err := wa.BeginRegistration(waUser, options...)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req PasskeyRegisterRequest
	session := sessions.Default(c)
	if err := c.ShouldBindJSON(&req); err == nil {
		deviceName := normalizePasskeyDeviceName(req.DeviceName)
		if deviceName != "" {
			session.Set(passkeyRegistrationDeviceNameSessionKey, deviceName)
		} else {
			session.Delete(passkeyRegistrationDeviceNameSessionKey)
		}
	} else {
		session.Delete(passkeyRegistrationDeviceNameSessionKey)
	}

	if err := passkeysvc.SaveSessionData(c, passkeysvc.RegistrationSessionKey, sessionData); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"options": creation,
		},
	})
}

func PasskeyRegisterFinish(c *gin.Context) {
	if !system_setting.GetPasskeySettings().Enabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未启用 Passkey 登录",
		})
		return
	}

	user, err := getSessionUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	if !requirePasskeyRegistrationVerification(c, user.Id) {
		return
	}

	settings := system_setting.GetPasskeySettings()
	maxPasskeys := settings.MaxPasskeysPerUser
	if maxPasskeys < 1 {
		maxPasskeys = 1
	}

	count, err := model.CountPasskeysByUserID(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if count >= int64(maxPasskeys) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("已达到 Passkey 数量上限 (%d 个)", maxPasskeys),
		})
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	sessionData, err := passkeysvc.PopSessionData(c, passkeysvc.RegistrationSessionKey)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(user, nil)
	credential, err := wa.FinishRegistration(waUser, *sessionData, c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	passkeyCredential := model.NewPasskeyCredentialFromWebAuthn(user.Id, credential)
	if passkeyCredential == nil {
		common.ApiErrorMsg(c, "无法创建 Passkey 凭证")
		return
	}

	session := sessions.Default(c)
	if deviceName, ok := session.Get(passkeyRegistrationDeviceNameSessionKey).(string); ok {
		passkeyCredential.DeviceName = normalizePasskeyDeviceName(deviceName)
	}
	session.Delete(passkeyRegistrationDeviceNameSessionKey)
	_ = session.Save()

	if err := model.CreatePasskeyCredential(passkeyCredential); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Passkey 注册成功",
	})
}

func PasskeyDelete(c *gin.Context) {
	user, err := getSessionUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	if !requirePasskeyDeleteVerification(c, user.Id) {
		return
	}

	idStr := c.Param("id")
	if idStr == "" {
		if err := model.DeletePasskeyByUserID(user.Id); err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			common.ApiErrorMsg(c, "无效的 Passkey ID")
			return
		}

		credential, err := model.GetPasskeyByID(id)
		if err != nil {
			common.ApiError(c, err)
			return
		}

		if credential.UserID != user.Id {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "无权删除此 Passkey",
			})
			return
		}

		if err := model.DeletePasskeyByID(id); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Passkey 已解绑",
	})
}

func PasskeyStatus(c *gin.Context) {
	user, err := getSessionUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	credentials, err := model.GetPasskeysByUserID(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	settings := system_setting.GetPasskeySettings()
	maxPasskeys := settings.MaxPasskeysPerUser
	if maxPasskeys < 1 {
		maxPasskeys = 1
	}

	passkeys := make([]PasskeyListItem, 0, len(credentials))
	for _, cred := range credentials {
		passkeys = append(passkeys, PasskeyListItem{
			ID:             cred.ID,
			DeviceName:     cred.DeviceName,
			Attachment:     cred.Attachment,
			BackupEligible: cred.BackupEligible,
			BackupState:    cred.BackupState,
			LastUsedAt:     cred.LastUsedAt,
			CreatedAt:      cred.CreatedAt,
		})
	}

	data := gin.H{
		"enabled":      len(credentials) > 0,
		"passkeys":     passkeys,
		"count":        len(credentials),
		"max_passkeys": maxPasskeys,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}

func PasskeyLoginBegin(c *gin.Context) {
	if !system_setting.GetPasskeySettings().Enabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未启用 Passkey 登录",
		})
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	assertion, sessionData, err := wa.BeginDiscoverableLogin()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := passkeysvc.SaveSessionData(c, passkeysvc.LoginSessionKey, sessionData); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"options": assertion,
		},
	})
}

func PasskeyLoginFinish(c *gin.Context) {
	if !system_setting.GetPasskeySettings().Enabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未启用 Passkey 登录",
		})
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	sessionData, err := passkeysvc.PopSessionData(c, passkeysvc.LoginSessionKey)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	handler := func(rawID, userHandle []byte) (webauthnlib.User, error) {
		// 首先通过凭证ID查找用户
		credential, err := model.GetPasskeyByCredentialID(rawID)
		if err != nil {
			return nil, fmt.Errorf("未找到 Passkey 凭证: %w", err)
		}

		// 通过凭证获取用户
		user := &model.User{Id: credential.UserID}
		if err := user.FillUserById(); err != nil {
			return nil, fmt.Errorf("用户信息获取失败: %w", err)
		}

		if user.Status != common.UserStatusEnabled {
			return nil, errors.New("该用户已被禁用")
		}

		if len(userHandle) > 0 {
			userID, parseErr := strconv.Atoi(string(userHandle))
			if parseErr != nil {
				// 记录异常但继续验证，因为某些客户端可能使用非数字格式
				common.SysLog(fmt.Sprintf("PasskeyLogin: userHandle parse error for credential, length: %d", len(userHandle)))
			} else if userID != user.Id {
				return nil, errors.New("用户句柄与凭证不匹配")
			}
		}

		return passkeysvc.NewWebAuthnUser(user, credential), nil
	}

	waUser, credential, err := wa.FinishPasskeyLogin(handler, *sessionData, c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	userWrapper, ok := waUser.(*passkeysvc.WebAuthnUser)
	if !ok {
		common.ApiErrorMsg(c, "Passkey 登录状态异常")
		return
	}

	modelUser := userWrapper.ModelUser()
	if modelUser == nil {
		common.ApiErrorMsg(c, "Passkey 登录状态异常")
		return
	}

	if modelUser.Status != common.UserStatusEnabled {
		common.ApiErrorMsg(c, "该用户已被禁用")
		return
	}

	// 更新凭证信息
	updatedCredential := model.NewPasskeyCredentialFromWebAuthn(modelUser.Id, credential)
	if updatedCredential == nil {
		common.ApiErrorMsg(c, "Passkey 凭证更新失败")
		return
	}

	existingCred, err := model.GetPasskeyByCredentialID(credential.ID)
	if err == nil && existingCred != nil {
		updatedCredential.ID = existingCred.ID
		updatedCredential.DeviceName = existingCred.DeviceName
	}

	now := time.Now()
	updatedCredential.LastUsedAt = &now
	if updatedCredential.ID > 0 {
		if err := model.UpdatePasskeyCredential(updatedCredential); err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		if err := model.CreatePasskeyCredential(updatedCredential); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	setupLogin(modelUser, c)
	return
}

func AdminResetPasskey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}

	user := &model.User{Id: id}
	if err := user.FillUserById(); err != nil {
		common.ApiError(c, err)
		return
	}

	count, err := model.CountPasskeysByUserID(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if count == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该用户尚未绑定 Passkey",
		})
		return
	}

	if err := model.DeletePasskeyByUserID(user.Id); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Passkey 已重置",
	})
}

func PasskeyVerifyBegin(c *gin.Context) {
	if !system_setting.GetPasskeySettings().Enabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未启用 Passkey 登录",
		})
		return
	}

	user, err := getSessionUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	credentials, err := model.GetPasskeysByUserID(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(credentials) == 0 {
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

	waUser := passkeysvc.NewWebAuthnUser(user, nil)
	assertion, sessionData, err := wa.BeginLogin(waUser)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := passkeysvc.SaveSessionData(c, passkeysvc.VerifySessionKey, sessionData); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"options": assertion,
		},
	})
}

func PasskeyVerifyFinish(c *gin.Context) {
	if !system_setting.GetPasskeySettings().Enabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未启用 Passkey 登录",
		})
		return
	}

	user, err := getSessionUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	credentials, err := model.GetPasskeysByUserID(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(credentials) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该用户尚未绑定 Passkey",
		})
		return
	}

	sessionData, err := passkeysvc.PopSessionData(c, passkeysvc.VerifySessionKey)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(user, nil)
	validatedCred, err := wa.FinishLogin(waUser, *sessionData, c.Request)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 查找被验证的凭证并更新最后使用时间
	for _, cred := range credentials {
		credBytes, _ := base64.StdEncoding.DecodeString(cred.CredentialID)
		if string(credBytes) == string(validatedCred.ID) {
			now := time.Now()
			cred.LastUsedAt = &now
			cred.SignCount = validatedCred.Authenticator.SignCount
			if err := model.UpdatePasskeyCredential(cred); err != nil {
				common.ApiError(c, err)
				return
			}
			break
		}
	}

	_, err = PasskeyVerifyAndMarkReadySession(c, user.Id)
	if err != nil {
		common.ApiError(c, fmt.Errorf("保存验证状态失败: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Passkey 验证成功",
	})
}

func getSessionUser(c *gin.Context) (*model.User, error) {
	session := sessions.Default(c)
	idRaw := session.Get("id")
	if idRaw == nil {
		return nil, errors.New("未登录")
	}
	id, ok := idRaw.(int)
	if !ok {
		return nil, errors.New("无效的会话信息")
	}
	user := &model.User{Id: id}
	if err := user.FillUserById(); err != nil {
		return nil, err
	}
	if user.Status != common.UserStatusEnabled {
		return nil, errors.New("该用户已被禁用")
	}
	return user, nil
}

func requirePasskeyRegistrationVerification(c *gin.Context, userID int) bool {
	twoFA, err := model.GetTwoFAByUserId(userID)
	if err != nil {
		common.ApiError(c, err)
		return false
	}
	if twoFA == nil || !twoFA.IsEnabled {
		return true
	}
	return requireSecureVerificationMethod(c, userID, secureVerificationMethod2FA)
}

func requirePasskeyDeleteVerification(c *gin.Context, userID int) bool {
	twoFA, err := model.GetTwoFAByUserId(userID)
	if err != nil {
		common.ApiError(c, err)
		return false
	}
	if twoFA != nil && twoFA.IsEnabled {
		return requireSecureVerificationMethod(c, userID, secureVerificationMethod2FA)
	}

	_, err = model.GetPasskeyByUserID(userID)
	if err != nil {
		if errors.Is(err, model.ErrPasskeyNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "该用户尚未绑定 Passkey",
			})
			return false
		}
		common.ApiError(c, err)
		return false
	}

	return requireSecureVerificationMethod(c, userID, secureVerificationMethodPasskey)
}

type PasskeyUpdateRequest struct {
	DeviceName string `json:"device_name"`
}

func PasskeyUpdate(c *gin.Context) {
	user, err := getSessionUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiErrorMsg(c, "无效的 Passkey ID")
		return
	}

	credential, err := model.GetPasskeyByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if credential.UserID != user.Id {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "无权修改此 Passkey",
		})
		return
	}

	var req PasskeyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求参数错误")
		return
	}

	credential.DeviceName = normalizePasskeyDeviceName(req.DeviceName)
	if err := model.UpdatePasskeyCredential(credential); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Passkey 已更新",
	})
}

func requireSecureVerificationMethod(c *gin.Context, userID int, method string) bool {
	if !hasSecureVerificationMethodForUser(c, userID, method) {
		common.ApiErrorMsg(c, "请先完成对应的安全验证")
		return false
	}
	return true
}
