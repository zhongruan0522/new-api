package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/middleware"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/setting/system_setting"
	"gorm.io/gorm"
)

func setupSecureVerificationTestDB(t *testing.T) {
	t.Helper()

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.PasskeyCredential{}, &model.TwoFA{}, &model.Log{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func createSecureVerificationTestUser(t *testing.T, id int, accessToken string) model.User {
	t.Helper()

	user := model.User{
		Id:          id,
		Username:    fmt.Sprintf("secure-user-%d", id),
		Password:    "password123",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		DisplayName: fmt.Sprintf("Secure User %d", id),
		AccessToken: &accessToken,
		Group:       "default",
		AffCode:     fmt.Sprintf("secure-aff-%d", id),
	}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func createSecureVerificationTestPasskey(t *testing.T, userId int) {
	t.Helper()

	passkey := model.PasskeyCredential{
		UserID:          userId,
		CredentialID:    fmt.Sprintf("Y3JlZGVudGlhbC0%d", userId),
		PublicKey:       "cHVibGljLWtleQ==",
		AttestationType: "none",
	}
	if err := model.DB.Create(&passkey).Error; err != nil {
		t.Fatalf("create passkey credential: %v", err)
	}
}

func createSecureVerificationTestTwoFA(t *testing.T, userId int) {
	t.Helper()

	twoFA := model.TwoFA{
		UserId:    userId,
		Secret:    "secret",
		IsEnabled: true,
	}
	if err := model.DB.Create(&twoFA).Error; err != nil {
		t.Fatalf("create twofa: %v", err)
	}
}

func secureVerificationSessionMiddleware(userId int, method string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", userId)
		session.Set(SecureVerificationSessionKey, time.Now().Unix())
		session.Set(SecureVerificationUserIDSessionKey, userId)
		session.Set(secureVerificationMethodSessionKey, method)
		_ = session.Save()
		c.Next()
	}
}

func TestUniversalVerifyPasskeyRequiresFinishedPasskeySession(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	accessToken := "root-management-token"
	user := model.User{
		Id:          1,
		Username:    "root",
		Password:    "password123",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Root User",
		AccessToken: &accessToken,
		Group:       "default",
	}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatalf("create root user: %v", err)
	}
	passkey := model.PasskeyCredential{
		UserID:          user.Id,
		CredentialID:    "Y3JlZGVudGlhbA==",
		PublicKey:       "cHVibGljLWtleQ==",
		AttestationType: "none",
	}
	if err := model.DB.Create(&passkey).Error; err != nil {
		t.Fatalf("create passkey credential: %v", err)
	}

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.POST("/verify", middleware.UserAuth(), UniversalVerify)

	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(`{"method":"passkey"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("New-Api-User", "1")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if strings.Contains(body, `"success":true`) {
		t.Fatalf("direct passkey verification succeeded without WebAuthn finish: %s", body)
	}
	if !strings.Contains(body, "请先完成 Passkey 验证") {
		t.Fatalf("expected passkey finish requirement, got: %s", body)
	}
}

func TestUniversalVerifyPasskeyConsumesReadyMarker(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	accessToken := "passkey-ready-token"
	user := createSecureVerificationTestUser(t, 2, accessToken)
	createSecureVerificationTestPasskey(t, user.Id)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(PasskeyReadySessionKey, time.Now().Unix())
		session.Set(SecureVerificationUserIDSessionKey, user.Id)
		_ = session.Save()
		c.Next()
	})
	router.POST("/verify", middleware.UserAuth(), UniversalVerify)

	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(`{"method":"passkey"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("New-Api-User", fmt.Sprintf("%d", user.Id))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if !strings.Contains(body, `"success":true`) {
		t.Fatalf("passkey verify did not consume ready marker: %s", body)
	}
}

func TestPasskeyDeleteRequiresTwoFAMethodWhenTwoFAEnabled(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	user := createSecureVerificationTestUser(t, 3, "delete-passkey-token")
	createSecureVerificationTestPasskey(t, user.Id)
	createSecureVerificationTestTwoFA(t, user.Id)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(secureVerificationSessionMiddleware(user.Id, secureVerificationMethodPasskey))
	router.DELETE("/passkey", PasskeyDelete)

	req := httptest.NewRequest(http.MethodDelete, "/passkey", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if strings.Contains(body, `"success":true`) {
		t.Fatalf("passkey delete succeeded with passkey verification despite enabled 2FA: %s", body)
	}
	if _, err := model.GetPasskeyByUserID(user.Id); err != nil {
		t.Fatalf("passkey was deleted despite wrong verification method: %v", err)
	}
}

func TestPasskeyDeleteAllowsTwoFAMethodWhenTwoFAEnabled(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	user := createSecureVerificationTestUser(t, 4, "delete-passkey-2fa-token")
	createSecureVerificationTestPasskey(t, user.Id)
	createSecureVerificationTestTwoFA(t, user.Id)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(secureVerificationSessionMiddleware(user.Id, secureVerificationMethod2FA))
	router.DELETE("/passkey", PasskeyDelete)

	req := httptest.NewRequest(http.MethodDelete, "/passkey", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if !strings.Contains(body, `"success":true`) {
		t.Fatalf("passkey delete did not accept 2FA verification: %s", body)
	}
	if _, err := model.GetPasskeyByUserID(user.Id); err == nil {
		t.Fatalf("passkey still exists after verified delete")
	}
}

func TestPasskeyRegisterBeginRequiresTwoFAMethodWhenTwoFAEnabled(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	settings := system_setting.GetPasskeySettings()
	oldEnabled := settings.Enabled
	settings.Enabled = true
	t.Cleanup(func() {
		settings.Enabled = oldEnabled
	})

	user := createSecureVerificationTestUser(t, 5, "register-passkey-token")
	createSecureVerificationTestTwoFA(t, user.Id)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(secureVerificationSessionMiddleware(user.Id, secureVerificationMethodPasskey))
	router.POST("/passkey/register/begin", PasskeyRegisterBegin)

	req := httptest.NewRequest(http.MethodPost, "/passkey/register/begin", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if strings.Contains(body, `"success":true`) {
		t.Fatalf("passkey registration began with passkey verification despite enabled 2FA: %s", body)
	}
	if !strings.Contains(body, "请先完成对应的安全验证") {
		t.Fatalf("expected method-specific verification error, got: %s", body)
	}
}
