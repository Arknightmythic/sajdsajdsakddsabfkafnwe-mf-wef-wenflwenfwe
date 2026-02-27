package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBDriver   string
	DBSchema   string

	AzureADClientID              string
	AzureADClientSecret          string
	AzureADTenantID              string
	AzureADRedirectURI           string
	FrontendAzureAuthCallbackURI string
	BackendURI                   string

	ServerPort string

	JWTSecret        string
	JWTAccessExpiry  string
	JWTRefreshExpiry string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       string

	MaxFileSizeAllowed int64
	UploadPath         string

	AllowedOrigins string
	BcryptSalt     string
	XAPIKey        string

	SuperadminPassword string

	ExternalAPIBaseURL string
	ExtractionWorkers  int
	MessagesAPIURL     string
	MessagesAPIKey     string

	CookieSecure             bool
	CookieHTTPOnly           bool
	CookieSameSite           string
	CookiePath               string
	CookieDomain             string
	CookieAccessTokenMaxAge  int
	CookieRefreshTokenMaxAge int

	GrafanaEmbedDailyURL   string
	GrafanaEmbedMonthlyURL string
	GrafanaEmbedYearlyURL  string
	GrafanaEmbedCustomURL  string

	WebSocketURL       string
	WebSocketSecretKey string

	HelpdeskQueuePeriodMinutes time.Duration

	LimitConcurrentRequests     int
	LimitConcurrentChatRequests int

	CronCleanupSchedule string

	DefaultTeam string
}

var AppConfig *Config

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables.")
	}

	cfg := &Config{

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "dokuprime"),
		DBDriver:   getEnv("DB_DRIVER", "postgres"),
		DBSchema:   getEnv("DB_SCHEMA", "bkpm"),

		AzureADClientID:              getEnv("AZURE_AD_CLIENT_ID", ""),
		AzureADClientSecret:          getEnv("AZURE_AD_CLIENT_SECRET", ""),
		AzureADTenantID:              getEnv("AZURE_AD_TENANT_ID", ""),
		AzureADRedirectURI:           getEnv("AZURE_AD_REDIRECT_URI", "/api/authazure/callback"),
		FrontendAzureAuthCallbackURI: getEnv("FRONTEND_AZURE_AUTH_CALLBACK_URI", "http://localhost:5173/auth-microsoft/callback"),
		BackendURI:                   getEnv("BACKEND_URI", "http://localhost:8000"),

		ServerPort: getEnv("SERVER_PORT", "8000"),

		JWTSecret:        getEnv("JWT_SECRET", "dokuprime-jaya-jaya-jaya"),
		JWTAccessExpiry:  getEnv("JWT_ACCESS_EXPIRY", "10h"),
		JWTRefreshExpiry: getEnv("JWT_REFRESH_EXPIRY", "168h"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnv("REDIS_DB", "0"),

		MaxFileSizeAllowed: getEnvAsInt64("MAX_FILE_SIZE_ALLOWED", 70),
		UploadPath:         getEnv("UPLOAD_PATH", "./uploads/documents"),

		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		BcryptSalt:     getEnv("BCRYPT_SALT", "dokuprime-jaya-jaya-jaya"),
		XAPIKey:        getEnv("X_API_KEY", "TestDeployDevBaruENV"),

		SuperadminPassword: getEnv("SUPERADMIN_PASSWORD", "superadmin"),

		ExternalAPIBaseURL: getEnv("EXTERNAL_API_BASE_URL", "http://localhost:8090"),
		ExtractionWorkers:  getEnvAsInt("EXTRACTION_WORKERS", 5),
		MessagesAPIURL:     getEnv("MESSAGES_API_URL", "http://172.16.9.171:9798"),
		MessagesAPIKey:     getEnv("MESSAGES_API_KEY", "BangJorAwesome"),

		CookieSecure:             getEnvAsBool("COOKIE_SECURE", false),
		CookieHTTPOnly:           getEnvAsBool("COOKIE_HTTP_ONLY", true),
		CookieSameSite:           getEnv("COOKIE_SAME_SITE", "Lax"),
		CookiePath:               getEnv("COOKIE_PATH", "/"),
		CookieDomain:             getEnv("COOKIE_DOMAIN", ""),
		CookieAccessTokenMaxAge:  getEnvAsInt("COOKIE_ACCESS_TOKEN_MAX_AGE", 3600),
		CookieRefreshTokenMaxAge: getEnvAsInt("COOKIE_REFRESH_TOKEN_MAX_AGE", 604800),

		GrafanaEmbedDailyURL:   getEnv("GRAFANA_EMBED_DAILY_URL", ""),
		GrafanaEmbedMonthlyURL: getEnv("GRAFANA_EMBED_MONTHLY_URL", ""),
		GrafanaEmbedYearlyURL:  getEnv("GRAFANA_EMBED_YEARLY_URL", ""),
		GrafanaEmbedCustomURL:  getEnv("GRAFANA_EMBED_CUSTOM_URL", ""),

		WebSocketURL:       getEnv("WEBSOCKET_URL", "ws://localhost:8080"),
		WebSocketSecretKey: getEnv("WEBSOCKET_SECRET_KEY", "bkpm-jaya-jaya-jaya"),

		HelpdeskQueuePeriodMinutes: time.Duration(getEnvAsInt("HELPDESK_QUEUE_PERIOD_MINUTES", 15)) * time.Minute,

		LimitConcurrentRequests:     getEnvAsInt("LIMIT_CONCURRENT_REQUESTS", 150),
		LimitConcurrentChatRequests: getEnvAsInt("LIMIT_CONCURRENT_CHAT_REQUESTS", 2),

		CronCleanupSchedule: getEnv("CRON_AUDIT_CLEANUP", "0 0 1 * * *"),

		DefaultTeam: getEnv("DEFAULT_TEAM", "finance"),
	}

	AppConfig = cfg

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func GetConfig() *Config {
	if AppConfig == nil {
		return LoadConfig()
	}
	return AppConfig
}
