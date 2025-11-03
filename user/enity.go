package user

import "dokuprime-be/role"

type User struct {
	ID          int64   `db:"id" json:"id"`
	Name        string  `db:"name" json:"name"`
	Email       string  `db:"email" json:"email"`
	Password    string  `db:"password" json:"password"`
	AccountType *string `db:"account_type" json:"account_type,omitempty"`
	RoleID      *int    `db:"role_id" json:"role_id,omitempty"`
	Phone       *string `db:"phone" json:"phone,omitempty"`
}

type GetUserDTO struct {
	ID          int64            `db:"id" json:"id"`
	Name        string           `db:"name" json:"name"`
	Email       string           `db:"email" json:"email"`
	AccountType *string          `db:"account_type" json:"account_type,omitempty"`
	Role        *role.GetRoleDTO `db:"role" json:"role,omitempty"`
	Phone       *string          `db:"phone" json:"phone,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         *GetUserDTO `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type PaginatedUsersResponse struct {
	Data       []GetUserDTO `json:"data"`
	Total      int          `json:"total"`
	Limit      int          `json:"limit"`
	Offset     int          `json:"offset"`
	TotalPages int          `json:"total_pages"`
}

type GetUsersQuery struct {
	Limit       int     `form:"limit"`
	Offset      int     `form:"offset"`
	AccountType *string `form:"account_type"`
	RoleID      *int    `form:"role_id"`
	TeamID      *int    `form:"team_id"`
	Search      *string `form:"search"`
}
