package role

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"dokuprime-be/audit"
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

	auditRepo := audit.NewAuditRepository(db)
	auditService := audit.NewAuditService(auditRepo)

	roleGroup := r.Group("/api/roles")

	roleGroup.Use(middleware.AuthMiddleware())
	roleGroup.Use(middleware.AuditMiddleware(auditService))
	roleGroup.Use(middleware.GlobalConcurrencyLimitMiddleware())
	roleGroup.Use(middleware.TimeoutMiddleware(10 * time.Minute))
	{
		roleGroup.POST("", handler.Create)
		roleGroup.GET("", handler.GetAll)
		roleGroup.GET("/by-team-id/:id", handler.GetRoleByTeamID)
		roleGroup.GET("/:id", handler.GetByID)
		roleGroup.PUT("/:id", handler.Update)
		roleGroup.DELETE("/:id", handler.Delete)
	}
}
