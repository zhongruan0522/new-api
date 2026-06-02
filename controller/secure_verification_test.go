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
	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/middleware"
	"github.com/zhongruan0522/new-api/model"
	"gorm.io/gorm"
)

func setupSecureVerificationTestDB(t *testing.T) {
	t.Helper()

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.PasskeyCredential{}, &model.Log{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = oldDB
		model.LOG_DB = oldLogDB
	})
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
