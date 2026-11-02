// Command ehttp is the ehttp app of the benchmark.
package main

import (
	"context"
	"log"

	"github.com/EqualSoftware/ehttp"
	"github.com/EqualSoftware/ehttp/middleware"
	"github.com/EqualSoftware/ehttp_benchmark/internal/common"
	"go.uber.org/zap"
)

func serverHeader() ehttp.MiddlewareHandlerFunc {
	return func(next ehttp.Handler) ehttp.HandlerFunc {
		return func(res *ehttp.ResponseWriter, req *ehttp.Request) {
			res.Own("Server", common.ServerName)
			next.ServeHTTP(res, req)
		}
	}
}

func sendJSON(res *ehttp.ResponseWriter, v any) {
	res.Set("Content-Type", common.JSONType)
	res.JSON(v)
}

func main() {
	port := common.Addr()[1:]

	server := ehttp.NewServer(
		ehttp.WithLogger(zap.NewNop()),
		ehttp.WithPort(port),
		ehttp.WithReadHeaderTimeout(common.ReadHeaderTimeout),
	)
	server.Use(serverHeader())

	jsonHandler := func(res *ehttp.ResponseWriter, _ *ehttp.Request) {
		sendJSON(res, map[string]string{"message": common.Message})
	}

	server.GET("/plaintext", func(res *ehttp.ResponseWriter, _ *ehttp.Request) {
		res.Set("Content-Type", common.PlainType)
		res.String(common.PlainBody)
	})
	server.GET("/json", jsonHandler)
	server.GET("/users/:id", func(res *ehttp.ResponseWriter, req *ehttp.Request) {
		sendJSON(res, map[string]string{"id": req.Param("id")})
	})
	server.Route("/mw",
		middleware.Recover(),
		middleware.RequestId(),
		middleware.Secure(middleware.DefaultSecureConfig),
	).GET("/json", jsonHandler)

	log.Fatal(server.Listen(context.Background()))
}
