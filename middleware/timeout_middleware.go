// middleware/timeout.go
package middleware

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware - apply this LAST (after all other middlewares)
func TimeoutMiddleware(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a buffered channel to signal when handler finishes
		finish := make(chan struct{})

		// Use atomic flag to track timeout state (thread-safe without mutex)
		var timedOut int32

		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// Run handler in goroutine
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// If panic occurs after timeout, suppress it
					if atomic.LoadInt32(&timedOut) == 0 {
						panic(r) // Re-panic if not timed out
					}
					// Otherwise silently ignore the panic
				}
			}()

			c.Next()
			close(finish)
		}()

		// Wait for either finish or timeout
		select {
		case <-finish:
			// Handler completed normally
			return
		case <-ctx.Done():
			// Set timeout flag BEFORE trying to write response
			atomic.StoreInt32(&timedOut, 1)

			// Check if response was already written by the handler
			// (this can happen in race conditions)
			if c.Writer.Written() {
				c.Abort()
				return
			}

			// Write timeout response
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":   "Request timeout",
				"message": "The request took too long to process",
			})
			c.Abort()
		}
	}
}

// NoTimeoutMiddleware - removes timeout
func NoTimeoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
