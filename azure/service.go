package azure

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dokuprime-be/auth"

	"github.com/redis/go-redis/v9"
)

type AzureService struct {
	Config *AzureConfig
	Repo   *AzureRepository
	Redis  *redis.Client
}

func NewAzureService(config *AzureConfig, repo *AzureRepository, redis *redis.Client) *AzureService {
	return &AzureService{
		Config: config,
		Repo:   repo,
		Redis:  redis,
	}
}

func (s *AzureService) GetAuthURL() (string, string, error) {
	state := generateState()

	ctx := context.Background()
	key := fmt.Sprintf("oauth_state:%s", state)
	err := s.Redis.Set(ctx, key, "valid", 10*time.Minute).Err()
	if err != nil {
		return "", "", fmt.Errorf("failed to store state: %w", err)
	}

	params := url.Values{}
	params.Add("client_id", s.Config.ClientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", s.Config.RedirectURI)
	params.Add("response_mode", "query")
	params.Add("scope", strings.Join(s.Config.Scope, " "))
	params.Add("state", state)
	params.Add("prompt", "select_account")

	authURL := fmt.Sprintf("%s?%s", s.Config.AuthorizationURL, params.Encode())
	return authURL, state, nil
}

func (s *AzureService) ValidateState(state string) error {
	if state == "" {
		return errors.New("state parameter is empty")
	}

	ctx := context.Background()
	key := fmt.Sprintf("oauth_state:%s", state)

	exists, err := s.Redis.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to validate state: %w", err)
	}

	if exists == 0 {
		return errors.New("invalid or expired state parameter")
	}

	s.Redis.Del(ctx, key)

	return nil
}

func (s *AzureService) ExchangeCodeForToken(code string) (*AzureTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", s.Config.ClientID)
	data.Set("client_secret", s.Config.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", s.Config.RedirectURI)
	data.Set("scope", strings.Join(s.Config.Scope, " "))

	req, err := http.NewRequest("POST", s.Config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to authenticate with Azure AD: %s", string(body))
	}

	var tokenResp AzureTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (s *AzureService) GetAzureUserInfo(accessToken string) (*AzureUserInfo, error) {
	req, err := http.NewRequest("GET", s.Config.GraphAPIURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch user info from Microsoft Graph: %s", string(body))
	}

	var userInfo AzureUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (s *AzureService) ProcessAzureLogin(code string) (*AzureLoginResponse, error) {

	tokenData, err := s.ExchangeCodeForToken(code)
	if err != nil {
		return nil, err
	}

	userInfo, err := s.GetAzureUserInfo(tokenData.AccessToken)
	if err != nil {
		return nil, err
	}

	email := userInfo.Mail
	if email == "" {
		email = userInfo.UserPrincipalName
	}

	if email == "" {
		return nil, errors.New("email not provided by Microsoft")
	}

	user, err := s.Repo.FindOrCreateUser(userInfo.DisplayName, email)
	if err != nil {
		return nil, err
	}

	accountType := ""
	accType, err := s.Repo.GetAccountTypeByUserID(user.ID)
	if err == nil && accType != nil {
		accountType = *accType
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Name, user.Email, accountType)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	sessionID, err := generateSessionID()
	if err != nil {
		return nil, errors.New("failed to generate session")
	}

	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%d:%s", user.ID, sessionID)
	err = s.Redis.Set(ctx, key, refreshToken, 7*24*time.Hour).Err()
	if err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	sessionSetKey := fmt.Sprintf("user_sessions:%d", user.ID)
	err = s.Redis.SAdd(ctx, sessionSetKey, sessionID).Err()
	if err != nil {
		return nil, errors.New("failed to register session")
	}

	s.Redis.Expire(ctx, sessionSetKey, 7*24*time.Hour+time.Hour)

	return &AzureLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		SessionID:    sessionID,
		User: map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	}, nil
}

func (s *AzureService) GetLogoutURL() string {
	postLogoutRedirect := fmt.Sprintf("%s/auth-microsoft/callback?logout=logout", s.Config.FrontendCallbackURL)
	return fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/logout?post_logout_redirect_uri=%s",
		s.Config.TenantID,
		url.QueryEscape(postLogoutRedirect),
	)
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
