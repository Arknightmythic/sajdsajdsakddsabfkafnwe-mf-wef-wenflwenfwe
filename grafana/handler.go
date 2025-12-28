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

func (h *GrafanaHandler) GenerateEmbedURL(c *gin.Context) {

	var req GenerateEmbedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	token, err := h.service.GenerateEmbedURL(&req)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	isSecure := c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https"
	if envSecure := os.Getenv("COOKIE_SECURE"); envSecure != "" {
		isSecure = envSecure == "true"
	}

	httpOnly := getEnvBool("COOKIE_HTTP_ONLY", true)
	domain := os.Getenv("COOKIE_DOMAIN")

	path := "/api/grafana/view-embed"

	sameSiteEnv := os.Getenv("COOKIE_SAME_SITE")
	c.SetSameSite(getSameSiteMode(sameSiteEnv))

	c.SetCookie(grafanaCookieName, token, 60, path, domain, isSecure, httpOnly)

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

func (h *GrafanaHandler) ViewEmbed(c *gin.Context) {

	token, err := c.Cookie(grafanaCookieName)
	if err != nil || token == "" {
		util.ErrorResponse(c, http.StatusUnauthorized, "Missing access token (cookie required)")
		return
	}

	domain := os.Getenv("COOKIE_DOMAIN")
	path := os.Getenv("COOKIE_PATH")
	if path == "" {
		path = "/api/grafana/view-embed"
	}
	c.SetCookie(grafanaCookieName, "", -1, path, domain, false, true)

	ctxRedis := context.Background()
	key := "grafana_embed_token:" + token

	finalGrafanaURL, err := h.redis.Get(ctxRedis, key).Result()
	if err == redis.Nil {
		util.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to validate token")
		return
	}

	h.redis.Del(ctxRedis, key)

	c.Redirect(http.StatusFound, finalGrafanaURL)
}

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
