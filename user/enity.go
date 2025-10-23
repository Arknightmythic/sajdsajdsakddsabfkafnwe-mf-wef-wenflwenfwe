package user

type User struct {
	ID          int64  `db:"id" json:"id"`
	Email       string `db:"email" json:"email"`
	Password    string `db:"password" json:"password"`
	AccountType string `db:"account_type" json:"account_type"`
	Phone       string `db:"phone" json:"phone"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
