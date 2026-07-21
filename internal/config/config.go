package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment     string
	DemoMode        bool
	DatabaseURL     string
	Port            string
	JWTSecret       string
	JWTTTL          time.Duration
	JWTIssuer       string
	JWTAudience     string
	AllowedOrigins  []string
	RedisURL        string
	CacheTTL        time.Duration
	RateLimitMax    int
	RateLimitWindow time.Duration
	SwaggerEnabled  bool
	CookieSecure    bool
	CookieSameSite  string
	CookieDomain    string
	UpstreamBaseURL string
	UpstreamCode    string
	UpstreamTimeout time.Duration
	SessionTTL      time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment:     getEnv("APP_ENV", "development"),
		DemoMode:        getEnvBool("DEMO_MODE", false),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://user:user@localhost:5432/ystu_db?sslmode=disable"),
		Port:            getEnv("PORT", "8080"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me"),
		JWTTTL:          getEnvDuration("JWT_TTL", 24*time.Hour),
		JWTIssuer:       getEnv("JWT_ISSUER", "ystuportal"),
		JWTAudience:     getEnv("JWT_AUDIENCE", "ystuportal-web"),
		AllowedOrigins:  splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:8080,http://127.0.0.1:8080,http://localhost:8081,http://127.0.0.1:8081")),
		RedisURL:        getEnv("REDIS_URL", ""),
		CacheTTL:        getEnvDuration("CACHE_TTL", 10*time.Minute),
		RateLimitMax:    getEnvInt("RATE_LIMIT_MAX", 10),
		RateLimitWindow: getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		SwaggerEnabled:  getEnvBool("SWAGGER_ENABLED", true),
		CookieSecure:    getEnvBool("COOKIE_SECURE", false),
		CookieSameSite:  getEnv("COOKIE_SAMESITE", "lax"),
		CookieDomain:    getEnv("COOKIE_DOMAIN", ""),
		UpstreamBaseURL: getEnv("YSTU_BASE_URL", "https://www.ystu.ru"),
		UpstreamCode:    getEnv("YSTU_PORTAL_CODE", "330785001"),
		UpstreamTimeout: getEnvDuration("YSTU_HTTP_TIMEOUT", 15*time.Second),
		SessionTTL:      getEnvDuration("YSTU_SESSION_TTL", 24*time.Hour),
		ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) validate() error {
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is empty")
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET is empty")
	}
	if strings.EqualFold(cfg.Environment, "production") && cfg.JWTSecret == "change-me" {
		return fmt.Errorf("JWT_SECRET must be changed in production")
	}
	if strings.EqualFold(cfg.Environment, "production") && len(cfg.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must contain at least 32 characters in production")
	}
	if cfg.JWTTTL <= 0 || cfg.CacheTTL <= 0 || cfg.RateLimitWindow <= 0 {
		return fmt.Errorf("JWT_TTL, CACHE_TTL and RATE_LIMIT_WINDOW must be positive")
	}
	if cfg.RateLimitMax <= 0 {
		return fmt.Errorf("RATE_LIMIT_MAX must be positive")
	}
	if cfg.UpstreamTimeout <= 0 || cfg.SessionTTL <= 0 {
		return fmt.Errorf("YSTU_HTTP_TIMEOUT and YSTU_SESSION_TTL must be positive")
	}
	if cfg.CookieSameSite != "lax" && cfg.CookieSameSite != "strict" && cfg.CookieSameSite != "none" {
		return fmt.Errorf("COOKIE_SAMESITE must be lax, strict or none")
	}
	if cfg.CookieSameSite == "none" && !cfg.CookieSecure {
		return fmt.Errorf("COOKIE_SECURE must be true when COOKIE_SAMESITE=none")
	}
	return nil
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
