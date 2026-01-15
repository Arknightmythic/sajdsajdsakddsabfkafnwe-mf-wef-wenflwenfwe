package helpdesk

import (
	"dokuprime-be/audit"
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/messaging"
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repo := NewHelpdeskRepository(db)
	service := NewHelpdeskService(repo)

	auditRepo := audit.NewAuditRepository(db)
	auditService := audit.NewAuditService(auditRepo)

	externalAPIConfig := config.LoadExternalAPIConfig()
	externalClient := external.NewClient(externalAPIConfig, auditService)

	wsURL := config.AppConfig.WebSocketURL
	if wsURL == "" {
		wsURL = "ws://localhost:8080"
	}

	wsToken := config.AppConfig.WebSocketSecretKey
	if wsToken == "" {
		wsToken = "bkpm-jaya-jaya-jaya"
	}

	messageService := messaging.NewMessageService(db, wsURL, wsToken, externalClient)

	handler := NewHelpdeskHandler(service, messageService)

	helpdeskRoutes := r.Group("/api/helpdesk")
	helpdeskRoutes.Use(middleware.AuthMiddleware())
	helpdeskRoutes.Use(middleware.AuditMiddleware(auditService))
	helpdeskRoutes.Use(middleware.GlobalConcurrencyLimitMiddleware())
	{
		helpdeskRoutes.POST("", handler.CreateHelpdesk)
		helpdeskRoutes.GET("", handler.GetAll)
		helpdeskRoutes.GET("/:id", handler.GetHelpdeskByID)
		helpdeskRoutes.PUT("/:id", handler.UpdateHelpdesk)
		helpdeskRoutes.DELETE("/:id", handler.DeleteHelpdesk)
		helpdeskRoutes.POST("/ask", handler.AskHelpdesk)

		helpdeskRoutes.POST("/solved/:id", handler.SolvedConversation)
		helpdeskRoutes.GET("/switch", handler.GetSwitchStatus)
		helpdeskRoutes.POST("/switch", middleware.PermissionMiddleware(db, "helpdesk:toggle-helpdesk"), handler.UpdateSwitchStatus)
	}
}
