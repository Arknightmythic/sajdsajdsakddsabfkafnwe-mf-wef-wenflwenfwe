package grafana

import (
	"context"
	"dokuprime-be/util"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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


func (h *GrafanaHandler) GenerateEmbedURL(ctx *gin.Context) {
	var req GenerateEmbedRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request payload")
		return
	}

	token, err := h.service.GenerateEmbedURL(&req)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, err.Error())
		return
	}

	
	scheme := "https"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, ctx.Request.Host)
	viewURL := fmt.Sprintf("%s/api/grafana/view-embed?token=%s", baseURL, token)

	util.SuccessResponse(ctx, "Grafana embed URL generated successfully", gin.H{
		"url": viewURL,
	})
}


func (h *GrafanaHandler) ViewEmbed(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Token is required")
		return
	}

	key := "grafana_embed_token:" + token
	ctxRedis := context.Background()

	
	grafanaURL, err := h.redis.Get(ctxRedis, key).Result()
	if err == redis.Nil {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to validate token: %v", err)
		util.ErrorResponse(ctx, http.StatusInternalServerError, errorMsg)
		return
	}

	
	h.redis.Del(ctxRedis, key)

	
	ctx.Redirect(http.StatusFound, grafanaURL)
}