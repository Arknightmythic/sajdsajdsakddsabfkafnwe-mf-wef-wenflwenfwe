// util/cookie_helper.go (atau tambahkan di util/response.go)
package util

import (
	"net/http"
	"os"
	"strings"
)

func GetEnvBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val == "true"
}

func GetSameSiteMode(mode string) http.SameSite {
	switch strings.ToLower(mode) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}