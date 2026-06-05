package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL     string
	Port            string
	JWTSecret       string
	JWTTTL          time.Duration
	AllowedOrigins  []string
	RedisURL        string
	CacheTTL        time.Duration
	RateLimitMax    int
	RateLimitWindow time.Duration
	SwaggerEnabled  bool
	CookieSecure    bool
	CookieSameSite  string
	CookieDomain    string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://user:user@localhost:5432/ystu_db?sslmode=disable"),
		Port:            getEnv("PORT", "8080"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me"),
		JWTTTL:          getEnvDuration("JWT_TTL", 24*time.Hour),
		AllowedOrigins:  splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:8080,http://127.0.0.1:8080,http://localhost:8081,http://127.0.0.1:8081")),
		RedisURL:        getEnv("REDIS_URL", ""),
		CacheTTL:        getEnvDuration("CACHE_TTL", 10*time.Minute),
		RateLimitMax:    getEnvInt("RATE_LIMIT_MAX", 10),
		RateLimitWindow: getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		SwaggerEnabled:  getEnvBool("SWAGGER_ENABLED", true),
		CookieSecure:    getEnvBool("COOKIE_SECURE", false),
		CookieSameSite:  getEnv("COOKIE_SAMESITE", "lax"),
		CookieDomain:    getEnv("COOKIE_DOMAIN", ""),
	}

	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
