package grafana

import (
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, redisClient *redis.Client) {
	service := NewGrafanaService(redisClient)
	handler := NewGrafanaHandler(service, redisClient)

	
	r.GET("/api/grafana/view-embed", handler.ViewEmbed)

	
	grafanaGroup := r.Group("/api/grafana")
	grafanaGroup.Use(middleware.AuthMiddleware())
	{
		grafanaGroup.POST("/generate-embed-url", handler.GenerateEmbedURL)
	}
}