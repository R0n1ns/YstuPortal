package api

import (
	"YstuPortal/internal/logic"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type UserApi struct {
	UserManager logic.UserManager
}

func NewUserApi(router fiber.Router, d logic.UserManager) *UserApi {
	u := UserApi{
		UserManager: d,
	}
	router = router.Group("/user")
	router.Get("/info", u.UserInfoGet)
	return &u
}

func (u UserApi) UserInfoGet(ctx fiber.Ctx) error {
	user, err := u.UserManager.UserStorage.GetUser(ctx, uuid.MustParse(ctx.Value("UserId").(string)))
	if err != nil {
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"error": err.Error()})
	}
	ctx.Status(fiber.StatusOK)
	return ctx.JSON(user)
}
