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

func getGlobalLimiter() *ConcurrencyLimiter {
	globalLimiterOnce.Do(func() {
		globalLimiter = NewConcurrencyLimiter(getConcurrencyLimit())
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

		bodyBytes, err := c.GetRawData()
		if err == nil && len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &req); err == nil {
				if req.ConversationID != "" && checker != nil {
					log.Println(req.ConversationID, "Conversation ID")
					isHelpdesk, err := checker.IsHelpdeskConversation(req.ConversationID)
					log.Println("Is Helpdesk : ", isHelpdesk)
					if err == nil && isHelpdesk {
						c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
						c.Next()
						return
					}
				}
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		localAcquired := limiter.Acquire()
		if !localAcquired {
			sendBusyResponse(c, externalClient, req)
			return
		}

		globalAcquired := gl.Acquire()
		if !globalAcquired {
			limiter.Release()
			sendBusyResponse(c, externalClient, req)
			return
		}

		released := make(chan struct{})

		defer func() {
			select {
			case <-released:

			default:
				limiter.Release()
				gl.Release()
			}
		}()

		go func() {
			timer := time.NewTimer(45 * time.Second)
			defer timer.Stop()

			select {
			case <-timer.C:

				select {
				case <-released:

					return
				default:
				}

				log.Printf("⚠️ Request timeout after 45s for user %s on platform %s — force releasing slot", req.PlatformUniqueID, req.Platform)

				limiter.Release()
				gl.Release()
				close(released)

				c.Abort()

				if req.Platform != "web" && req.Platform != "" {
					busyResponse := buildBusyPayload(req)
					if err := externalClient.SendMessageToAPI(busyResponse); err != nil {
						log.Printf("Failed to send timeout notification: %v", err)
					} else {
						log.Printf("✅ Sent timeout notification to user %s on platform %s", req.PlatformUniqueID, req.Platform)
					}
				} else {

					if !c.Writer.Written() {
						util.ErrorResponse(c, http.StatusGatewayTimeout, answerValue)
					}
				}

			case <-released:

			}
		}()

		c.Next()

		select {
		case <-released:

		default:
			close(released)
		}
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
			log.Printf("Failed to send busy notification: %v", err)
		} else {
			log.Printf("✅ Sent busy notification to user %s on platform %s", req.PlatformUniqueID, req.Platform)
		}
	}

	util.ErrorResponse(c, http.StatusTooManyRequests, answerValue)
	c.Abort()
}
