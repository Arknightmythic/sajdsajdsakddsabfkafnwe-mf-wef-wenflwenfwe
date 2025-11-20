package user

import (
	"dokuprime-be/middleware"
	"dokuprime-be/permission"
	"dokuprime-be/role"
	"dokuprime-be/team"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, db *sqlx.DB, redisClient *redis.Client) {
	repo := NewUserRepository(db)
	repoRole := role.NewRoleRepository(db)
	repoTeam := team.NewTeamRepository(db)
	repoPermission := permission.NewPermissionRepository(db)
	serviceRole := role.NewRoleService(repoRole, repoTeam, repoPermission)
	service := NewUserService(repo, redisClient, serviceRole)
	handler := NewUserHandler(service)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", handler.Login)
		authGroup.POST("/refresh", handler.RefreshToken)
		authGroup.POST("/logout", middleware.AuthMiddleware(), handler.Logout)
		authGroup.POST("/logout-all", middleware.AuthMiddleware(), handler.LogoutAllSessions)
		authGroup.GET("/sessions", middleware.AuthMiddleware(), handler.GetActiveSessions)
	}

	userGroup := r.Group("/api/users")
	userGroup.Use(middleware.AuthMiddleware())
	{
		userGroup.POST("/", handler.CreateUser)
		userGroup.GET("/", handler.GetUsers)
		userGroup.GET("/:id", handler.GetUserByID)
		userGroup.PUT("/:id", handler.UpdateUser)
		userGroup.DELETE("/:id", handler.DeleteUser)
	}
}
