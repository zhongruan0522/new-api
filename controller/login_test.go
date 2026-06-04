package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/i18n"
	"github.com/zhongruan0522/new-api/model"
)

func TestLoginHidesDatabaseErrors(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)
	if err := i18n.Init(); err != nil {
		t.Fatalf("init i18n: %v", err)
	}

	sqlDB, err := model.DB.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	router := gin.New()
	router.POST("/api/user/login", Login)

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"username":"alice","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "登录服务暂时不可用，请联系管理员") {
		t.Fatalf("expected generic login unavailable message, got: %s", body)
	}
	if strings.Contains(body, "sql: database is closed") || strings.Contains(body, "用户名或密码错误") {
		t.Fatalf("response leaked backend or auth details: %s", body)
	}
}
