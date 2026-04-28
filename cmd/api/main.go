package main

import (
	"YstuPortal/internal/delivery/api"
	"YstuPortal/internal/repository/userData"

	"github.com/gofiber/fiber/v3"
)

var UserStorage = userData.NewUserStorage()

func main() {
	app := fiber.New()
	api.AuthRoutes(app)
	api.UserRoutes(app)
	app.Listen(":8080")
}
