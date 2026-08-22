package passkeycontroller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/config/system"
	secureverificationcontroller "github.com/NookMux/NookMux/internal/httpapi/controller/secure_verification"
	"github.com/NookMux/NookMux/internal/httpapi/controller/testsupport"
	"github.com/NookMux/NookMux/internal/store/passkey"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestPasskeyDeleteRequiresTwoFAMethodWhenTwoFAEnabled(t *testing.T) {
	testsupport.SetupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	user := testsupport.CreateSecureVerificationTestUser(t, 3, "delete-passkey-token")
	testsupport.CreateSecureVerificationTestPasskey(t, user.Id)
	testsupport.CreateSecureVerificationTestTwoFA(t, user.Id)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(testsupport.SecureVerificationSessionMiddleware(user.Id, secureverificationcontroller.SecureVerificationMethodPasskey))
	router.DELETE("/passkey", PasskeyDelete)

	req := httptest.NewRequest(http.MethodDelete, "/passkey", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if strings.Contains(body, `"success":true`) {
		t.Fatalf("passkey delete succeeded with passkey verification despite enabled 2FA: %s", body)
	}
	if _, err := passkeystore.GetPasskeyByUserID(user.Id); err != nil {
		t.Fatalf("passkey was deleted despite wrong verification method: %v", err)
	}
}

func TestPasskeyDeleteAllowsTwoFAMethodWhenTwoFAEnabled(t *testing.T) {
	testsupport.SetupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	user := testsupport.CreateSecureVerificationTestUser(t, 4, "delete-passkey-2fa-token")
	testsupport.CreateSecureVerificationTestPasskey(t, user.Id)
	testsupport.CreateSecureVerificationTestTwoFA(t, user.Id)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(testsupport.SecureVerificationSessionMiddleware(user.Id, secureverificationcontroller.SecureVerificationMethod2FA))
	router.DELETE("/passkey", PasskeyDelete)

	req := httptest.NewRequest(http.MethodDelete, "/passkey", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if !strings.Contains(body, `"success":true`) {
		t.Fatalf("passkey delete did not accept 2FA verification: %s", body)
	}
	if _, err := passkeystore.GetPasskeyByUserID(user.Id); err == nil {
		t.Fatalf("passkey still exists after verified delete")
	}
}

func TestPasskeyRegisterBeginRequiresTwoFAMethodWhenTwoFAEnabled(t *testing.T) {
	testsupport.SetupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	settings := system.GetPasskeySettings()
	oldEnabled := settings.Enabled
	settings.Enabled = true
	t.Cleanup(func() {
		settings.Enabled = oldEnabled
	})

	user := testsupport.CreateSecureVerificationTestUser(t, 5, "register-passkey-token")
	testsupport.CreateSecureVerificationTestTwoFA(t, user.Id)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(testsupport.SecureVerificationSessionMiddleware(user.Id, secureverificationcontroller.SecureVerificationMethodPasskey))
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
