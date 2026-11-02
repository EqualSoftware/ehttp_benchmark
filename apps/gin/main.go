// Command gin is the Gin app of the benchmark.
package main

import (
	"log"
	"net/http"

	"github.com/EqualSoftware/ehttp_benchmark/internal/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func sendJSON(c *gin.Context, v any) {
	c.Header("Content-Type", common.JSONType)
	c.JSON(http.StatusOK, v)
}

func serverHeader(c *gin.Context) {
	c.Header("Server", common.ServerName)
	c.Next()
}

func requestID(c *gin.Context) {
	id := c.GetHeader("X-Request-Id")
	if id == "" {
		id = uuid.New().String()
	}

	c.Header("X-Request-Id", id)
	c.Next()
}

func secure(c *gin.Context) {
	c.Header("X-XSS-Protection", common.XSSProtection)
	c.Header("X-Content-Type-Options", common.ContentTypeNosniff)
	c.Header("X-Frame-Options", common.XFrameOptions)
	c.Next()
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New() // no Logger middleware: no access log
	r.Use(serverHeader)

	jsonHandler := func(c *gin.Context) {
		sendJSON(c, map[string]string{"message": common.Message})
	}

	r.GET("/plaintext", func(c *gin.Context) {
		c.Header("Content-Type", common.PlainType)
		c.String(http.StatusOK, common.PlainBody)
	})
	r.GET("/json", jsonHandler)
	r.GET("/users/:id", func(c *gin.Context) {
		sendJSON(c, map[string]string{"id": c.Param("id")})
	})
	r.Group("/mw", gin.Recovery(), requestID, secure).GET("/json", jsonHandler)

	log.Fatal(common.Server(r).ListenAndServe())
}
