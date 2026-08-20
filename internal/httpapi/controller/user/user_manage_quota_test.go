package usercontroller

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/httpapi/controller/testsupport"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func createManageQuotaTestUser(t *testing.T, id int, quota int) {
	t.Helper()

	accessToken := fmt.Sprintf("manage-quota-token-%d", id)
	user := userstore.User{
		Id:          id,
		Username:    fmt.Sprintf("mq-user-%d", id),
		Password:    "password123",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		DisplayName: fmt.Sprintf("Manage Quota User %d", id),
		AccessToken: &accessToken,
		Group:       "default",
		AffCode:     fmt.Sprintf("mq-aff-%d", id),
		Quota:       quota,
	}
	if err := dbstore.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func requestManageQuota(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/manage", strings.NewReader(body))
	c.Set("role", common.RoleRootUser)

	ManageUser(c)

	return recorder
}

func getManageQuotaTestUserQuota(t *testing.T, id int) int {
	t.Helper()

	var user userstore.User
	if err := dbstore.DB.First(&user, id).Error; err != nil {
		t.Fatalf("get user: %v", err)
	}
	return user.Quota
}

func TestManageUserAddQuotaAllowsNegativeOriginQuota(t *testing.T) {
	testsupport.SetupSecureVerificationTestDB(t)
	createManageQuotaTestUser(t, 1001, -100000000)

	recorder := requestManageQuota(t, `{"id":1001,"action":"add_quota","mode":"add","value":100}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"success":true`) {
		t.Fatalf("response = %s, want success", recorder.Body.String())
	}
	if got := getManageQuotaTestUserQuota(t, 1001); got != -99999900 {
		t.Fatalf("quota = %d, want -99999900", got)
	}
}

func TestManageUserOverrideQuotaFromNegativeOrigin(t *testing.T) {
	testsupport.SetupSecureVerificationTestDB(t)
	createManageQuotaTestUser(t, 1002, -100000000)

	recorder := requestManageQuota(t, `{"id":1002,"action":"add_quota","mode":"override","value":100}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"success":true`) {
		t.Fatalf("response = %s, want success", recorder.Body.String())
	}
	if got := getManageQuotaTestUserQuota(t, 1002); got != 100 {
		t.Fatalf("quota = %d, want 100", got)
	}
}

func TestUpdateUserPreservesQuotaWhenQuotaOmitted(t *testing.T) {
	testsupport.SetupSecureVerificationTestDB(t)
	createManageQuotaTestUser(t, 1003, -100000000)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/user/", strings.NewReader(`{
		"id":1003,
		"username":"mq-user-1003",
		"display_name":"Updated Name",
		"group":"default",
		"remark":"updated"
	}`))
	c.Set("role", common.RoleRootUser)

	UpdateUser(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"success":true`) {
		t.Fatalf("response = %s, want success", recorder.Body.String())
	}
	if got := getManageQuotaTestUserQuota(t, 1003); got != -100000000 {
		t.Fatalf("quota = %d, want -100000000", got)
	}
}
