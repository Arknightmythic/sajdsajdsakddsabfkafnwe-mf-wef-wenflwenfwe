package user

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"dokuprime-be/auth"
	"dokuprime-be/util"

	"github.com/redis/go-redis/v9"
)

type UserService struct {
	Repo  *UserRepository
	Redis *redis.Client
}

func NewUserService(repo *UserRepository, redisClient *redis.Client) *UserService {
	return &UserService{
		Repo:  repo,
		Redis: redisClient,
	}
}

func (s *UserService) CreateUser(user *User) (*User, error) {
	log.Println(user.Password, " User passsword")
	hashedPassword, err := util.GenerateDeterministicHash(user.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}
	user.Password = hashedPassword

	log.Printf("Creating user with email: %s", user.Email)

	return s.Repo.CreateUser(user)
}

func (s *UserService) GetUsers() ([]User, error) {
	return s.Repo.GetUsers()
}

func (s *UserService) GetUserByID(id int) (*User, error) {
	return s.Repo.GetUserByID(id)
}

func (s *UserService) UpdateUser(id int, user *User) (*User, error) {
	if user.Password != "" {
		hashedPassword, err := util.GenerateDeterministicHash(user.Password)
		if err != nil {
			return nil, errors.New("failed to hash password")
		}
		user.Password = hashedPassword
	}
	return s.Repo.UpdateUser(id, user)
}

func (s *UserService) DeleteUser(id int) error {
	return s.Repo.DeleteUser(id)
}

func (s *UserService) Login(email, password string) (*LoginResponse, error) {
	log.Printf("Login attempt for email: %s", email)

	user, err := s.Repo.GetUserByEmail(email)
	if err != nil {
		log.Printf("User not found: %v", err)
		return nil, errors.New("invalid email or password")
	}

	// Verify password using deterministic hash
	err = util.VerifyPassword(user.Password, password)
	if err != nil {
		log.Printf("Password verification failed: %v", err)
		return nil, errors.New("invalid email or password")
	}

	log.Println("Password verified successfully")

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Email, user.AccountType)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%d", user.ID)
	err = s.Redis.Set(ctx, key, refreshToken, 7*24*time.Hour).Err()
	if err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	user.Password = ""
	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *UserService) Logout(userID int64) error {
	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%d", userID)
	return s.Redis.Del(ctx, key).Err()
}

func (s *UserService) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := auth.ValidateToken(refreshToken)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return "", errors.New("invalid user ID in token")
	}

	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%d", userID)
	storedToken, err := s.Redis.Get(ctx, key).Result()
	if err != nil || storedToken != refreshToken {
		return "", errors.New("refresh token not found or invalid")
	}

	user, err := s.Repo.GetUserByID(int(userID))
	if err != nil {
		return "", errors.New("user not found")
	}

	return auth.GenerateAccessToken(user.ID, user.Email, user.AccountType)
}
