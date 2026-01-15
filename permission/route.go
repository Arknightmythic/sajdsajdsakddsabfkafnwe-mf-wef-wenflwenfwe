package permission

import (
	"dokuprime-be/audit"
	"dokuprime-be/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repo := NewPermissionRepository(db)
	service := NewPermissionService(repo)
	handler := NewPermissionHandler(service)

	auditRepo := audit.NewAuditRepository(db)
	auditService := audit.NewAuditService(auditRepo)

	permissionRoutes := r.Group("/api/permissions")

	permissionRoutes.Use(middleware.AuthMiddleware())
	permissionRoutes.Use(middleware.AuditMiddleware(auditService))
	permissionRoutes.Use(middleware.GlobalConcurrencyLimitMiddleware())
	{
		permissionRoutes.POST("", handler.CreatePermission)
		permissionRoutes.GET("", handler.GetPermissions)
		permissionRoutes.GET("/:id", handler.GetPermissionByID)
		permissionRoutes.PUT("/:id", handler.UpdatePermission)
		permissionRoutes.DELETE("/:id", handler.DeletePermission)
	}

}
