package server

import (
	"YstuPortal/internal/config"
	"YstuPortal/internal/delivery/api"
	"YstuPortal/internal/logic"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func New(cfg config.Config, manager logic.UserManager) *fiber.App {
	metrics := api.NewMetrics()

	app := fiber.New(fiber.Config{ErrorHandler: api.ErrorHandler})
	app.Use(requestid.New(requestid.Config{
		Header: "X-Request-ID",
	}))
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip} - ${status} - ${latency} ${method} ${path} req_id=${locals:requestid}\n",
	}))
	app.Use(metrics.Middleware())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api.RegisterSwagger(app, cfg.SwaggerEnabled)
	app.Get("/metrics", metrics.Handler())

	r := app.Group("/api")
	_ = api.NewLoginApi(r, manager, cfg)
	_ = api.NewUserApi(r, manager)

	return app
}
