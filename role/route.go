package role

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"dokuprime-be/middleware"
	"dokuprime-be/permission"
	"dokuprime-be/team"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repoRole := NewRoleRepository(db)
	repoTeam := team.NewTeamRepository(db)
	repoPermission := permission.NewPermissionRepository(db)
	service := NewRoleService(repoRole, repoTeam, repoPermission)
	handler := NewRoleHandler(service)

	roleGroup := r.Group("/api/roles")

	roleGroup.Use(middleware.AuthMiddleware())
	{
		roleGroup.POST("", handler.Create)
		roleGroup.GET("", handler.GetAll)
		roleGroup.GET("/:id", handler.GetByID)
		roleGroup.PUT("/:id", handler.Update)
		roleGroup.DELETE("/:id", handler.Delete)
	}
}
