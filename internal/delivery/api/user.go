package api

import (
	"YstuPortal/internal/logic"

	"github.com/gofiber/fiber/v3"
)

type UserApi struct {
	UserManager logic.UserManager
}

func NewUserApi(router fiber.Router, d logic.UserManager) *UserApi {
	u := UserApi{
		UserManager: d,
	}
	router = router.Group("/user")
	router.Get("/info", u.GetUserInfo)
	return &u
}

func (u UserApi) GetUserInfo(ctx fiber.Ctx) error {
	username := ctx.Value("UserName")
	if username == nil {
		return ctx.JSON(map[string]string{"error": "Ошибка логина"})
	}
	user, err := u.UserManager.GetInfo(ctx, username.(string))
	if err != nil {
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"error": err.Error()})
	}
	ctx.Status(fiber.StatusOK)
	return ctx.JSON(user)
}

func (u UserApi) GetUserEstimations(ctx fiber.Ctx) error {
	user, err := u.UserManager.GetEstimations(ctx, ctx.Value("UserName").(string))
	if err != nil {
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"error": err.Error()})
	}
	ctx.Status(fiber.StatusOK)
	return ctx.JSON(user)
}
