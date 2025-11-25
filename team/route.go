package team

import (
	"dokuprime-be/middleware"
	"dokuprime-be/permission"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB) {
	repo := NewTeamRepository(db)
	repoPermission := permission.NewPermissionRepository(db)
	service := NewTeamService(repo, repoPermission)
	handler := NewTeamHandler(service)

	teamRoutes := r.Group("/api/teams")

	teamRoutes.Use(middleware.AuthMiddleware())
	{
		teamRoutes.POST("", handler.CreateTeam)
		teamRoutes.GET("", handler.GetAll)
		teamRoutes.GET("/:id", handler.GetTeamByID)
		teamRoutes.PUT("/:id", handler.UpdateTeam)
		teamRoutes.DELETE("/:id", handler.DeleteTeam)
	}
}
