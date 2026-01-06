package middleware

import (
	"net/http"
	"os"
	"strconv"
	"sync"

	"dokuprime-be/util"

	"github.com/gin-gonic/gin"
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
			util.ErrorResponse(c, http.StatusTooManyRequests, "Sistem sedang sibuk")
			c.Abort()
			return
		}
		defer limiter.Release()

		if !globalLimiter.Acquire() {
			util.ErrorResponse(c, http.StatusTooManyRequests, "Sistem sedang sibuk")
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
			util.ErrorResponse(c, http.StatusTooManyRequests, "Sistem sedang sibuk")
			c.Abort()
			return
		}
		defer globalLimiter.Release()

		c.Next()
	}
}
