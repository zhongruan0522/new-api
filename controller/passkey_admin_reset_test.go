package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/i18n"
	"github.com/NookMux/NookMux/model"

	"github.com/gin-gonic/gin"
)

func createAdminResetPasskeyUser(t *testing.T, id int, role int) model.User {
	t.Helper()

	user := model.User{
		Id:       id,
		Username: fmt.Sprintf("passkey-reset-user-%d", id),
		Password: "password123",
		Role:     role,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func runAdminResetPasskey(t *testing.T, adminRole int, targetId int) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("role", adminRole)
		c.Next()
	})
	router.POST("/api/user/:id/reset_passkey", AdminResetPasskey)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/user/%d/reset_passkey", targetId), nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	return recorder
}

// TestAdminResetPasskeyRejectsSameLevelAdmin 非 root admin 重置同级 admin
// 必须被拒绝。
func TestAdminResetPasskeyRejectsSameLevelAdmin(t *testing.T) {
	setupSecureVerificationTestDB(t)
	if err := i18n.Init(); err != nil {
		t.Fatalf("init i18n: %v", err)
	}

	target := createAdminResetPasskeyUser(t, 11, common.RoleAdminUser)

	recorder := runAdminResetPasskey(t, common.RoleAdminUser, target.Id)

	body := recorder.Body.String()
	if strings.Contains(body, `"success":true`) {
		t.Fatalf("non-root admin should not reset same-level admin passkey: %s", body)
	}
	if !strings.Contains(body, "无权更新同权限等级或更高权限等级的用户信息") {
		t.Fatalf("expected role hierarchy rejection, got: %s", body)
	}
}

// TestAdminResetPasskeyRejectsRootTarget 非 root admin 重置 root 用户必须被拒绝。
func TestAdminResetPasskeyRejectsRootTarget(t *testing.T) {
	setupSecureVerificationTestDB(t)
	if err := i18n.Init(); err != nil {
		t.Fatalf("init i18n: %v", err)
	}

	target := createAdminResetPasskeyUser(t, 12, common.RoleRootUser)

	recorder := runAdminResetPasskey(t, common.RoleAdminUser, target.Id)

	body := recorder.Body.String()
	if strings.Contains(body, `"success":true`) {
		t.Fatalf("non-root admin should not reset root user passkey: %s", body)
	}
	if !strings.Contains(body, "无权更新同权限等级或更高权限等级的用户信息") {
		t.Fatalf("expected role hierarchy rejection, got: %s", body)
	}
}

// TestAdminResetPasskeyRootPassesRoleCheck root 重置普通用户应通过角色检查，
// 后续因目标用户无 passkey 返回 MsgPasskeyNotBound（证明已越过角色检查）。
func TestAdminResetPasskeyRootPassesRoleCheck(t *testing.T) {
	setupSecureVerificationTestDB(t)
	if err := i18n.Init(); err != nil {
		t.Fatalf("init i18n: %v", err)
	}

	target := createAdminResetPasskeyUser(t, 13, common.RoleCommonUser)

	recorder := runAdminResetPasskey(t, common.RoleRootUser, target.Id)

	body := recorder.Body.String()
	if strings.Contains(body, "无权更新同权限等级或更高权限等级的用户信息") {
		t.Fatalf("root should pass role check, got: %s", body)
	}
	if !strings.Contains(body, "该用户尚未绑定 Passkey") {
		t.Fatalf("expected passkey-not-bound after passing role check, got: %s", body)
	}
}
