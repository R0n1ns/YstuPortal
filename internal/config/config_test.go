package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	_ = os.Unsetenv("JWT_SECRET")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Port == "" {
		t.Fatalf("expected default port")
	}
}

func TestLoadConfigCustom(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_TTL", "2h")
	t.Setenv("RATE_LIMIT_MAX", "5")
	t.Setenv("RATE_LIMIT_WINDOW", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.JWTTTL != 2*time.Hour {
		t.Fatalf("expected 2h jwt ttl")
	}
	if cfg.RateLimitMax != 5 {
		t.Fatalf("expected rate limit max 5")
	}
	if cfg.RateLimitWindow != 30*time.Second {
		t.Fatalf("expected rate limit window 30s")
	}
}
