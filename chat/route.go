package chat

import (
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/helpdesk"
	"dokuprime-be/messaging"
	"dokuprime-be/middleware"
	"log"

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

	wsURL := config.AppConfig.WebSocketURL
	if wsURL == "" {
		wsURL = "ws://localhost:8080"
	}

	wsToken := config.AppConfig.WebSocketSecretKey
	if wsToken == "" {
		wsToken = "bkpm-jaya-jaya-jaya"
	}

	messageService := messaging.NewMessageService(db, wsURL, wsToken, externalClient)

	handler := NewChatHandler(service, externalClient, wsURL, wsToken, *helpdeskService, *messageService)

	defaulChattLimit := 8

	limitChat := config.AppConfig.LimitConcurrentChatRequests
	if limitChat != 0 && limitChat > 0 {
		defaulChattLimit = limitChat
	}

	log.Println(defaulChattLimit, "is the limit for concurrent chat requests")

	chatRoutes := r.Group("/api/chat")
	chatRoutes.Use(middleware.AuthMiddleware())
	chatRoutes.Use(middleware.GlobalConcurrencyLimitMiddleware())
	{
		chatRoutes.POST("/history", handler.CreateChatHistory)
		chatRoutes.GET("/history", handler.GetChatHistories)
		chatRoutes.GET("/history/download", handler.DownloadChatHistory)
		chatRoutes.GET("/history/session/:session_id", handler.GetChatHistoryBySessionID)
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

		limitedRoutes := chatRoutes.Group("")
		limitedRoutes.Use(middleware.ChatConcurrencyLimitMiddleware(defaulChattLimit, externalClient))
		{
			limitedRoutes.POST("/ask", handler.Ask)
		}

		limitedRoutes.POST("/validate", handler.ValidateAnswer)
		chatRoutes.POST("/feedback", handler.Feedback)
	}

	apiKeyRoutes := r.Group("/api/chat/multichannel")
	apiKeyRoutes.Use(middleware.APIKeyMiddleware())
	apiKeyRoutes.Use(middleware.GlobalConcurrencyLimitMiddleware())
	{
		apiKeyRoutes.POST("/feedback", handler.Feedback)

		limitedMultichannel := apiKeyRoutes.Group("")
		limitedMultichannel.Use(middleware.ChatConcurrencyLimitMiddleware(defaulChattLimit, externalClient))
		{
			limitedMultichannel.POST("/ask", handler.Ask)
		}
	}
}
