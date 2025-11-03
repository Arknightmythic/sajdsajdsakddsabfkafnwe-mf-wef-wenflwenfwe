package document

import (
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	repo := NewDocumentRepository(db)
	service := NewDocumentService(repo, redisClient)
	handler := NewDocumentHandler(service, redisClient)

	r.GET("/api/documents/view-file", handler.ViewDocument)

	documentRoutes := r.Group("/api/documents")
	documentRoutes.Use(middleware.AuthMiddleware())
	{
		documentRoutes.POST("/generate-view-url", handler.GenerateViewURL)
		documentRoutes.POST("/upload", handler.UploadDocument)
		documentRoutes.GET("", handler.GetDocuments)
		documentRoutes.GET("/details", handler.GetDocumentDetails)
		documentRoutes.PUT("/update", handler.UpdateDocument)
		documentRoutes.PUT("/approve/:id", handler.ApproveDocument)
		documentRoutes.PUT("/reject/:id", handler.RejectDocument)
		documentRoutes.GET("/download/:filename", handler.DownloadDocument)
	}
}