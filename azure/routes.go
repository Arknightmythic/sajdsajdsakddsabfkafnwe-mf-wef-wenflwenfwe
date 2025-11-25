package azure

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	config := NewAzureConfig()
	repo := NewAzureRepository(db)
	service := NewAzureService(config, repo, redisClient)
	handler := NewAzureHandler(service)

	azureGroup := r.Group("/api/authazure")
	{
		azureGroup.GET("/login", handler.Login)
		azureGroup.GET("/callback", handler.Callback)
		azureGroup.GET("/logout", handler.Logout)
	}
}