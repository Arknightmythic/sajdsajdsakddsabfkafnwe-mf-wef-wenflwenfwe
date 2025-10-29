package document

import (
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repo := NewDocumentRepository(db)
	service := NewDocumentService(repo)
	handler := NewDocumentHandler(service)

	documentRoutes := r.Group("/api/documents")
	documentRoutes.Use(middleware.AuthMiddleware())
	{
		documentRoutes.POST("/upload", handler.UploadDocument)
		documentRoutes.GET("", handler.GetDocuments)
		documentRoutes.GET("/details", handler.GetDocumentDetails)
		documentRoutes.PUT("/update", handler.UpdateDocument)
		documentRoutes.PUT("/approve/:id", handler.ApproveDocument)
		documentRoutes.PUT("/reject/:id", handler.RejectDocument)
		documentRoutes.GET("/download/:filename", handler.DownloadDocument)
	}
}