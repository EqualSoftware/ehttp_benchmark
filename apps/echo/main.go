// Command echo is the Echo app of the benchmark.
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/EqualSoftware/ehttp_benchmark/internal/common"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func sendJSON(c *echo.Context, v any) error {
	c.Response().Header().Set("Content-Type", common.JSONType)
	return c.JSON(http.StatusOK, v)
}

// serializer writes json.Marshal bytes. The default serializer of Echo adds a
// trailing newline, and the other apps do not.
type serializer struct{ echo.DefaultJSONSerializer }

func (serializer) Serialize(c *echo.Context, target any, _ string) error {
	b, err := json.Marshal(target)
	if err != nil {
		return err
	}

	_, err = c.Response().Write(b)
	return err
}

func serverHeader(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		c.Response().Header().Set("Server", common.ServerName)
		return next(c)
	}
}

func main() {
	e := echo.New()
	e.JSONSerializer = serializer{}
	e.Use(serverHeader) // no Logger middleware: no access log

	jsonHandler := func(c *echo.Context) error {
		return sendJSON(c, map[string]string{"message": common.Message})
	}

	e.GET("/plaintext", func(c *echo.Context) error {
		c.Response().Header().Set("Content-Type", common.PlainType)
		return c.String(http.StatusOK, common.PlainBody)
	})
	e.GET("/json", jsonHandler)
	e.GET("/users/:id", func(c *echo.Context) error {
		return sendJSON(c, map[string]string{"id": c.Param("id")})
	})

	e.Group("/mw",
		middleware.Recover(),
		middleware.RequestID(),
		middleware.SecureWithConfig(middleware.SecureConfig{
			XSSProtection:      common.XSSProtection,
			ContentTypeNosniff: common.ContentTypeNosniff,
			XFrameOptions:      common.XFrameOptions,
		}),
	).GET("/json", jsonHandler)

	log.Fatal(common.Server(e).ListenAndServe())
}
