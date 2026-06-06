package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/oauth"
)

type oauthBindTestProvider struct{}

func (p *oauthBindTestProvider) GetName() string { return "OAuthBindTest" }

func (p *oauthBindTestProvider) IsEnabled() bool { return true }

func (p *oauthBindTestProvider) ExchangeToken(ctx context.Context, code string, c *gin.Context) (*oauth.OAuthToken, error) {
	return &oauth.OAuthToken{AccessToken: "bind-token"}, nil
}

func (p *oauthBindTestProvider) GetUserInfo(ctx context.Context, token *oauth.OAuthToken) (*oauth.OAuthUser, error) {
	return &oauth.OAuthUser{
		ProviderUserID: "provider-user-1",
		Username:       "bind-user",
		Extra:          map[string]any{},
	}, nil
}

func (p *oauthBindTestProvider) IsUserIDTaken(providerUserID string) bool { return false }

func (p *oauthBindTestProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	return nil
}

func (p *oauthBindTestProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.GitHubId = providerUserID
}

func TestHandleOAuthBindReturnsStructuredBindAction(t *testing.T) {
	setupSecureVerificationTestDB(t)
	gin.SetMode(gin.TestMode)

	oauth.Register("oauth-bind-test", &oauthBindTestProvider{})

	user := createSecureVerificationTestUser(t, 1, "oauth-bind-access-token")

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", user.Id)
		session.Set("username", user.Username)
		session.Set("oauth_state", "state-123")
		_ = session.Save()
		c.Next()
	})
	router.GET("/api/oauth/:provider", HandleOAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/oauth/oauth-bind-test?code=abc&state=state-123", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var body struct {
		Success bool `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Action string `json:"action"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.Success {
		t.Fatalf("expected success response, got: %s", recorder.Body.String())
	}
	if body.Message == "" {
		t.Fatalf("expected translated bind success message, got empty response body: %s", recorder.Body.String())
	}
	if body.Data.Action != "bind" {
		t.Fatalf("data.action = %q, want %q", body.Data.Action, "bind")
	}

	updatedUser, err := model.GetUserById(user.Id, true)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if updatedUser.GitHubId != "provider-user-1" {
		t.Fatalf("github_id = %q, want %q", updatedUser.GitHubId, "provider-user-1")
	}
}
