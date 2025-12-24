package grafana

import (
	"context"
	"dokuprime-be/util"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	grafanaCookieName = "grafana_view_token"
)

type GrafanaHandler struct {
	service *GrafanaService
	redis   *redis.Client
}

func NewGrafanaHandler(service *GrafanaService, redisClient *redis.Client) *GrafanaHandler {
	return &GrafanaHandler{
		service: service,
		redis:   redisClient,
	}
}

// GenerateEmbedURL: Menanam Cookie & Return Clean URL
func (h *GrafanaHandler) GenerateEmbedURL(c *gin.Context) {
	// Gunakan struct request yang ada di entity.go (atau definisikan inline jika perlu, 
    // tapi karena Service minta *GenerateEmbedRequest, kita pakai itu)
	var req GenerateEmbedRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 1. Panggil Service yang sudah ada
	token, err := h.service.GenerateEmbedURL(&req)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 2. Logika Secure Cookie
	isSecure := c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https"
	if envSecure := os.Getenv("COOKIE_SECURE"); envSecure != "" {
		isSecure = envSecure == "true"
	}

	httpOnly := getEnvBool("COOKIE_HTTP_ONLY", true)
	domain := os.Getenv("COOKIE_DOMAIN")
	
	path := "/api/grafana/view-embed"
	
	sameSiteEnv := os.Getenv("COOKIE_SAME_SITE")
	c.SetSameSite(getSameSiteMode(sameSiteEnv))

	// 3. Set Cookie (MaxAge 1 menit sesuai dengan service redis TTL)
	c.SetCookie(grafanaCookieName, token, 60, path, domain, isSecure, httpOnly)

	// 4. Return Clean URL
	scheme := "https"
	if isSecure {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	viewURL := fmt.Sprintf("%s/api/grafana/view-embed", baseURL)

	util.SuccessResponse(c, "Embed URL generated successfully", gin.H{
		"url": viewURL,
	})
}

// ViewEmbed: Membaca Cookie -> Redirect ke Grafana Asli
func (h *GrafanaHandler) ViewEmbed(c *gin.Context) {
	// 1. Ambil Token dari Cookie
	token, err := c.Cookie(grafanaCookieName)
	if err != nil || token == "" {
		util.ErrorResponse(c, http.StatusUnauthorized, "Missing access token (cookie required)")
		return
	}

	// 2. Hapus Cookie (One-time use)
	domain := os.Getenv("COOKIE_DOMAIN")
	path := os.Getenv("COOKIE_PATH")
	if path == "" {
		path = "/api/grafana/view-embed"
	}
	c.SetCookie(grafanaCookieName, "", -1, path, domain, false, true)

	// 3. Validasi ke Redis
	ctxRedis := context.Background()
	key := "grafana_embed_token:" + token 

	// Ambil URL Asli Grafana yang disimpan Service
	finalGrafanaURL, err := h.redis.Get(ctxRedis, key).Result()
	if err == redis.Nil {
		util.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to validate token")
		return
	}

	// Hapus token dari Redis
	h.redis.Del(ctxRedis, key)

	// 4. REDIRECT ke URL Grafana
	// Kita tidak mem-proxy konten (GetEmbedContent), melainkan melempar user ke URL yang sudah dibuild service.
	c.Redirect(http.StatusFound, finalGrafanaURL)
}

// Helper Functions
func getEnvBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val == "true"
}

func getSameSiteMode(mode string) http.SameSite {
	switch strings.ToLower(mode) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}