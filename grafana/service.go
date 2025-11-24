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
	queryParams.Set("kiosk", "true")
	queryParams.Set("theme", "light")
	queryParams.Set("refresh", "5s")
	queryParams.Set("timezone", "browser")
	switch req.Category {
	case "daily":
		
		queryParams.Set("from", "now/d")
		queryParams.Set("to", "now")
	case "monthly":
		
		queryParams.Set("from", "now/M")
		queryParams.Set("to", "now")
	case "yearly":
		
		queryParams.Set("from", "now/y")
		queryParams.Set("to", "now")
	case "custom":
		fromTime, toTime, err := parseCustomDatesToUTC(req.StartDate, req.EndDate)
		if err != nil {
			return "", err
		}
		queryParams.Set("from", fromTime)
		queryParams.Set("to", toTime)
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


func parseCustomDatesToUTC(startDateStr, endDateStr string) (string, string, error) {
	layout := "2006-01-02"
	loc := time.FixedZone("WIB", 7*60*60)
	tStart, err := time.ParseInLocation(layout, startDateStr, loc)
	if err != nil {
		return "", "", fmt.Errorf("invalid start_date format: %w", err)
	}
	
	tEnd, err := time.ParseInLocation(layout, endDateStr, loc)
	if err != nil {
		return "", "", fmt.Errorf("invalid end_date format: %w", err)
	}
	tEnd = time.Date(tEnd.Year(), tEnd.Month(), tEnd.Day(), 23, 59, 59, 0, loc)
	isoLayout := "2006-01-02T15:04:05.000Z"

	return tStart.UTC().Format(isoLayout), tEnd.UTC().Format(isoLayout), nil
}