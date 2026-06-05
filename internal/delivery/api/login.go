package api

import (
	"YstuPortal/internal/config"
	"YstuPortal/internal/logic"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type LoginApi struct {
	DataManager    logic.UserManager
	jwtSecret      []byte
	jwtTTL         time.Duration
	cookieSecure   bool
	cookieSameSite string
	cookieDomain   string
	limitMax       int
	limitWindow    time.Duration
}

type AccessClaims struct {
	Role string `json:"role"`
	jwt.StandardClaims
}

func NewLoginApi(r fiber.Router, manager logic.UserManager, cfg config.Config) *LoginApi {
	mngr := &LoginApi{
		DataManager:    manager,
		jwtSecret:      []byte(cfg.JWTSecret),
		jwtTTL:         cfg.JWTTTL,
		cookieSecure:   cfg.CookieSecure,
		cookieSameSite: cfg.CookieSameSite,
		cookieDomain:   cfg.CookieDomain,
		limitMax:       cfg.RateLimitMax,
		limitWindow:    cfg.RateLimitWindow,
	}
	router := r.Group("/auth")
	router.Post("/login", limiter.New(limiter.Config{
		Max:        mngr.limitMax,
		Expiration: mngr.limitWindow,
		LimitReached: func(ctx fiber.Ctx) error {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit"})
		},
	}), mngr.LoginPost)
	router.Post("/logout", mngr.LogoutPost)
	r.Use(mngr.AuthMiddleware)
	return mngr
}

func (u *LoginApi) AuthMiddleware(ctx fiber.Ctx) error {
	cookie := ctx.Cookies("jwt")
	if cookie == "" {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "not authorized")
	}
	token, err := jwt.ParseWithClaims(cookie, &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		return u.jwtSecret, nil
	})
	if err != nil {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "invalid token")
	}
	if !token.Valid {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "not authorized")
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "invalid token")
	}

	ctx.Locals("UserName", claims.Issuer)
	ctx.Locals("Role", claims.Role)
	return ctx.Next()
}

func RequireRole(role string) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		currentRole, _ := ctx.Locals("Role").(string)
		if currentRole != role {
			return WriteError(ctx, fiber.StatusForbidden, "forbidden", "forbidden")
		}
		return ctx.Next()
	}
}

func (u *LoginApi) LoginPost(ctx fiber.Ctx) error {
	if !ctx.HasBody() {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "request body is required")
	}

	data := new(struct {
		Name string `json:"login"`
		Pass string `json:"password"`
	})

	if err := json.Unmarshal(ctx.Body(), &data); err != nil {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "invalid json")
	}
	name := strings.TrimSpace(data.Name)
	pass := strings.TrimSpace(data.Pass)
	if name == "" || pass == "" {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "login and password are required")
	}
	if len(name) < 3 || len(name) > 64 {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "login must be 3-64 characters")
	}
	if len(pass) < 6 || len(pass) > 128 {
		return WriteError(ctx, fiber.StatusBadRequest, "bad_request", "password must be 6-128 characters")
	}

	user, err := u.DataManager.Login(ctx, name, pass)
	if err != nil {
		return WriteError(ctx, fiber.StatusForbidden, "forbidden", "invalid credentials")
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, AccessClaims{
		Role: user.Role,
		StandardClaims: jwt.StandardClaims{
			Id:        uuid.New().String(),
			Issuer:    user.UserName,
			ExpiresAt: time.Now().Add(u.jwtTTL).Unix(),
		},
	})

	token, err := claims.SignedString(u.jwtSecret)
	if err != nil {
		return WriteError(ctx, fiber.StatusInternalServerError, "internal", "failed to sign token")
	}

	ctx.Cookie(&fiber.Cookie{
		Expires:  time.Now().Add(u.jwtTTL),
		Name:     "jwt",
		Value:    token,
		HTTPOnly: true,
		Secure:   u.cookieSecure,
		SameSite: parseSameSite(u.cookieSameSite),
		Domain:   u.cookieDomain,
	})
	_ = ctx.SendStatus(fiber.StatusOK)
	ctx.Locals("UserName", user.UserName)
	ctx.Locals("Role", user.Role)

	return nil

}

func (u *LoginApi) LogoutPost(ctx fiber.Ctx) error {
	ctx.ClearCookie()
	_ = ctx.SendStatus(fiber.StatusOK)
	return nil

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
