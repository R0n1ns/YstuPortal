package api

import (
	"errors"

	"github.com/R0n1ns/YstuPortal/internal/domain"
	"github.com/R0n1ns/YstuPortal/internal/logic"

	"github.com/gofiber/fiber/v3"
)

type UserAPI struct {
	manager logic.UserManagerType
}

type userResponse struct {
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Patronymic string `json:"patronymic,omitempty"`
	UserName   string `json:"username"`
	Mail       string `json:"mail,omitempty"`
	Registered bool   `json:"registered"`
	Role       string `json:"role"`
	Group      string `json:"group,omitempty"`
}

func NewUserAPI(router fiber.Router, manager logic.UserManagerType) *UserAPI {
	api := &UserAPI{manager: manager}
	user := router.Group("/user")
	user.Get("/info", api.GetUserInfo)
	user.Get("/grades", api.GetUserGrades)
	user.Get("/admin/ping", RequireRole("admin"), api.AdminPing)
	return api
}

func (api *UserAPI) GetUserInfo(ctx fiber.Ctx) error {
	userName, _ := ctx.Locals(userNameLocal).(string)
	user, err := api.manager.GetInfo(ctx, userName)
	if err != nil {
		return writeUserError(ctx, err)
	}
	return ctx.JSON(userResponse{
		FirstName: user.FirstName, LastName: user.LastName, Patronymic: user.Patronymic,
		UserName: user.UserName, Mail: user.Mail, Registered: user.Registered,
		Role: user.Role, Group: user.Group,
	})
}

func (api *UserAPI) GetUserGrades(ctx fiber.Ctx) error {
	userName, _ := ctx.Locals(userNameLocal).(string)
	grades, err := api.manager.GetGrades(ctx, userName)
	if err != nil {
		return writeUserError(ctx, err)
	}
	return ctx.JSON(grades)
}

func (api *UserAPI) AdminPing(ctx fiber.Ctx) error {
	return ctx.JSON(fiber.Map{"status": "ok"})
}

func writeUserError(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return WriteError(ctx, fiber.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, domain.ErrSessionNotFound):
		return WriteError(ctx, fiber.StatusConflict, "session_expired", "sign in again to refresh data")
	default:
		return WriteError(ctx, fiber.StatusInternalServerError, "internal", "failed to load user data")
	}
}
