package middleware

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
)

func PlatformContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.ContentType() != "application/json" {
			c.Next()
			return
		}

		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {

				var body map[string]interface{}
				if json.Unmarshal(bodyBytes, &body) == nil {
					if platformID, exists := body["platform_unique_id"]; exists {
						if pid, ok := platformID.(string); ok && pid != "" {
							c.Set("platform_unique_id", pid)
						}
					}

					if platform, exists := body["platform"]; exists {
						if p, ok := platform.(string); ok && p != "" && p != "web" {

							c.Set("is_platform_user", true)
						}
					}
				}

				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		c.Next()
	}
}
