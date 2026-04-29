package main

import (
	"YstuPortal/internal/delivery/api"
	"YstuPortal/internal/logic"
	"YstuPortal/internal/repository/userProvider"

	"github.com/gofiber/fiber/v3"
)

var UserStorage = userProvider.NewUserStorage()

func main() {
	parser := userProvider.NewUserStorage()
	dataManager, _ := logic.NewUserManager(parser, parser)

	app := fiber.New()

	_ = api.NewLoginApi(app, *dataManager)
	_ = api.NewUserApi(app, *dataManager)

	app.Listen(":8080")
}
