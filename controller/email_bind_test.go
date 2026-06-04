package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
)

func TestEmailBindUsesPostJsonBody(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	user := createSecureVerificationTestUser(t, 1, "email-bind-token")
	email := fmt.Sprintf("email-bind-%d@example.com", user.Id)
	common.RegisterVerificationCodeWithKey(email, "123456", common.EmailVerificationPurpose)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", user.Id)
		_ = session.Save()
		c.Next()
	})
	router.POST("/api/oauth/email/bind", EmailBind)

	body := fmt.Sprintf(`{"email":%q,"code":"123456"}`, email)
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/email/bind?email=attacker@example.com&code=bad", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var updated model.User
	if err := model.DB.First(&updated, user.Id).Error; err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if updated.Email != email {
		t.Fatalf("updated email = %q, want %q", updated.Email, email)
	}
}
