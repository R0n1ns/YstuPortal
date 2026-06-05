package integration

import (
	"YstuPortal/internal/config"
	"YstuPortal/internal/delivery/api"
	"YstuPortal/internal/domain"
	"YstuPortal/internal/logic"
	rediscache "YstuPortal/internal/repository/cache/redis"
	"YstuPortal/internal/repository/userStorage/db"
	"YstuPortal/internal/server"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testProvider struct {
	user        domain.User
	estimations []domain.Subject
}

func (p *testProvider) GetUser(ctx context.Context, username, password string) (*domain.User, error) {
	user := p.user
	user.UserName = username
	return &user, nil
}

func (p *testProvider) GetEstimations(ctx context.Context) (*[]domain.Subject, error) {
	data := append([]domain.Subject(nil), p.estimations...)
	return &data, nil
}

func TestAuthLoginAndAdminPing(t *testing.T) {
	ctx := context.Background()
	dbURL, redisURL, cleanup := setupContainers(t, ctx)
	defer cleanup()

	cfg := config.Config{
		DatabaseURL:     dbURL,
		Port:            "0",
		JWTSecret:       "test-secret",
		JWTTTL:          time.Hour,
		AllowedOrigins:  []string{"http://localhost"},
		RedisURL:        redisURL,
		CacheTTL:        time.Minute,
		RateLimitMax:    100,
		RateLimitWindow: time.Minute,
		SwaggerEnabled:  false,
		CookieSecure:    false,
		CookieSameSite:  "lax",
		CookieDomain:    "",
	}

	provider := &testProvider{user: domain.User{
		FirstName:  "Test",
		LastName:   "User",
		Mail:       "user@example.com",
		Password:   "hash",
		Registered: true,
		Role:       "admin",
	}}
	storage := db.NewUserStorage(cfg.DatabaseURL)
	defer storage.Close()

	cache, err := rediscache.NewEstimationsCache(cfg.RedisURL)
	if err != nil {
		t.Fatalf("redis cache: %v", err)
	}
	defer func() { _ = cache.Close() }()

	manager, err := logic.NewUserManagerWithCache(provider, storage, cache, cfg.CacheTTL)
	if err != nil {
		t.Fatalf("user manager: %v", err)
	}

	app := server.New(cfg, *manager)

	body := []byte(`{"login":"student","password":"pass1234"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := app.Test(loginReq)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", loginResp.StatusCode)
	}

	cookies := loginResp.Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected jwt cookie")
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/api/user/admin/ping", nil)
	adminReq.AddCookie(cookies[0])
	adminResp, err := app.Test(adminReq)
	if err != nil {
		t.Fatalf("admin request: %v", err)
	}
	if adminResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", adminResp.StatusCode)
	}
}

func TestValidationErrorFormat(t *testing.T) {
	ctx := context.Background()
	dbURL, redisURL, cleanup := setupContainers(t, ctx)
	defer cleanup()

	cfg := config.Config{
		DatabaseURL:     dbURL,
		Port:            "0",
		JWTSecret:       "test-secret",
		JWTTTL:          time.Hour,
		AllowedOrigins:  []string{"http://localhost"},
		RedisURL:        redisURL,
		CacheTTL:        time.Minute,
		RateLimitMax:    100,
		RateLimitWindow: time.Minute,
		SwaggerEnabled:  false,
		CookieSecure:    false,
		CookieSameSite:  "lax",
		CookieDomain:    "",
	}

	provider := &testProvider{user: domain.User{
		FirstName:  "Test",
		LastName:   "User",
		Mail:       "user@example.com",
		Password:   "hash",
		Registered: true,
		Role:       "student",
	}}
	storage := db.NewUserStorage(cfg.DatabaseURL)
	defer storage.Close()

	cache, err := rediscache.NewEstimationsCache(cfg.RedisURL)
	if err != nil {
		t.Fatalf("redis cache: %v", err)
	}
	defer func() { _ = cache.Close() }()

	manager, err := logic.NewUserManagerWithCache(provider, storage, cache, cfg.CacheTTL)
	if err != nil {
		t.Fatalf("user manager: %v", err)
	}

	app := server.New(cfg, *manager)

	body := []byte(`{"login":"ab"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var payload api.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("expected error code and message")
	}
}

func setupContainers(t *testing.T, ctx context.Context) (string, string, func()) {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("ystu_db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("user"),
	)
	if err != nil {
		t.Skipf("docker unavailable for postgres: %v", err)
	}

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Skipf("docker unavailable for redis: %v", err)
	}

	cleanup := func() {
		_ = redisContainer.Terminate(ctx)
		_ = pgContainer.Terminate(ctx)
	}

	dbURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cleanup()
		t.Fatalf("postgres connection string: %v", err)
	}

	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		cleanup()
		t.Fatalf("redis host: %v", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		cleanup()
		t.Fatalf("redis port: %v", err)
	}
	redisURL := "redis://" + redisHost + ":" + redisPort.Port() + "/0"

	applyMigrations(t, dbURL)

	return dbURL, redisURL, cleanup
}

func applyMigrations(t *testing.T, dbURL string) {
	root := projectRoot(t)
	path := filepath.ToSlash(filepath.Join(root, "migrations"))

	m, err := migrate.New("file://"+path, dbURL)
	if err != nil {
		t.Fatalf("create migrate: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up: %v", err)
	}
}

func projectRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	t.Fatalf("project root not found from %s", wd)
	return ""
}
