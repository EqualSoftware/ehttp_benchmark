// Command nethttp is the net/http (stdlib) app of the benchmark.
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/EqualSoftware/ehttp_benchmark/internal/common"
	"github.com/google/uuid"
)

func writeJSON(w http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", common.JSONType)
	_, _ = w.Write(b)
}

func serverHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", common.ServerName)
		next.ServeHTTP(w, r)
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.New().String()
		}

		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r)
	})
}

func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-XSS-Protection", common.XSSProtection)
		h.Set("X-Content-Type-Options", common.ContentTypeNosniff)
		h.Set("X-Frame-Options", common.XFrameOptions)
		next.ServeHTTP(w, r)
	})
}

func main() {
	jsonHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"message": common.Message})
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /plaintext", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", common.PlainType)
		_, _ = w.Write([]byte(common.PlainBody))
	})
	mux.Handle("GET /json", jsonHandler)
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"id": r.PathValue("id")})
	})
	mux.Handle("GET /mw/json", recoverPanic(requestID(secure(jsonHandler))))

	log.Fatal(common.Server(serverHeader(mux)).ListenAndServe())
}
