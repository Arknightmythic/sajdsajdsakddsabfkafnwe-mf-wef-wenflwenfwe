package guide

import (
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	repo := NewGuideRepository(db)
	service := NewGuideService(repo, redisClient)
	handler := NewGuideHandler(service, redisClient)

	r.GET("/api/guides/view-file", handler.ViewFile)

	guideGroup := r.Group("/api/guides")
	guideGroup.Use(middleware.AuthMiddleware())
	{
		guideGroup.POST("", handler.UploadGuide)
		guideGroup.GET("", handler.GetAll)
		guideGroup.GET("/:id", handler.GetByID)
		guideGroup.PUT("/:id", handler.UpdateGuide)
		guideGroup.DELETE("/:id", handler.DeleteGuide)

		guideGroup.POST("/generate-view-url", handler.GenerateViewURL)
	}
}
