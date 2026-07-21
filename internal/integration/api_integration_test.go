package integration

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/R0n1ns/YstuPortal/internal/config"
	"github.com/R0n1ns/YstuPortal/internal/domain"
	"github.com/R0n1ns/YstuPortal/internal/logic"
	rediscache "github.com/R0n1ns/YstuPortal/internal/repository/cache/redis"
	"github.com/R0n1ns/YstuPortal/internal/repository/userStorage/db"
	"github.com/R0n1ns/YstuPortal/internal/server"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testProvider struct {
	user   domain.User
	grades []domain.Subject
}

func (p *testProvider) GetUser(_ context.Context, username, _ string) (*domain.User, error) {
	user := p.user
	user.UserName = username
	return &user, nil
}

func (p *testProvider) GetGrades(_ context.Context, _ string) ([]domain.Subject, error) {
	return append([]domain.Subject(nil), p.grades...), nil
}

func TestCompleteUserJourney(t *testing.T) {
	ctx := context.Background()
	databaseURL, redisURL, cleanup := setupContainers(t, ctx)
	defer cleanup()

	cfg := testConfig(databaseURL, redisURL)
	provider := &testProvider{
		user: domain.User{
			FirstName: "Test", LastName: "User", Registered: true, Role: "admin", Group: "TEST-01",
		},
		grades: []domain.Subject{{Course: 1, Semester: 2, Title: "Go", TypeOfControl: "Exam", Zed: 4, Mark: "95", Evaluation: "Excellent"}},
	}
	storage, err := db.NewUserStorage(ctx, databaseURL)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	defer storage.Close()
	cache, err := rediscache.NewGradesCache(redisURL)
	if err != nil {
		t.Fatalf("redis cache: %v", err)
	}
	defer func() { _ = cache.Close() }()
	manager, err := logic.NewUserManagerWithCache(provider, storage, cache, time.Minute)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	app := server.New(cfg, manager)

	var authCookie *http.Cookie
	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"login":"student","password":"pass1234"}`))
		request.Header.Set("Content-Type", "application/json")
		response, requestErr := app.Test(request)
		if requestErr != nil {
			t.Fatalf("login attempt %d: %v", attempt+1, requestErr)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("login attempt %d: expected 200, got %d", attempt+1, response.StatusCode)
		}
		if len(response.Cookies()) == 0 {
			t.Fatal("JWT cookie is missing")
		}
		authCookie = response.Cookies()[0]
	}

	infoResponse := authenticatedRequest(t, app, http.MethodGet, "/api/user/info", authCookie)
	infoBody, err := io.ReadAll(infoResponse.Body)
	if err != nil {
		t.Fatalf("read info response: %v", err)
	}
	if infoResponse.StatusCode != http.StatusOK {
		t.Fatalf("info: expected 200, got %d: %s", infoResponse.StatusCode, infoBody)
	}
	if strings.Contains(strings.ToLower(string(infoBody)), "password") {
		t.Fatalf("user response leaks password data: %s", infoBody)
	}

	gradesResponse := authenticatedRequest(t, app, http.MethodGet, "/api/user/grades", authCookie)
	gradesBody, err := io.ReadAll(gradesResponse.Body)
	if err != nil {
		t.Fatalf("read grades response: %v", err)
	}
	if gradesResponse.StatusCode != http.StatusOK || !strings.Contains(string(gradesBody), `"title":"Go"`) {
		t.Fatalf("grades: unexpected response %d: %s", gradesResponse.StatusCode, gradesBody)
	}

	adminResponse := authenticatedRequest(t, app, http.MethodGet, "/api/user/admin/ping", authCookie)
	if adminResponse.StatusCode != http.StatusOK {
		t.Fatalf("admin endpoint: expected 200, got %d", adminResponse.StatusCode)
	}

	storedUser, err := storage.GetUser(ctx, "student")
	if err != nil {
		t.Fatalf("read stored user: %v", err)
	}
	if len(storedUser.Grades) != 1 {
		t.Fatalf("expected one persisted grade, got %d", len(storedUser.Grades))
	}
}

func authenticatedRequest(t *testing.T, app *fiber.App, method, target string, cookie *http.Cookie) *http.Response {
	t.Helper()
	request := httptest.NewRequest(method, target, nil)
	request.AddCookie(cookie)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	return response
}

func testConfig(databaseURL, redisURL string) config.Config {
	return config.Config{
		Environment: "test", DatabaseURL: databaseURL, Port: "0",
		JWTSecret: "test-secret-with-enough-entropy", JWTTTL: time.Hour,
		JWTIssuer: "ystuportal-test", JWTAudience: "ystuportal-test-client",
		AllowedOrigins: []string{"http://localhost"}, RedisURL: redisURL, CacheTTL: time.Minute,
		RateLimitMax: 100, RateLimitWindow: time.Minute, SwaggerEnabled: false,
		CookieSameSite: "lax", ReadTimeout: time.Second, WriteTimeout: time.Second, IdleTimeout: time.Second,
	}
}

func setupContainers(t *testing.T, ctx context.Context) (string, string, func()) {
	t.Helper()
	postgresContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("ystu_db"), postgres.WithUsername("user"), postgres.WithPassword("user"),
	)
	if err != nil {
		t.Skipf("Docker unavailable for PostgreSQL: %v", err)
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "redis:7-alpine", ExposedPorts: []string{"6379/tcp"}, WaitingFor: wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		_ = postgresContainer.Terminate(ctx)
		t.Skipf("Docker unavailable for Redis: %v", err)
	}
	cleanup := func() {
		_ = redisContainer.Terminate(ctx)
		_ = postgresContainer.Terminate(ctx)
	}

	databaseURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cleanup()
		t.Fatalf("PostgreSQL connection string: %v", err)
	}
	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		cleanup()
		t.Fatalf("Redis host: %v", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		cleanup()
		t.Fatalf("Redis port: %v", err)
	}
	applyMigrations(t, databaseURL)
	return databaseURL, "redis://" + redisHost + ":" + redisPort.Port() + "/0", cleanup
}

func applyMigrations(t *testing.T, databaseURL string) {
	t.Helper()
	path := filepath.ToSlash(filepath.Join(projectRoot(t), "migrations"))
	migration, err := migrate.New("file://"+path, databaseURL)
	if err != nil {
		t.Fatalf("create migration: %v", err)
	}
	defer func() { _, _ = migration.Close() }()
	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("apply migrations: %v", err)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("project root not found")
		}
		current = parent
	}
}
