package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/R0n1ns/YstuPortal/internal/config"
	"github.com/R0n1ns/YstuPortal/internal/domain"
	"github.com/R0n1ns/YstuPortal/internal/logic"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	userNameLocal = "user_name"
	roleLocal     = "role"
	jwtCookieName = "jwt"
)

type LoginAPI struct {
	manager        logic.UserManagerType
	jwtSecret      []byte
	jwtTTL         time.Duration
	jwtIssuer      string
	jwtAudience    string
	cookieSecure   bool
	cookieSameSite string
	cookieDomain   string
}

type AccessClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func NewLoginAPI(router fiber.Router, manager logic.UserManagerType, cfg config.Config) *LoginAPI {
	api := &LoginAPI{
		manager:        manager,
		jwtSecret:      []byte(cfg.JWTSecret),
		jwtTTL:         cfg.JWTTTL,
		jwtIssuer:      cfg.JWTIssuer,
		jwtAudience:    cfg.JWTAudience,
		cookieSecure:   cfg.CookieSecure,
		cookieSameSite: cfg.CookieSameSite,
		cookieDomain:   cfg.CookieDomain,
	}

	auth := router.Group("/auth")
	auth.Post("/login", limiter.New(limiter.Config{
		Max:        cfg.RateLimitMax,
		Expiration: cfg.RateLimitWindow,
		LimitReached: func(ctx fiber.Ctx) error {
			return WriteError(ctx, fiber.StatusTooManyRequests, "rate_limit", "too many login attempts")
		},
	}), api.LoginPost)
	return api
}

func (api *LoginAPI) RegisterProtected(router fiber.Router) {
	router.Post("/auth/logout", api.LogoutPost)
}

func (api *LoginAPI) AuthMiddleware(ctx fiber.Ctx) error {
	cookie := ctx.Cookies(jwtCookieName)
	if cookie == "" {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "authentication is required")
	}

	token, err := jwt.ParseWithClaims(
		cookie,
		&AccessClaims{},
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}
			return api.jwtSecret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(api.jwtIssuer),
		jwt.WithAudience(api.jwtAudience),
	)
	if err != nil || !token.Valid {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "invalid or expired token")
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok || claims.Subject == "" {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "invalid token claims")
	}

	ctx.Locals(userNameLocal, claims.Subject)
	ctx.Locals(roleLocal, claims.Role)
	return ctx.Next()
}

func RequireRole(role string) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		currentRole, _ := ctx.Locals(roleLocal).(string)
		if currentRole != role {
			return WriteError(ctx, fiber.StatusForbidden, "forbidden", "insufficient permissions")
		}
		return ctx.Next()
	}
}

func (api *LoginAPI) LoginPost(ctx fiber.Ctx) error {
	if !ctx.HasBody() {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "request body is required")
	}

	var request loginRequest
	decoder := json.NewDecoder(strings.NewReader(string(ctx.Body())))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	request.Login = strings.TrimSpace(request.Login)
	request.Password = strings.TrimSpace(request.Password)
	if len(request.Login) < 3 || len(request.Login) > 64 {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "login must contain 3-64 characters")
	}
	if len(request.Password) < 6 || len(request.Password) > 128 {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "password must contain 6-128 characters")
	}

	user, err := api.manager.Login(ctx, request.Login, request.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return WriteError(ctx, fiber.StatusForbidden, "invalid_credentials", "invalid login or password")
		}
		return WriteError(ctx, fiber.StatusInternalServerError, "internal", "authentication service is unavailable")
	}

	now := time.Now()
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, AccessClaims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   user.UserName,
			Issuer:    api.jwtIssuer,
			Audience:  jwt.ClaimStrings{api.jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(api.jwtTTL)),
		},
	})
	token, err := claims.SignedString(api.jwtSecret)
	if err != nil {
		return WriteError(ctx, fiber.StatusInternalServerError, "internal", "failed to create access token")
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     jwtCookieName,
		Value:    token,
		Path:     "/",
		Domain:   api.cookieDomain,
		Expires:  now.Add(api.jwtTTL),
		HTTPOnly: true,
		Secure:   api.cookieSecure,
		SameSite: parseSameSite(api.cookieSameSite),
	})
	return ctx.SendStatus(fiber.StatusOK)
}

func (api *LoginAPI) LogoutPost(ctx fiber.Ctx) error {
	userName, _ := ctx.Locals(userNameLocal).(string)
	api.manager.Logout(userName)
	ctx.Cookie(&fiber.Cookie{
		Name:     jwtCookieName,
		Value:    "",
		Path:     "/",
		Domain:   api.cookieDomain,
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   api.cookieSecure,
		SameSite: parseSameSite(api.cookieSameSite),
	})
	return ctx.SendStatus(fiber.StatusNoContent)
}

func parseSameSite(value string) string {
	switch value {
	case "none":
		return fiber.CookieSameSiteNoneMode
	case "strict":
		return fiber.CookieSameSiteStrictMode
	default:
		return fiber.CookieSameSiteLaxMode
	}
}
