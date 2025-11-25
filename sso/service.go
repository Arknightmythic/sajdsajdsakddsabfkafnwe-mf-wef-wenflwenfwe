package sso

import (
	"context"
	"crypto/rand"
	"dokuprime-be/auth"
	"dokuprime-be/user"
	"dokuprime-be/util"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
	"golang.org/x/oauth2"
	"github.com/redis/go-redis/v9"
)

type SSOService struct {
	userRepo *user.UserRepository
	redis    *redis.Client
}

func NewSSOService(userRepo *user.UserRepository, redisClient *redis.Client) *SSOService {
	return &SSOService{
		userRepo: userRepo,
		redis:    redisClient,
	}
}

func (s *SSOService) GetLoginURL(state string) string {
    oauthConfig := GetMicrosoftOAuthConfig()
    return oauthConfig.AuthCodeURL(
        state, 
        oauth2.SetAuthURLParam("prompt", "select_account"),
    )
}

func (s *SSOService) HandleCallback(ctx context.Context, code string) (*user.LoginResponse, error) {
	oauthConfig := GetMicrosoftOAuthConfig()


	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}


	client := oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var msUser MicrosoftUser
	if err := json.Unmarshal(body, &msUser); err != nil {
		return nil, err
	}


	email := msUser.Email
	if email == "" {
		email = msUser.UserPrincipalName
	}

	if email == "" {
		return nil, errors.New("email not provided by identity provider")
	}


	dbUser, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
	
	
		randomPass := util.RandString(32)
		hashedPassword, _ := util.GenerateDeterministicHash(randomPass)

		accountType := "microsoft"
		var defaultRole *int = nil

		newUser := &user.User{
			Name:        msUser.DisplayName,
			Email:       email,
			Password:    hashedPassword,
			AccountType: &accountType,
			RoleID:      defaultRole,
			Phone:       nil,
		}

		dbUser, err = s.userRepo.CreateUser(newUser)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}


	sessionIDBytes := make([]byte, 16)
	rand.Read(sessionIDBytes)
	sessionID := hex.EncodeToString(sessionIDBytes)

	accountTypeStr := ""
	if dbUser.AccountType != nil {
		accountTypeStr = *dbUser.AccountType
	}

	accessToken, err := auth.GenerateAccessToken(dbUser.ID, dbUser.Name, dbUser.Email, accountTypeStr)
	if err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateRefreshToken(dbUser.ID)
	if err != nil {
		return nil, err
	}


	redisKey := fmt.Sprintf("refresh_token:%d:%s", dbUser.ID, sessionID)
	err = s.redis.Set(ctx, redisKey, refreshToken, 7*24*time.Hour).Err()
	if err != nil {
		return nil, err
	}

	sessionSetKey := fmt.Sprintf("user_sessions:%d", dbUser.ID)
	s.redis.SAdd(ctx, sessionSetKey, sessionID)
	s.redis.Expire(ctx, sessionSetKey, 7*24*time.Hour+time.Hour)


	userDTO := &user.GetUserDTO{
		ID:          dbUser.ID,
		Name:        dbUser.Name,
		Email:       dbUser.Email,
		AccountType: dbUser.AccountType,
		Role:        nil,
		Phone:       dbUser.Phone,
	}

	return &user.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		SessionID:    sessionID,
		User:         userDTO,
	}, nil
}