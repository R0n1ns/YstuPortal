package api

import (
	"YstuPortal/internal"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func UserRoutes(router fiber.Router) {
	router = router.Group("/user")
	router.Get("/info", UserInfoGet)
}
func UserInfoGet(ctx fiber.Ctx) error {
	user, err := internal.UserStor.GetUser(uuid.MustParse(ctx.Value("UserId").(string)))
	if err != nil {
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.JSON(fiber.Map{"error": err.Error()})
	}
	ctx.Status(fiber.StatusOK)
	return ctx.JSON(user)
}
