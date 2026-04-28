package api

import (
	"YstuPortal/internal/domain"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type UserApi struct {
	DataManager *domain.UserProvider
}

var jwtSecret = "test"

func NewUserApi(router fiber.Router, manager *domain.UserProvider) *UserApi {
	mngr := &UserApi{DataManager: manager}
	router = router.Group("/")
	router.Post("/login", mngr.LoginPost)
	router.Post("/logout", mngr.LogoutPost)
	router.Use(mngr.AuthMiddleware)
	return mngr
}

func (u *UserApi) AuthMiddleware(ctx fiber.Ctx) error {
	cookie := ctx.Cookies("jwt")
	if cookie == "" {
		ctx.Status(fiber.StatusUnauthorized)
		return ctx.JSON(fiber.Map{"error": "Not authorized"})
	}
	token, err := jwt.ParseWithClaims(cookie, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"error": err.Error()})
	}
	if !token.Valid {
		ctx.Status(fiber.StatusUnauthorized)
		return ctx.JSON(fiber.Map{"error": "Not authorized"})
	}
	claims := token.Claims.(*jwt.StandardClaims)

	ctx.Locals("UserId", claims.Issuer)

	fmt.Println(claims)

	return ctx.Next()
}

func (u *UserApi) LoginPost(ctx fiber.Ctx) error {
	if !ctx.HasBody() {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}

	data := new(struct {
		Name string `json:"name"`
		Pass string `json:"password"`
	})

	if err := json.Unmarshal(ctx.Body(), &data); err != nil {
		fmt.Println("json", err)
		return err
	}

	user, err := u.DataManager.AuthUser(ctx)
	if err != nil {
		return ctx.SendStatus(fiber.StatusForbidden)
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Id:        uuid.New().String(),
		Issuer:    (*user).GetId().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	})

	token, err := claims.SignedString([]byte(jwtSecret))
	if err != nil {
		fmt.Println("claims", err)
		ctx.Status(fiber.StatusInternalServerError)
		return err
	}
	ctx.Cookie(&fiber.Cookie{
		Expires:  time.Now().Add(24 * time.Hour),
		Name:     "jwt",
		Value:    token,
		HTTPOnly: true,
	})
	fmt.Println(data.Name, data.Pass)
	ctx.SendStatus(fiber.StatusOK)
	return nil

}

func LogoutPost(ctx fiber.Ctx) error {
	ctx.ClearCookie()
	ctx.SendStatus(fiber.StatusOK)
	return nil

}
