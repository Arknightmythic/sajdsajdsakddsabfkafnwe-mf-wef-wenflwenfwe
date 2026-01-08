package middleware

import (
	"dokuprime-be/config"
	"dokuprime-be/util"
	"log"
	"net/http"

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

		expectedAPIKey := config.AppConfig.XAPIKey
		log.Println(apiKey, "is the expected API key")
		log.Println(expectedAPIKey, "is the provided API key")

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
