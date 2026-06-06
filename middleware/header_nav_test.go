package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
)

func withHeaderNavModules(t *testing.T, raw string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	previous, hadPrevious := common.OptionMap["HeaderNavModules"]
	common.OptionMap["HeaderNavModules"] = raw
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadPrevious {
			common.OptionMap["HeaderNavModules"] = previous
			return
		}
		delete(common.OptionMap, "HeaderNavModules")
	})
}

func performHeaderNavRequest(t *testing.T, handler gin.HandlerFunc, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("header-nav-test"))))
	router.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("username", "tester")
		session.Set("role", common.RoleCommonUser)
		session.Set("id", 1)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false})
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/test", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	var cookies []*http.Cookie
	if authenticated {
		loginRecorder := httptest.NewRecorder()
		loginRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
		router.ServeHTTP(loginRecorder, loginRequest)
		if loginRecorder.Code != http.StatusNoContent {
			t.Fatalf("login status = %d, body = %s", loginRecorder.Code, loginRecorder.Body.String())
		}
		cookies = loginRecorder.Result().Cookies()
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if authenticated {
		request.Header.Set("New-Api-User", "1")
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestHeaderNavModuleAuthAllowsDefaultPublicAccess(t *testing.T) {
	withHeaderNavModules(t, "")

	recorder := performHeaderNavRequest(t, HeaderNavModuleAuth("pricing"), false)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHeaderNavModuleAuthRejectsDisabledModule(t *testing.T) {
	withHeaderNavModules(t, `{"pricing":{"enabled":false,"requireAuth":false}}`)

	recorder := performHeaderNavRequest(t, HeaderNavModuleAuth("pricing"), false)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHeaderNavModuleAuthRequiresLogin(t *testing.T) {
	withHeaderNavModules(t, `{"pricing":{"enabled":true,"requireAuth":true}}`)

	recorder := performHeaderNavRequest(t, HeaderNavModuleAuth("pricing"), false)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHeaderNavModuleAuthAllowsLoggedInUser(t *testing.T) {
	withHeaderNavModules(t, `{"pricing":{"enabled":true,"requireAuth":true}}`)

	recorder := performHeaderNavRequest(t, HeaderNavModuleAuth("pricing"), true)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHeaderNavModuleAuthRejectsLegacyDisabledModule(t *testing.T) {
	withHeaderNavModules(t, `{"pricing":false}`)

	recorder := performHeaderNavRequest(t, HeaderNavModuleAuth("pricing"), false)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", recorder.Code, recorder.Body.String())
	}
}
