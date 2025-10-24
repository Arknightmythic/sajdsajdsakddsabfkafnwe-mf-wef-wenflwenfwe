package middleware

import (
	"net/http"
	"strings"

	"dokuprime-be/auth"
	"dokuprime-be/util"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		token, err := c.Cookie("access_token")

		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				util.ErrorResponse(c, http.StatusUnauthorized, "Authorization header required")
				c.Abort()
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				util.ErrorResponse(c, http.StatusUnauthorized, "Invalid authorization format")
				c.Abort()
				return
			}
			token = parts[1]
		}

		claims, err := auth.ValidateToken(token)
		if err != nil {
			util.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("account_type", claims.AccountType)
		c.Next()
	}
}
