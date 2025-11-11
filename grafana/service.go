package grafana

import (
	"context"
	"dokuprime-be/util"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type GrafanaService struct {
	redis *redis.Client
}

func NewGrafanaService(redisClient *redis.Client) *GrafanaService {
	return &GrafanaService{
		redis: redisClient,
	}
}


func (s *GrafanaService) GenerateEmbedURL(req *GenerateEmbedRequest) (string, error) {
	var baseURL string
	var err error

	
	switch req.Category {
	case "daily":
		baseURL = os.Getenv("GRAFANA_EMBED_DAILY_URL")
	case "monthly":
		baseURL = os.Getenv("GRAFANA_EMBED_MONTHLY_URL")
	case "yearly":
		baseURL = os.Getenv("GRAFANA_EMBED_YEARLY_URL")
	case "custom":
		baseURL = os.Getenv("GRAFANA_EMBED_CUSTOM_URL")
		if req.StartDate == "" || req.EndDate == "" {
			return "", fmt.Errorf("start_date and end_date are required for custom category")
		}
	default:
		return "", fmt.Errorf("invalid category specified")
	}

	if baseURL == "" {
		return "", fmt.Errorf("grafana embed URL for category '%s' is not set in .env", req.Category)
	}

	
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL from .env: %w", err)
	}

	
	queryParams := parsedURL.Query()
	queryParams.Set("theme", "light")
	queryParams.Set("refresh", "5s") 

	
	if req.Category == "custom" {
		fromMs, err := parseDateToMillis(req.StartDate)
		if err != nil {
			return "", fmt.Errorf("invalid start_date format: %w", err)
		}
		toMs, err := parseDateToMillis(req.EndDate)
		if err != nil {
			return "", fmt.Errorf("invalid end_date format: %w", err)
		}

		queryParams.Set("from", fmt.Sprintf("%d", fromMs))
		queryParams.Set("to", fmt.Sprintf("%d", toMs))
	}

	parsedURL.RawQuery = queryParams.Encode()
	finalGrafanaURL := parsedURL.String()

	
	token := util.RandString(32)
	key := "grafana_embed_token:" + token
	ctx := context.Background()

	
	err = s.redis.Set(ctx, key, finalGrafanaURL, 1*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store embed token in redis: %w", err)
	}

	return token, nil
}


func parseDateToMillis(dateString string) (int64, error) {
	t, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		return 0, err
	}
	
	tStartOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return tStartOfDay.Unix() * 1000, nil
}