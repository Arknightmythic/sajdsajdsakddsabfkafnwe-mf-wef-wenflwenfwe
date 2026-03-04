package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

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
		var req struct {
			Platform         string `json:"platform"`
			PlatformUniqueID string `json:"platform_unique_id"`
			Query            string `json:"query"`
			ConversationID   string `json:"conversation_id"`
		}

		bodyBytes, err := c.GetRawData()
		if err == nil && len(bodyBytes) > 0 {
			if jsonErr := json.Unmarshal(bodyBytes, &req); jsonErr == nil {

				if req.ConversationID != "" && checker != nil {
					isHelpdesk, err := checker.IsHelpdeskConversation(req.ConversationID)
					log.Printf("[ChatLimiter] ConversationID=%s IsHelpdesk=%v err=%v", req.ConversationID, isHelpdesk, err)
					if err == nil && isHelpdesk {
						c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
						c.Next()
						return
					}
				}
			}

			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		if !limiter.Acquire() {
			log.Printf("[ChatLimiter] Limit reached (%d/%d) for user=%s platform=%s", limiter.GetCurrent(), limit, req.PlatformUniqueID, req.Platform)
			sendBusyResponse(c, externalClient, req)
			return
		}

		var releaseOnce sync.Once
		release := func() {
			releaseOnce.Do(func() {
				limiter.Release()
				log.Printf("[ChatLimiter] Slot released for user=%s platform=%s current=%d", req.PlatformUniqueID, req.Platform, limiter.GetCurrent())
			})
		}

		done := make(chan struct{})
		go func() {
			timer := time.NewTimer(45 * time.Second)
			defer timer.Stop()

			select {
			case <-timer.C:
				log.Printf("⚠️ [ChatLimiter] Timeout after 45s for user=%s platform=%s — force releasing slot", req.PlatformUniqueID, req.Platform)
				release()

				if req.Platform != "web" && req.Platform != "" {
					payload := buildBusyPayload(req)
					if err := externalClient.SendMessageToAPI(payload); err != nil {
						log.Printf("[ChatLimiter] Failed to send timeout notification: %v", err)
					} else {
						log.Printf("✅ [ChatLimiter] Sent timeout notification to user=%s platform=%s", req.PlatformUniqueID, req.Platform)
					}
				} else {
					if !c.Writer.Written() {
						util.ErrorResponse(c, http.StatusGatewayTimeout, answerValue)
					}
				}

			case <-done:

			}
		}()

		c.Next()

		close(done)
		release()
	}
}

func buildBusyPayload(req struct {
	Platform         string `json:"platform"`
	PlatformUniqueID string `json:"platform_unique_id"`
	Query            string `json:"query"`
	ConversationID   string `json:"conversation_id"`
}) map[string]interface{} {
	return map[string]interface{}{
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
}

func sendBusyResponse(c *gin.Context, externalClient *external.Client, req struct {
	Platform         string `json:"platform"`
	PlatformUniqueID string `json:"platform_unique_id"`
	Query            string `json:"query"`
	ConversationID   string `json:"conversation_id"`
}) {
	if req.Platform != "web" && req.Platform != "" {
		payload := buildBusyPayload(req)
		if err := externalClient.SendMessageToAPI(payload); err != nil {
			log.Printf("[ChatLimiter] Failed to send busy notification: %v", err)
		} else {
			log.Printf("✅ [ChatLimiter] Sent busy notification to user=%s platform=%s", req.PlatformUniqueID, req.Platform)
		}
	}

	util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
	c.Abort()
}
