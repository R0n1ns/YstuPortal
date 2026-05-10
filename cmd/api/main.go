package main

import (
	"YstuPortal/internal/delivery/api"
	"YstuPortal/internal/logic"
	"YstuPortal/internal/repository/userProvider"
	"YstuPortal/internal/repository/userStorage/db"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func main() {
	storage := db.NewUserStorage("postgres://user:user@localhost:5432/ystu_db")
	defer storage.Close()
	parser := userProvider.NewUserParser()
	defer parser.Close()
	dataManager, _ := logic.NewUserManager(parser, storage)

	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "http://127.0.0.1:8080"},
		AllowMethods:     []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	r := fiber.Router.Group(app, "/api/")
	_ = api.NewLoginApi(r, *dataManager)
	_ = api.NewUserApi(r, *dataManager)

	_ = app.Listen(":8080")
}
