package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID      int64  `json:"user_id"`
	Email       string `json:"email"`
	AccountType string `json:"account_type"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID int64, email, accountType string) (string, error) {
	expiryStr := os.Getenv("JWT_ACCESS_EXPIRY")
	if expiryStr == "" {
		expiryStr = "15m"
	}

	expiry, _ := time.ParseDuration(expiryStr)

	claims := JWTClaims{
		UserID:      userID,
		Email:       email,
		AccountType: accountType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	return token.SignedString([]byte(secret))
}

func GenerateRefreshToken(userID int64) (string, error) {
	expiryStr := os.Getenv("JWT_REFRESH_EXPIRY")
	if expiryStr == "" {
		expiryStr = "168h"
	}
	expiry, _ := time.ParseDuration(expiryStr)

	claims := jwt.RegisteredClaims{
		Subject:   string(rune(userID)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
