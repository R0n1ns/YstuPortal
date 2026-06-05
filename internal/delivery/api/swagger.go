package api

import (
	_ "embed"

	"github.com/gofiber/fiber/v3"
)

//go:embed docs/openapi.yaml
var openapiYAML string

//go:embed docs/swagger.html
var swaggerHTML string

func RegisterSwagger(app *fiber.App, enabled bool) {
	if !enabled {
		return
	}

	app.Get("/swagger", func(ctx fiber.Ctx) error {
		ctx.Set("Content-Type", "text/html; charset=utf-8")
		return ctx.SendString(swaggerHTML)
	})

	app.Get("/swagger/openapi.yaml", func(ctx fiber.Ctx) error {
		ctx.Set("Content-Type", "application/yaml")
		return ctx.SendString(openapiYAML)
	})
}
