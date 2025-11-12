package middleware

import (
	"net/http"
	"os"

	"dokuprime-be/util"

	"github.com/gin-gonic/gin"
)

func APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("x-api-key")

		if apiKey == "" {
			util.ErrorResponse(c, http.StatusUnauthorized, "API key required")
			c.Abort()
			return
		}

		expectedAPIKey := os.Getenv("X_API_KEY")
		if expectedAPIKey == "" {
			util.ErrorResponse(c, http.StatusInternalServerError, "API key not configured")
			c.Abort()
			return
		}

		if apiKey != expectedAPIKey {
			util.ErrorResponse(c, http.StatusUnauthorized, "Invalid API key")
			c.Abort()
			return
		}

		c.Next()
	}
}
