package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

func setupUserAccessTokenTestDB(t *testing.T) {
	t.Helper()

	oldDB := DB
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
	})
}

func createAccessTokenTestUser(t *testing.T, id int, role int, accessToken *string) User {
	t.Helper()

	user := User{
		Id:          id,
		Username:    fmt.Sprintf("user-%d", id),
		Password:    "password123",
		Role:        role,
		Status:      common.UserStatusEnabled,
		DisplayName: fmt.Sprintf("User %d", id),
		AccessToken: accessToken,
		Group:       "default",
		AffCode:     fmt.Sprintf("access-token-aff-%d", id),
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func TestNormalizeAccessToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		want          string
	}{
		{name: "empty", authorization: "", want: ""},
		{name: "spaces", authorization: "   ", want: ""},
		{name: "bare bearer", authorization: "Bearer", want: ""},
		{name: "empty bearer", authorization: "Bearer ", want: ""},
		{name: "empty bearer lowercase", authorization: "bearer ", want: ""},
		{name: "bearer with token", authorization: "Bearer token-123", want: "token-123"},
		{name: "bearer lowercase with spaces", authorization: "bearer     token-123", want: "token-123"},
		{name: "malformed bearer", authorization: "Bearer token-123 extra", want: ""},
		{name: "raw token", authorization: "token-123", want: "token-123"},
		{name: "trim raw token", authorization: "  token-123  ", want: "token-123"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeAccessToken(test.authorization); got != test.want {
				t.Fatalf("normalizeAccessToken(%q) = %q, want %q", test.authorization, got, test.want)
			}
		})
	}
}

func TestValidateAccessTokenRejectsEmptyBearerWhenEmptyTokenExists(t *testing.T) {
	setupUserAccessTokenTestDB(t)
	emptyAccessToken := ""
	createAccessTokenTestUser(t, 1, common.RoleRootUser, &emptyAccessToken)

	for _, authorization := range []string{"", "   ", "Bearer", "Bearer ", "bearer "} {
		t.Run(fmt.Sprintf("authorization_%q", authorization), func(t *testing.T) {
			got, err := ValidateAccessToken(authorization)
			if err != ErrTokenInvalid {
				t.Fatalf("ValidateAccessToken(%q) err = %v, want %v", authorization, err, ErrTokenInvalid)
			}
			if got != nil {
				t.Fatalf("ValidateAccessToken(%q) returned user %d, want nil", authorization, got.Id)
			}
		})
	}
}

func TestValidateAccessTokenAcceptsNonEmptyBearerToken(t *testing.T) {
	setupUserAccessTokenTestDB(t)
	accessToken := "valid-management-token"
	user := createAccessTokenTestUser(t, 1, common.RoleRootUser, &accessToken)

	for _, authorization := range []string{"Bearer " + accessToken, "bearer     " + accessToken, accessToken} {
		t.Run(fmt.Sprintf("authorization_%q", authorization), func(t *testing.T) {
			got, err := ValidateAccessToken(authorization)
			if err != nil {
				t.Fatalf("ValidateAccessToken(%q) err = %v", authorization, err)
			}
			if got == nil {
				t.Fatalf("ValidateAccessToken(%q) returned nil", authorization)
			}
			if got.Id != user.Id {
				t.Fatalf("ValidateAccessToken(%q) returned user %d, want %d", authorization, got.Id, user.Id)
			}
		})
	}
}

func TestUserAccessTokenIsNotSerialized(t *testing.T) {
	accessToken := "sensitive-management-token"
	user := User{
		Id:          1,
		Username:    "root",
		Password:    "password123",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Root User",
		AccessToken: &accessToken,
		Group:       "default",
		AffCode:     "serialize-aff-1",
	}

	data, err := common.Marshal(user)
	if err != nil {
		t.Fatalf("marshal user: %v", err)
	}
	body := string(data)
	if strings.Contains(body, "access_token") || strings.Contains(body, accessToken) {
		t.Fatalf("serialized user leaked access token: %s", body)
	}
}

func TestUserPublicQueriesOmitAccessToken(t *testing.T) {
	setupUserAccessTokenTestDB(t)
	rootAccessToken := "root-management-token"
	adminAccessToken := "admin-management-token"
	createAccessTokenTestUser(t, 1, common.RoleRootUser, &rootAccessToken)
	createAccessTokenTestUser(t, 2, common.RoleAdminUser, &adminAccessToken)

	users, total, err := GetAllUsers(&common.PageInfo{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("GetAllUsers error: %v", err)
	}
	if total != 2 {
		t.Fatalf("GetAllUsers total = %d, want 2", total)
	}
	for _, user := range users {
		if user.AccessToken != nil {
			t.Fatalf("GetAllUsers returned access token for user %d", user.Id)
		}
	}

	searchUsers, total, err := SearchUsers("user", "", "", 0, 10)
	if err != nil {
		t.Fatalf("SearchUsers error: %v", err)
	}
	if total != 2 {
		t.Fatalf("SearchUsers total = %d, want 2", total)
	}
	for _, user := range searchUsers {
		if user.AccessToken != nil {
			t.Fatalf("SearchUsers returned access token for user %d", user.Id)
		}
	}

	publicUser, err := GetUserById(1, false)
	if err != nil {
		t.Fatalf("GetUserById(false) error: %v", err)
	}
	if publicUser.AccessToken != nil {
		t.Fatalf("GetUserById(false) returned access token")
	}

	fullUser, err := GetUserById(1, true)
	if err != nil {
		t.Fatalf("GetUserById(true) error: %v", err)
	}
	if fullUser.AccessToken == nil || *fullUser.AccessToken != rootAccessToken {
		t.Fatalf("GetUserById(true) access token = %v, want %q", fullUser.AccessToken, rootAccessToken)
	}
}

func TestCleanupEmptyAccessTokensConvertsEmptyStringToNull(t *testing.T) {
	setupUserAccessTokenTestDB(t)
	emptyAccessToken := ""
	validAccessToken := "valid-management-token"
	createAccessTokenTestUser(t, 1, common.RoleRootUser, &emptyAccessToken)
	createAccessTokenTestUser(t, 2, common.RoleCommonUser, &validAccessToken)

	if err := cleanupEmptyAccessTokens(); err != nil {
		t.Fatalf("cleanupEmptyAccessTokens() error: %v", err)
	}

	var root User
	if err := DB.First(&root, 1).Error; err != nil {
		t.Fatalf("get root user: %v", err)
	}
	if root.AccessToken != nil {
		t.Fatalf("root access token = %q, want nil", *root.AccessToken)
	}

	var commonUser User
	if err := DB.First(&commonUser, 2).Error; err != nil {
		t.Fatalf("get common user: %v", err)
	}
	if commonUser.AccessToken == nil || *commonUser.AccessToken != validAccessToken {
		t.Fatalf("common user access token = %v, want %q", commonUser.AccessToken, validAccessToken)
	}
}
