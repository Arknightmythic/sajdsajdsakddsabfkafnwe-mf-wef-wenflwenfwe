package document

import (
	"dokuprime-be/audit"
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutesWithProcessor(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) *AsyncProcessor {
	auditRepo := audit.NewAuditRepository(db)
	auditService := audit.NewAuditService(auditRepo)

	externalConfig := config.LoadExternalAPIConfig()
	externalClient := external.NewClient(externalConfig, auditService)

	asyncProcessor := NewAsyncProcessor(externalClient, 5)

	repo := NewDocumentRepository(db)
	service := NewDocumentService(repo, redisClient, asyncProcessor, externalClient)
	handler := NewDocumentHandler(service, redisClient)

	r.GET("/api/documents/view-file", handler.ViewDocument)

	documentRoutes := r.Group("/api/documents")

	documentRoutes.Use(middleware.AuthMiddleware())
	documentRoutes.Use(middleware.AuditMiddleware(auditService))
	documentRoutes.Use(middleware.GlobalConcurrencyLimitMiddleware())
	documentRoutes.Use(middleware.TimeoutMiddleware(10 * time.Minute))
	{
		documentRoutes.POST("/batch-upload", handler.BatchUploadDocument)
		documentRoutes.GET("/batch-status", handler.GetBatchUploadStatus)
		documentRoutes.POST("/batch-delete", handler.BatchDeleteDocument)

		documentRoutes.POST("/generate-view-url", handler.GenerateViewURL)
		documentRoutes.POST("/generate-view-url-id", handler.GenerateViewURLByID)
		documentRoutes.POST("/generate-view-url-docid", handler.GenerateViewURLByDocumentID)
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
		documentRoutes.POST("/check-duplicates", handler.CheckDuplicates)
	}

	crawlerRoutes := r.Group("/api/documents/crawler")
	crawlerRoutes.Use(middleware.APIKeyMiddleware())
	crawlerRoutes.Use(middleware.AuditMiddleware(auditService))
	crawlerRoutes.Use(middleware.GlobalConcurrencyLimitMiddleware())
	crawlerRoutes.Use(middleware.TimeoutMiddleware(10 * time.Minute))
	{
		crawlerRoutes.POST("/upload", handler.CrawlerBatchUpload)
	}

	return asyncProcessor
}

func RegisterRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	RegisterRoutesWithProcessor(r, db, redisClient)
}
