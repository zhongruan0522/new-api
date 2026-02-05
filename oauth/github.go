package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func init() {
	Register("github", &GitHubProvider{})
}

// GitHubProvider implements OAuth for GitHub
type GitHubProvider struct{}

type gitHubOAuthResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type gitHubUser struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (p *GitHubProvider) GetName() string {
	return "GitHub"
}

func (p *GitHubProvider) IsEnabled() bool {
	return common.GitHubOAuthEnabled
}

func (p *GitHubProvider) ExchangeToken(ctx context.Context, code string, c *gin.Context) (*OAuthToken, error) {
	if code == "" {
		return nil, NewOAuthError(i18n.MsgOAuthInvalidCode, nil)
	}

	logger.LogDebug(ctx, "[OAuth-GitHub] ExchangeToken: code=%s...", code[:min(len(code), 10)])

	values := map[string]string{
		"client_id":     common.GitHubClientId,
		"client_secret": common.GitHubClientSecret,
		"code":          code,
	}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := http.Client{
		Timeout: 20 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] ExchangeToken error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "GitHub"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-GitHub] ExchangeToken response status: %d", res.StatusCode)

	var oAuthResponse gitHubOAuthResponse
	err = json.NewDecoder(res.Body).Decode(&oAuthResponse)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] ExchangeToken decode error: %s", err.Error()))
		return nil, err
	}

	if oAuthResponse.AccessToken == "" {
		logger.LogError(ctx, "[OAuth-GitHub] ExchangeToken failed: empty access token")
		return nil, NewOAuthError(i18n.MsgOAuthTokenFailed, map[string]any{"Provider": "GitHub"})
	}

	logger.LogDebug(ctx, "[OAuth-GitHub] ExchangeToken success: scope=%s", oAuthResponse.Scope)

	return &OAuthToken{
		AccessToken: oAuthResponse.AccessToken,
		TokenType:   oAuthResponse.TokenType,
		Scope:       oAuthResponse.Scope,
	}, nil
}

func (p *GitHubProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	logger.LogDebug(ctx, "[OAuth-GitHub] GetUserInfo: fetching user info")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	client := http.Client{
		Timeout: 20 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] GetUserInfo error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "GitHub"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-GitHub] GetUserInfo response status: %d", res.StatusCode)

	var githubUser gitHubUser
	err = json.NewDecoder(res.Body).Decode(&githubUser)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] GetUserInfo decode error: %s", err.Error()))
		return nil, err
	}

	if githubUser.Login == "" {
		logger.LogError(ctx, "[OAuth-GitHub] GetUserInfo failed: empty login field")
		return nil, NewOAuthError(i18n.MsgOAuthUserInfoEmpty, map[string]any{"Provider": "GitHub"})
	}

	logger.LogDebug(ctx, "[OAuth-GitHub] GetUserInfo success: login=%s, name=%s, email=%s", githubUser.Login, githubUser.Name, githubUser.Email)

	return &OAuthUser{
		ProviderUserID: githubUser.Login,
		Username:       githubUser.Login,
		DisplayName:    githubUser.Name,
		Email:          githubUser.Email,
	}, nil
}

func (p *GitHubProvider) IsUserIDTaken(providerUserID string) bool {
	return model.IsGitHubIdAlreadyTaken(providerUserID)
}

func (p *GitHubProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	user.GitHubId = providerUserID
	return user.FillUserByGitHubId()
}

func (p *GitHubProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.GitHubId = providerUserID
}

func (p *GitHubProvider) GetProviderPrefix() string {
	return "github_"
}
