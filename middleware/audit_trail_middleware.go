package middleware

import (
	"bytes"
	"dokuprime-be/audit"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func AuditMiddleware(auditService *audit.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {

		if shouldSkipAudit(c.Request.URL.Path) {
			c.Next()
			return
		}

		startTime := time.Now()

		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestBody = string(bodyBytes)

				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = writer

		c.Next()

		duration := time.Since(startTime).Milliseconds()

		var userID *string
		var userType *string

		if id, exists := c.Get("user_id"); exists {
			if uid, ok := id.(int64); ok {
				uidStr := fmt.Sprintf("%d", uid)
				userID = &uidStr
				uType := "authenticated"
				userType = &uType
			}
		}

		if userID == nil {
			if platformID, exists := c.Get("platform_unique_id"); exists {
				if pid, ok := platformID.(string); ok && pid != "" {
					userID = &pid
					uType := "platform"
					userType = &uType
				}
			}
		}

		responseBody := writer.body.String()

		if len(requestBody) > 5000 {
			requestBody = requestBody[:5000] + "... (truncated)"
		}
		if len(responseBody) > 5000 {
			responseBody = responseBody[:5000] + "... (truncated)"
		}

		var errorMessage *string
		if len(c.Errors) > 0 {
			errMsg := c.Errors.String()
			errorMessage = &errMsg
		}

		action := determineAction(c.Request.Method, c.Request.URL.Path)
		resource := extractResource(c.Request.URL.Path)

		statusCode := c.Writer.Status()
		auditService.Log(audit.CreateAuditLogRequest{
			UserID:       userID,
			UserType:     userType,
			Action:       action,
			Resource:     resource,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			StatusCode:   &statusCode,
			RequestBody:  stringPtr(requestBody),
			ResponseBody: stringPtr(responseBody),
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			ErrorMessage: errorMessage,
			Duration:     &duration,
		})
	}
}

func shouldSkipAudit(path string) bool {
	skipPaths := []string{
		"/health",
		"/metrics",
		"/ping",
	}

	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

func determineAction(method, path string) string {
	switch method {
	case "POST":
		return "CREATE"
	case "GET":
		return "READ"
	case "PUT", "PATCH":
		return "UPDATE"
	case "DELETE":
		return "DELETE"
	default:
		return method
	}
}

func extractResource(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "unknown"
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
