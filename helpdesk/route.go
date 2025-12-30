package helpdesk

import (
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/messaging"
	"dokuprime-be/middleware"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repo := NewHelpdeskRepository(db)
	service := NewHelpdeskService(repo)

	externalAPIConfig := config.LoadExternalAPIConfig()
	externalClient := external.NewClient(externalAPIConfig)

	wsURL := os.Getenv("WEBSOCKET_URL")
	if wsURL == "" {
		wsURL = "ws://localhost:8080"
	}

	wsToken := os.Getenv("WEBSOCKET_SECRET_KEY")
	if wsToken == "" {
		wsToken = "bkpm-jaya-jaya-jaya"
	}

	messageService := messaging.NewMessageService(db, wsURL, wsToken, externalClient)

	handler := NewHelpdeskHandler(service, messageService)

	helpdeskRoutes := r.Group("/api/helpdesk")
	helpdeskRoutes.Use(middleware.AuthMiddleware())
	{
		helpdeskRoutes.POST("", handler.CreateHelpdesk)
		helpdeskRoutes.GET("", handler.GetAll)
		helpdeskRoutes.GET("/:id", handler.GetHelpdeskByID)
		helpdeskRoutes.PUT("/:id", handler.UpdateHelpdesk)
		helpdeskRoutes.DELETE("/:id", handler.DeleteHelpdesk)
		helpdeskRoutes.POST("/ask", handler.AskHelpdesk)
		helpdeskRoutes.GET("/summary", handler.GetSummary)
		helpdeskRoutes.POST("/solved/:id", handler.SolvedConversation)
		helpdeskRoutes.GET("/switch", handler.GetSwitchStatus)
		helpdeskRoutes.POST("/switch", handler.UpdateSwitchStatus)
	}
}
