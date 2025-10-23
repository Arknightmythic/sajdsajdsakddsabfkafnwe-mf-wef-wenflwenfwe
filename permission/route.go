package permission

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repo := NewPermissionRepository(db)
	service := NewPermissionService(repo)
	handler := NewPermissionHandler(service)

	permissionRoutes := r.Group("/api/permissions")
	{
		permissionRoutes.POST("", handler.CreatePermission)
		permissionRoutes.GET("", handler.GetPermissions)
		permissionRoutes.GET("/:id", handler.GetPermissionByID)
		permissionRoutes.PUT("/:id", handler.UpdatePermission)
		permissionRoutes.DELETE("/:id", handler.DeletePermission)
	}
}
