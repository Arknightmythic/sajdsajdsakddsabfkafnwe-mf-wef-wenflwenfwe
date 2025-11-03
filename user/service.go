package user

import (
	"context"
	"errors"
	"fmt"
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

func (s *UserService) GetUsers(query *GetUsersQuery) (*PaginatedUsersResponse, error) {
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	users, err := s.repo.GetUsers(query)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.GetTotalUsers(query)
	if err != nil {
		return nil, err
	}

	totalPages := (total + query.Limit - 1) / query.Limit

	getUserDTOs := make([]GetUserDTO, len(users))
	for i, user := range users {
		var roleDTO *role.GetRoleDTO
		if user.RoleID != nil {
			roleData, err := s.serviceRole.GetByID(*user.RoleID)
			if err == nil {
				roleDTO = roleData
			}
		}

		getUserDTOs[i] = GetUserDTO{
			ID:          user.ID,
			Name:        user.Name,
			Email:       user.Email,
			AccountType: user.AccountType,
			Role:        roleDTO,
			Phone:       user.Phone,
		}
	}

	return &PaginatedUsersResponse{
		Data:       getUserDTOs,
		Total:      total,
		Limit:      query.Limit,
		Offset:     query.Offset,
		TotalPages: totalPages,
	}, nil
}

func (s *UserService) GetUserByID(id int) (*GetUserDTO, error) {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	getUserDto := &GetUserDTO{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		AccountType: user.AccountType,
		Phone:       user.Phone,
	}

	if user.RoleID != nil {
		role, err := s.serviceRole.GetByID(*user.RoleID)
		if err == nil {
			getUserDto.Role = role
		}
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

	accountType := ""
	if user.AccountType != nil {
		accountType = *user.AccountType
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Name, user.Email, accountType)
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

	userID := claims.UserID

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

	accountType := ""
	if user.AccountType != nil {
		accountType = *user.AccountType
	}

	return auth.GenerateAccessToken(user.ID, user.Name, user.Email, accountType)
}
