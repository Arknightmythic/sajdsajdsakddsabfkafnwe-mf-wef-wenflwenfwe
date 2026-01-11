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

		// PERBAIKAN QUERY:
		// Kita melakukan JOIN ke tabel permissions.
		// Logikanya: Cari user -> cek rolenya -> cek apakah di dalam array permission role tersebut
		// ada ID yang memiliki nama sesuai 'requiredPermission'.
		query := `
			SELECT COUNT(*)
			FROM users u
			JOIN roles r ON u.role_id = r.id
			JOIN permissions p ON p.id::text = ANY(r.permissions)
			WHERE u.id = $1 AND p.name = $2
		`

		var count int
		// $1 = userID, $2 = requiredPermission (contoh: "helpdesk:toggle-helpdesk")
		err := db.Get(&count, query, userID, requiredPermission)
		if err != nil {
			util.ErrorResponse(c, http.StatusInternalServerError, "Failed to verify permissions")
			c.Abort()
			return
		}

		if count == 0 {
			// Anda bisa menyesuaikan pesan error sesuai kebutuhan
			util.ErrorResponse(c, http.StatusForbidden, "Anda tidak memiliki izin untuk fitur ini")
			c.Abort()
			return
		}

		c.Next()
	}
}