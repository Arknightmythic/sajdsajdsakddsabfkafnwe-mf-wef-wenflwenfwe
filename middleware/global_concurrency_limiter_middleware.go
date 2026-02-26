package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/util"

	"github.com/gin-gonic/gin"
)

const (
	answerValue = "Mohon maaf, saat ini terdapat peningkatan jumlah pesan yang masuk. Silakan kirim ulang pesan Anda beberapa saat lagi. Terimakasih."
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

type ConversationChecker interface {
	IsHelpdeskConversation(conversationID string) (bool, error)
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

	if config.AppConfig == nil {
		return defaultLimit
	}

	limit := config.AppConfig.LimitConcurrentRequests
	if limit == 0 {
		return defaultLimit
	}

	return limit
}

var (
	globalLimiter     *ConcurrencyLimiter
	globalLimiterOnce sync.Once
)

func InitGlobalLimiter() {
	globalLimiterOnce.Do(func() {
		limit := getConcurrencyLimit()
		globalLimiter = NewConcurrencyLimiter(limit)
		log.Printf("[ConcurrencyLimiter] Global limiter initialized with limit: %d", limit)
	})
}

func getGlobalLimiter() *ConcurrencyLimiter {
	globalLimiterOnce.Do(func() {
		limit := getConcurrencyLimit()
		globalLimiter = NewConcurrencyLimiter(limit)
		log.Printf("[ConcurrencyLimiter] WARN: Global limiter initialized lazily with limit: %d. Call InitGlobalLimiter() after config load.", limit)
	})
	return globalLimiter
}

func ConcurrencyLimitMiddleware(limit int) gin.HandlerFunc {
	limiter := NewConcurrencyLimiter(limit)

	return func(c *gin.Context) {
		if !limiter.Acquire() {
			util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
			c.Abort()
			return
		}
		defer limiter.Release()

		c.Next()
	}
}

func GlobalConcurrencyLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		gl := getGlobalLimiter()
		if !gl.Acquire() {
			util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
			c.Abort()
			return
		}
		defer gl.Release()

		c.Next()
	}
}

func ChatConcurrencyLimitMiddleware(limit int, externalClient *external.Client, checker ConversationChecker) gin.HandlerFunc {
	limiter := NewConcurrencyLimiter(limit)

	return func(c *gin.Context) {
		gl := getGlobalLimiter()

		var req struct {
			Platform         string `json:"platform"`
			PlatformUniqueID string `json:"platform_unique_id"`
			Query            string `json:"query"`
			ConversationID   string `json:"conversation_id"`
		}

		bodyBytes, readErr := c.GetRawData()
		if readErr != nil {
			log.Printf("[ConcurrencyLimiter] WARN: Failed to read request body: %v", readErr)
		} else {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if len(bodyBytes) > 0 {
				if jsonErr := json.Unmarshal(bodyBytes, &req); jsonErr != nil {
					log.Printf("[ConcurrencyLimiter] WARN: Failed to parse request body: %v", jsonErr)
				}
			}
		}

		if req.ConversationID != "" && checker != nil {
			log.Printf("[ConcurrencyLimiter] Checking conversation ID: %s", req.ConversationID)
			isHelpdesk, err := checker.IsHelpdeskConversation(req.ConversationID)
			log.Printf("[ConcurrencyLimiter] Is Helpdesk: %v", isHelpdesk)
			if err == nil && isHelpdesk {
				c.Next()
				return
			}
			if err != nil {
				log.Printf("[ConcurrencyLimiter] WARN: Failed to check helpdesk status: %v", err)
			}
		}

		localAcquired := limiter.Acquire()
		globalAcquired := false
		if localAcquired {
			globalAcquired = gl.Acquire()
		}

		if !localAcquired || !globalAcquired {
			if localAcquired {
				limiter.Release()
			}
			if globalAcquired {
				gl.Release()
			}

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
					log.Printf("[ConcurrencyLimiter] Failed to send busy notification: %v", err)
				} else {
					log.Printf("[ConcurrencyLimiter] ✅ Sent busy notification to user %s on platform %s", req.PlatformUniqueID, req.Platform)
				}
			}

			util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
			c.Abort()
			return
		}

		defer func() {
			limiter.Release()
			gl.Release()
		}()

		c.Next()
	}
}
