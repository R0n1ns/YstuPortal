package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/R0n1ns/YstuPortal/internal/config"
	"github.com/R0n1ns/YstuPortal/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type authManagerStub struct {
	logoutUser string
}

func (m *authManagerStub) Login(_ context.Context, username, _ string) (*domain.User, error) {
	return &domain.User{UserName: username, Role: "student"}, nil
}

func (m *authManagerStub) GetInfo(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (m *authManagerStub) GetGrades(context.Context, string) ([]domain.Subject, error) {
	return nil, nil
}

func (m *authManagerStub) Logout(userName string) {
	m.logoutUser = userName
}

func TestLoginAuthenticationAndLogout(t *testing.T) {
	manager := &authManagerStub{}
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	router := app.Group("/api")
	loginAPI := NewLoginAPI(router, manager, authTestConfig())
	protected := router.Group("", loginAPI.AuthMiddleware)
	loginAPI.RegisterProtected(protected)
	protected.Get("/private", func(ctx fiber.Ctx) error {
		return ctx.JSON(fiber.Map{"username": ctx.Locals(userNameLocal)})
	})
	protected.Get("/admin", RequireRole("admin"), func(ctx fiber.Ctx) error {
		return ctx.SendStatus(http.StatusOK)
	})

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"login":"demo","password":"demo123"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse, err := app.Test(loginRequest)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginResponse.StatusCode != http.StatusOK || len(loginResponse.Cookies()) == 0 {
		t.Fatalf("unexpected login response: %d", loginResponse.StatusCode)
	}
	cookie := loginResponse.Cookies()[0]
	if !cookie.HttpOnly || cookie.Value == "" {
		t.Fatalf("JWT cookie must be non-empty and HttpOnly: %+v", cookie)
	}

	privateRequest := httptest.NewRequest(http.MethodGet, "/api/private", nil)
	privateRequest.AddCookie(cookie)
	privateResponse, err := app.Test(privateRequest)
	if err != nil || privateResponse.StatusCode != http.StatusOK {
		t.Fatalf("private endpoint: status=%d err=%v", privateResponse.StatusCode, err)
	}

	adminRequest := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
	adminRequest.AddCookie(cookie)
	adminResponse, err := app.Test(adminRequest)
	if err != nil || adminResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("admin endpoint: status=%d err=%v", adminResponse.StatusCode, err)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutRequest.AddCookie(cookie)
	logoutResponse, err := app.Test(logoutRequest)
	if err != nil || logoutResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: status=%d err=%v", logoutResponse.StatusCode, err)
	}
	if manager.logoutUser != "demo" {
		t.Fatalf("expected upstream session cleanup for demo, got %q", manager.logoutUser)
	}
}

func TestLoginRejectsUnknownFields(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	NewLoginAPI(app.Group("/api"), &authManagerStub{}, authTestConfig())
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"login":"demo","password":"demo123","admin":true}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
}

func authTestConfig() config.Config {
	return config.Config{
		JWTSecret:       "test-secret-with-at-least-32-characters",
		JWTTTL:          time.Hour,
		JWTIssuer:       "test-issuer",
		JWTAudience:     "test-audience",
		CookieSameSite:  "lax",
		RateLimitMax:    100,
		RateLimitWindow: time.Minute,
	}
}
