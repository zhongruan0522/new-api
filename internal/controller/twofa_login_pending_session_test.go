package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/i18n"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// twofaLoginPendingSessionMiddleware 构造一个已写入 pending 2FA 字段的 session。
// setAt <= 0 表示不写入时间戳（模拟修复前创建的旧 session）。
func twofaLoginPendingSessionMiddleware(userId int, setAt int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("pending_username", "pending-user")
		session.Set("pending_user_id", userId)
		if setAt > 0 {
			session.Set("pending_2fa_set_at", setAt)
		}
		_ = session.Save()
		c.Next()
	}
}

func runVerify2FALoginWithPendingSession(t *testing.T, userId int, setAt int64) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(twofaLoginPendingSessionMiddleware(userId, setAt))
	router.POST("/api/user/login/2fa", Verify2FALogin)

	req := httptest.NewRequest(http.MethodPost, "/api/user/login/2fa", strings.NewReader(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	return recorder
}

// TestVerify2FALoginRejectsExpiredPendingSession 过期的 pending session
// （setAt = now - SecureVerificationTimeout - 1）必须被拒绝，且 pending 字段被清除。
func TestVerify2FALoginRejectsExpiredPendingSession(t *testing.T) {
	if err := i18n.Init(); err != nil {
		t.Fatalf("init i18n: %v", err)
	}

	recorder := runVerify2FALoginWithPendingSession(t, 1, time.Now().Unix()-SecureVerificationTimeout-1)

	body := recorder.Body.String()
	if !strings.Contains(body, "会话已过期，请重新登录") {
		t.Fatalf("expected session expired rejection, got: %s", body)
	}
}

// TestVerify2FALoginRejectsPendingSessionWithoutTimestamp 无时间戳的旧 session
// 必须被拒绝（fail-closed）。
func TestVerify2FALoginRejectsPendingSessionWithoutTimestamp(t *testing.T) {
	if err := i18n.Init(); err != nil {
		t.Fatalf("init i18n: %v", err)
	}

	recorder := runVerify2FALoginWithPendingSession(t, 1, 0)

	body := recorder.Body.String()
	if !strings.Contains(body, "会话已过期，请重新登录") {
		t.Fatalf("expected session expired rejection for legacy session, got: %s", body)
	}
}

// TestVerify2FALoginAcceptsFreshPendingSession 新鲜的 pending session 应通过时效
// 检查（后续因用户不存在而失败是正常路径，断言不是 session expired 错误即可，
// 因此本用例无需数据库）。
func TestVerify2FALoginAcceptsFreshPendingSession(t *testing.T) {
	setupSecureVerificationTestDB(t)
	if err := i18n.Init(); err != nil {
		t.Fatalf("init i18n: %v", err)
	}

	recorder := runVerify2FALoginWithPendingSession(t, 404, time.Now().Unix())

	body := recorder.Body.String()
	if strings.Contains(body, "会话已过期，请重新登录") {
		t.Fatalf("fresh pending session should pass timeout check, got: %s", body)
	}
	// 用户不存在是时效检查之后的正常失败路径，证明已越过时效校验。
	if !strings.Contains(body, "用户不存在") {
		t.Fatalf("expected user-not-exists path after timeout check, got: %s", body)
	}
}
