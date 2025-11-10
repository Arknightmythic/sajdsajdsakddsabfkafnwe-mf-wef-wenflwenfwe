package chat

import (
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repo := NewChatRepository(db)
	service := NewChatService(repo)

	externalAPIConfig := config.LoadExternalAPIConfig()
	externalClient := external.NewClient(externalAPIConfig)

	handler := NewChatHandler(service, externalClient)

	chatRoutes := r.Group("/api/chat")
	chatRoutes.Use(middleware.AuthMiddleware())
	{

		chatRoutes.POST("/history", handler.CreateChatHistory)
		chatRoutes.GET("/history", handler.GetChatHistories)
		chatRoutes.GET("/history/:id", handler.GetChatHistoryByID)
		chatRoutes.GET("/history/session/:session_id", handler.GetChatHistoryBySessionID)
		chatRoutes.PUT("/history/:id", handler.UpdateChatHistory)
		chatRoutes.DELETE("/history/:id", handler.DeleteChatHistory)

		chatRoutes.GET("/pairs/session/:session_id", handler.GetChatPairsBySessionID)
		chatRoutes.GET("/pairs/all", handler.GetChatPairsBySessionID)
		chatRoutes.GET("/debug/session/:session_id", handler.DebugChatHistory)

		chatRoutes.POST("/conversations", handler.CreateConversation)
		chatRoutes.GET("/conversations", handler.GetConversations)
		chatRoutes.GET("/conversations/:id", handler.GetConversationByID)
		chatRoutes.PUT("/conversations/:id", handler.UpdateConversation)
		chatRoutes.DELETE("/conversations/:id", handler.DeleteConversation)

		chatRoutes.POST("/ask", handler.Ask)
	}
}
