package chat

import (
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/helpdesk"
	"dokuprime-be/messaging"
	"dokuprime-be/middleware"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

const (
	urlHistoryID     = "/history/:id"
	urConversationID = "/conversations/:id"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repo := NewChatRepository(db)
	service := NewChatService(repo)

	externalAPIConfig := config.LoadExternalAPIConfig()
	externalClient := external.NewClient(externalAPIConfig)

	helpdeskService := helpdesk.NewHelpdeskService(helpdesk.NewHelpdeskRepository(db))

	wsURL := os.Getenv("WEBSOCKET_URL")
	if wsURL == "" {
		wsURL = "ws://localhost:8080"
	}

	wsToken := os.Getenv("WEBSOCKET_SECRET_KEY")
	if wsToken == "" {
		wsToken = "bkpm-jaya-jaya-jaya"
	}

	messageService := messaging.NewMessageService(db, wsURL, wsToken, externalClient)

	handler := NewChatHandler(service, externalClient, wsURL, wsToken, *helpdeskService, *messageService)

	chatRoutes := r.Group("/api/chat")
	chatRoutes.Use(middleware.AuthMiddleware())
	{
		chatRoutes.POST("/history", handler.CreateChatHistory)
		chatRoutes.GET("/history", handler.GetChatHistories)

		// PENTING: Route spesifik HARUS di atas route dengan parameter dinamis
		chatRoutes.GET("/history/download", handler.DownloadChatHistory)
		chatRoutes.GET("/history/session/:session_id", handler.GetChatHistoryBySessionID)

		// Route dengan parameter dinamis di bawah
		chatRoutes.GET(urlHistoryID, handler.GetChatHistoryByID)
		chatRoutes.PUT(urlHistoryID, handler.UpdateChatHistory)
		chatRoutes.DELETE(urlHistoryID, handler.DeleteChatHistory)

		chatRoutes.GET("/pairs/session/:session_id", handler.GetChatPairsBySessionID)
		chatRoutes.GET("/pairs/all", handler.GetChatPairsBySessionID)
		chatRoutes.GET("/debug/session/:session_id", handler.DebugChatHistory)

		chatRoutes.POST("/conversations", handler.CreateConversation)
		chatRoutes.GET("/conversations", handler.GetConversations)
		chatRoutes.GET(urConversationID, handler.GetConversationByID)
		chatRoutes.PUT(urConversationID, handler.UpdateConversation)
		chatRoutes.DELETE(urConversationID, handler.DeleteConversation)

		chatRoutes.POST("/ask", handler.Ask)
		chatRoutes.POST("/validate", handler.ValidateAnswer)

		chatRoutes.POST("/feedback", handler.Feedback)
	}

	apiKeyRoutes := r.Group("/api/chat/multichannel")
	apiKeyRoutes.Use(middleware.APIKeyMiddleware())
	{
		apiKeyRoutes.POST("/feedback", handler.Feedback)
		apiKeyRoutes.POST("/ask", handler.Ask)
	}
}
