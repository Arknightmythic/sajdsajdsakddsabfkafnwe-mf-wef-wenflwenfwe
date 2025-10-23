package user

import (
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	repo := NewUserRepository(db)
	service := NewUserService(repo, redisClient)
	handler := NewUserHandler(service)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", handler.Login)
		authGroup.POST("/refresh", handler.RefreshToken)
	}

	userGroup := r.Group("/users")

	userGroup.POST("/", handler.CreateUser)

	userGroup.Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/", handler.GetUsers)
		userGroup.GET("/:id", handler.GetUserByID)
		userGroup.PUT("/:id", handler.UpdateUser)
		userGroup.DELETE("/:id", handler.DeleteUser)
		userGroup.POST("/logout", handler.Logout)
	}
}
