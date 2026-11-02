// Package common holds the values that all benchmark apps share, so they
// answer the same bytes and use the same server settings.
package common

import (
	"net/http"
	"os"
	"time"
)

const (
	// PlainBody is the body of GET /plaintext.
	PlainBody = "Hello, World!"
	// PlainType is the Content-Type of GET /plaintext.
	PlainType = "text/plain; charset=utf-8"
	// JSONType is the Content-Type of every JSON response.
	JSONType = "application/json"
	// Message is the value of the "message" field of GET /json.
	Message = "Hello, World!"
	// ServerName is the value of the Server header. All apps send the same value.
	ServerName = "bench"

	// Security headers of the middleware route (the DefaultSecureConfig of ehttp).
	XSSProtection      = "0"
	ContentTypeNosniff = "nosniff"
	XFrameOptions      = "SAMEORIGIN"

	// ReadHeaderTimeout is the same in all apps that can set it (the ehttp default).
	// Read, write and idle timeouts stay at zero, as in ehttp.
	ReadHeaderTimeout = 10 * time.Second
)

// Addr returns the listen address. PORT overrides the default 3000.
func Addr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}

	return ":3000"
}

// Server returns an http.Server with the shared settings.
func Server(h http.Handler) *http.Server {
	return &http.Server{Addr: Addr(), Handler: h, ReadHeaderTimeout: ReadHeaderTimeout}
}
