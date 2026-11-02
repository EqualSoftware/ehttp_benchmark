// Command fiber is the Fiber app of the benchmark (prefork off).
package main

import (
	"log"

	"github.com/EqualSoftware/ehttp_benchmark/internal/common"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func sendJSON(c fiber.Ctx, v any) error {
	return c.JSON(v, common.JSONType)
}

func secure(c fiber.Ctx) error {
	c.Set("X-XSS-Protection", common.XSSProtection)
	c.Set("X-Content-Type-Options", common.ContentTypeNosniff)
	c.Set("X-Frame-Options", common.XFrameOptions)
	return c.Next()
}

func main() {
	// fasthttp has no ReadHeaderTimeout. ReadTimeout stays zero, as in the other apps.
	app := fiber.New(fiber.Config{ServerHeader: common.ServerName})

	jsonHandler := func(c fiber.Ctx) error {
		return sendJSON(c, map[string]string{"message": common.Message})
	}

	app.Get("/plaintext", func(c fiber.Ctx) error {
		c.Set("Content-Type", common.PlainType)
		return c.SendString(common.PlainBody)
	})
	app.Get("/json", jsonHandler)
	app.Get("/users/:id", func(c fiber.Ctx) error {
		return sendJSON(c, map[string]string{"id": c.Params("id")})
	})
	app.Group("/mw", recover.New(), requestid.New(), secure).Get("/json", jsonHandler)

	log.Fatal(app.Listen(common.Addr(), fiber.ListenConfig{DisableStartupMessage: true}))
}
