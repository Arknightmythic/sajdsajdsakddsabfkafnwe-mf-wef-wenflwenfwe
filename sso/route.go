package sso

import (
	"dokuprime-be/user"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	// Kita reuse user repository
	userRepo := user.NewUserRepository(db)
	service := NewSSOService(userRepo, redisClient)
	handler := NewSSOHandler(service)

	authGroup := r.Group("/auth/microsoft")
	{
		authGroup.GET("/login", handler.Login)
		authGroup.GET("/callback", handler.Callback)
	}
}