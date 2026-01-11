package middleware

import (
	"dokuprime-be/util"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func PermissionMiddleware(db *sqlx.DB, requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			util.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
			c.Abort()
			return
		}

		query := `
			SELECT COUNT(*)
			FROM users u
			JOIN roles r ON u.role_id = r.id
			WHERE u.id = $1 AND $2 = ANY(r.permissions)
		`

		var count int
		err := db.Get(&count, query, userID, requiredPermission)
		if err != nil {
			util.ErrorResponse(c, http.StatusInternalServerError, "Failed to verify permissions")
			c.Abort()
			return
		}

		if count == 0 {
			util.ErrorResponse(c, http.StatusForbidden, "Anda tidak memiliki izin untuk fitur ini")
			c.Abort()
			return
		}

		c.Next()
	}
}