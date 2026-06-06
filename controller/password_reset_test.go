package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
)

func TestSendPasswordResetEmailHidesUnknownEmail(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/reset_password", SendPasswordResetEmail)

	req := httptest.NewRequest(http.MethodGet, "/api/reset_password?email=missing@example.com", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.Success {
		t.Fatalf("expected success response, got: %s", recorder.Body.String())
	}
	if body.Message != "" {
		t.Fatalf("message = %q, want empty", body.Message)
	}
}

func TestSendPasswordResetEmailReturnsSuccessWhenDeliveryFails(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	user := model.User{
		Id:          1,
		Username:    "reset-user",
		Password:    "password123",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Reset User",
		Email:       "reset-user@example.com",
		Group:       "default",
		AffCode:     "reset-aff-1",
	}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	router := gin.New()
	router.GET("/api/reset_password", SendPasswordResetEmail)

	req := httptest.NewRequest(http.MethodGet, "/api/reset_password?email=reset-user@example.com", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.Success {
		t.Fatalf("expected success response despite email delivery failure, got: %s", recorder.Body.String())
	}
	if body.Message != "" {
		t.Fatalf("message = %q, want empty", body.Message)
	}
}
