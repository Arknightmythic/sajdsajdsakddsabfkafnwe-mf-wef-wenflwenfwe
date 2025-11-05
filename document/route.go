package document

import (
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutesWithProcessor(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) *AsyncProcessor {
	externalConfig := config.LoadExternalAPIConfig()
	externalClient := external.NewClient(externalConfig)

	asyncProcessor := NewAsyncProcessor(externalClient, 5)

	repo := NewDocumentRepository(db)
	service := NewDocumentService(repo, redisClient, asyncProcessor, externalClient)
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
		documentRoutes.DELETE("/:id", handler.DeleteDocument)
		documentRoutes.GET("/download/:filename", handler.DownloadDocument)
		documentRoutes.GET("/all-details", handler.GetAllDocumentDetails)
		documentRoutes.GET("/queue-status", handler.GetQueueStatus)
	}

	return asyncProcessor
}

func RegisterRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	RegisterRoutesWithProcessor(r, db, redisClient)
}
