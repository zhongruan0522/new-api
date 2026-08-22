package secureverificationcontroller_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	secureverificationcontroller "github.com/NookMux/NookMux/internal/httpapi/controller/secure_verification"
	"github.com/NookMux/NookMux/internal/httpapi/controller/testsupport"
	"github.com/NookMux/NookMux/internal/httpapi/middleware"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/passkey"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestUniversalVerifyPasskeyRequiresFinishedPasskeySession(t *testing.T) {
	if err := i18n.Init(); err != nil {
		t.Fatalf("init i18n: %v", err)
	}
	testsupport.SetupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	accessToken := "root-management-token"
	user := userstore.User{
		Id:          1,
		Username:    "root",
		Password:    "password123",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Root User",
		AccessToken: &accessToken,
		Group:       "default",
	}
	if err := dbstore.DB.Create(&user).Error; err != nil {
		t.Fatalf("create root user: %v", err)
	}
	passkey := passkeystore.PasskeyCredential{
		UserID:          user.Id,
		CredentialID:    "Y3JlZGVudGlhbA==",
		PublicKey:       "cHVibGljLWtleQ==",
		AttestationType: "none",
	}
	if err := dbstore.DB.Create(&passkey).Error; err != nil {
		t.Fatalf("create passkey credential: %v", err)
	}

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.POST("/verify", middleware.UserAuth(), secureverificationcontroller.UniversalVerify)

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
	testsupport.SetupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	accessToken := "passkey-ready-token"
	user := testsupport.CreateSecureVerificationTestUser(t, 2, accessToken)
	testsupport.CreateSecureVerificationTestPasskey(t, user.Id)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(secureverificationcontroller.PasskeyReadySessionKey, time.Now().Unix())
		session.Set(secureverificationcontroller.SecureVerificationUserIDSessionKey, user.Id)
		_ = session.Save()
		c.Next()
	})
	router.POST("/verify", middleware.UserAuth(), secureverificationcontroller.UniversalVerify)

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
