package grafana

import (
	"dokuprime-be/audit"
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, redisClient *redis.Client, db *sqlx.DB) {
	service := NewGrafanaService(redisClient)
	handler := NewGrafanaHandler(service, redisClient)

	auditRepo := audit.NewAuditRepository(db)
	auditService := audit.NewAuditService(auditRepo)

	r.GET("/api/grafana/view-embed", handler.ViewEmbed)

	grafanaGroup := r.Group("/api/grafana")
	grafanaGroup.Use(middleware.AuthMiddleware())
	grafanaGroup.Use(middleware.AuditMiddleware(auditService))
	grafanaGroup.Use(middleware.GlobalConcurrencyLimitMiddleware())
	{
		grafanaGroup.POST("/generate-embed-url", handler.GenerateEmbedURL)
	}
}
