package middleware

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"dokuprime-be/external"
	"dokuprime-be/util"

	"github.com/gin-gonic/gin"
)

const (
	answerValue = "Mohon maaf, saat ini sistem sedang sibuk. Silakan coba kembali beberapa saat lagi."
)

type ConcurrencyLimiter struct {
	limit   int
	current int
	mu      sync.Mutex
}

func NewConcurrencyLimiter(limit int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		limit: limit,
	}
}

func (cl *ConcurrencyLimiter) Acquire() bool {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.current >= cl.limit {
		return false
	}
	cl.current++
	return true
}

func (cl *ConcurrencyLimiter) Release() {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if cl.current > 0 {
		cl.current--
	}
}

func (cl *ConcurrencyLimiter) GetCurrent() int {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	return cl.current
}

func getConcurrencyLimit() int {
	defaultLimit := 150

	limitStr := os.Getenv("LIMIT_CONCURRENT_REQUESTS")
	if limitStr == "" {
		return defaultLimit
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		return defaultLimit
	}

	return limit
}

var globalLimiter = NewConcurrencyLimiter(getConcurrencyLimit())

func ConcurrencyLimitMiddleware(limit int) gin.HandlerFunc {
	limiter := NewConcurrencyLimiter(limit)

	return func(c *gin.Context) {

		if !limiter.Acquire() {
			util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
			c.Abort()
			return
		}
		defer limiter.Release()

		if !globalLimiter.Acquire() {
			util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
			c.Abort()
			return
		}
		defer globalLimiter.Release()

		c.Next()
	}
}

func GlobalConcurrencyLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !globalLimiter.Acquire() {
			util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
			c.Abort()
			return
		}
		defer globalLimiter.Release()

		c.Next()
	}
}

func ChatConcurrencyLimitMiddleware(limit int, externalClient *external.Client) gin.HandlerFunc {
	limiter := NewConcurrencyLimiter(limit)

	return func(c *gin.Context) {

		if !limiter.Acquire() || !globalLimiter.Acquire() {

			var req struct {
				Platform         string `json:"platform"`
				PlatformUniqueID string `json:"platform_unique_id"`
				Query            string `json:"query"`
				ConversationID   string `json:"conversation_id"`
			}

			if err := c.ShouldBindJSON(&req); err == nil {

				if req.Platform != "web" && req.Platform != "" {
					busyResponse := map[string]interface{}{
						"user":               req.PlatformUniqueID,
						"conversation_id":    req.ConversationID,
						"query":              req.Query,
						"rewritten_query":    "",
						"category":           "",
						"question_category":  []string{},
						"answer":             answerValue,
						"citations":          []interface{}{},
						"is_helpdesk":        false,
						"is_answered":        false,
						"platform":           req.Platform,
						"platform_unique_id": req.PlatformUniqueID,
						"question_id":        0,
						"answer_id":          0,
					}

					if err := externalClient.SendMessageToAPI(busyResponse); err != nil {
						log.Printf("Failed to send busy notification: %v", err)
					} else {
						log.Printf("✅ Sent busy notification to user %s on platform %s", req.PlatformUniqueID, req.Platform)
					}
				}
			}

			util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
			c.Abort()
			return
		}

		defer func() {
			limiter.Release()
			globalLimiter.Release()
		}()

		c.Next()
	}
}
