package model

import (
	"fmt"
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
			if got := ValidateAccessToken(authorization); got != nil {
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
			got := ValidateAccessToken(authorization)
			if got == nil {
				t.Fatalf("ValidateAccessToken(%q) returned nil", authorization)
			}
			if got.Id != user.Id {
				t.Fatalf("ValidateAccessToken(%q) returned user %d, want %d", authorization, got.Id, user.Id)
			}
		})
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
