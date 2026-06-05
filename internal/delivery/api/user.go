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
	router.Get("/estimations", u.GetUserEstimations)
	router.Get("/admin/ping", RequireRole("admin"), u.AdminPing)
	return &u
}

func (u UserApi) GetUserInfo(ctx fiber.Ctx) error {
	username, _ := ctx.Locals("UserName").(string)
	if username == "" {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "not authorized")
	}
	user, err := u.UserManager.GetInfo(ctx, username)
	if err != nil {
		return WriteError(ctx, fiber.StatusInternalServerError, "internal", "failed to load user info")
	}
	ctx.Status(fiber.StatusOK)
	return ctx.JSON(user)
}

func (u UserApi) GetUserEstimations(ctx fiber.Ctx) error {
	username, _ := ctx.Locals("UserName").(string)
	if username == "" {
		return WriteError(ctx, fiber.StatusUnauthorized, "unauthorized", "not authorized")
	}
	user, err := u.UserManager.GetEstimations(ctx, username)
	if err != nil {
		return WriteError(ctx, fiber.StatusInternalServerError, "internal", "failed to load estimations")
	}
	ctx.Status(fiber.StatusOK)
	return ctx.JSON(user)
}

func (u UserApi) AdminPing(ctx fiber.Ctx) error {
	return ctx.JSON(fiber.Map{"status": "ok"})
}
