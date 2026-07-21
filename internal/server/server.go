package server

import (
	"github.com/R0n1ns/YstuPortal/internal/config"
	"github.com/R0n1ns/YstuPortal/internal/delivery/api"
	"github.com/R0n1ns/YstuPortal/internal/logic"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func New(cfg config.Config, manager logic.UserManagerType) *fiber.App {
	metrics := api.NewMetrics()

	app := fiber.New(fiber.Config{
		ErrorHandler: api.ErrorHandler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		BodyLimit:    1 * 1024 * 1024,
	})
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
	app.Get("/health/live", func(ctx fiber.Ctx) error {
		return ctx.JSON(fiber.Map{"status": "ok"})
	})

	r := app.Group("/api")
	loginAPI := api.NewLoginAPI(r, manager, cfg)
	protected := r.Group("", loginAPI.AuthMiddleware)
	loginAPI.RegisterProtected(protected)
	_ = api.NewUserAPI(protected, manager)

	return app
}
