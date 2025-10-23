package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"dokuprime-be/auth"
	"dokuprime-be/role"
	"dokuprime-be/util"

	"github.com/redis/go-redis/v9"
)

type UserService struct {
	repo        *UserRepository
	Redis       *redis.Client
	serviceRole *role.RoleService
}

func NewUserService(repo *UserRepository, redisClient *redis.Client, serviceRole *role.RoleService) *UserService {
	return &UserService{
		repo:        repo,
		Redis:       redisClient,
		serviceRole: serviceRole,
	}
}

func (s *UserService) CreateUser(user *User) (*User, error) {
	hashedPassword, err := util.GenerateDeterministicHash(user.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}
	user.Password = hashedPassword

	return s.repo.CreateUser(user)
}

func (s *UserService) GetUsers() ([]GetUserDTO, error) {
	users, err := s.repo.GetUsers()
	if err != nil {
		return nil, err
	}

	var getUsersDto []GetUserDTO
	for _, user := range users {
		role, err := s.serviceRole.GetByID(user.RoleID)
		if err != nil {
			return nil, err
		}

		getUserDto := GetUserDTO{
			ID:          user.ID,
			Email:       user.Email,
			AccountType: user.AccountType,
			Role:        *role,
			Phone:       user.Phone,
		}

		getUsersDto = append(getUsersDto, getUserDto)
	}

	return getUsersDto, nil
}

func (s *UserService) GetUserByID(id int) (*GetUserDTO, error) {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	role, err := s.serviceRole.GetByID(user.RoleID)
	if err != nil {
		return nil, err
	}

	getUserDto := &GetUserDTO{
		ID:          user.ID,
		Email:       user.Email,
		AccountType: user.AccountType,
		Role:        *role,
		Phone:       user.Phone,
	}

	return getUserDto, nil
}

func (s *UserService) UpdateUser(id int, user *User) (*User, error) {
	if user.Password != "" {
		hashedPassword, err := util.GenerateDeterministicHash(user.Password)
		if err != nil {
			return nil, errors.New("failed to hash password")
		}
		user.Password = hashedPassword
	}
	return s.repo.UpdateUser(id, user)
}

func (s *UserService) DeleteUser(id int) error {
	return s.repo.DeleteUser(id)
}

func (s *UserService) Login(email, password string) (*LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	err = util.VerifyPassword(user.Password, password)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

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

	userValid, err := s.GetUserByID(int(user.ID))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userValid,
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

	user, err := s.repo.GetUserByID(int(userID))
	if err != nil {
		return "", errors.New("user not found")
	}

	return auth.GenerateAccessToken(user.ID, user.Email, user.AccountType)
}
