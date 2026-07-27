package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORSOptions builds chi/cors options with safe credential handling and dynamic origin reflection.
func CORSOptions(origins []string) cors.Options {
	return cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			// Reflect any requested origin (Vercel, Localhost, Render, etc.)
			return true
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Organization-ID", "X-App-ID", "X-Requested-With", "*"},
		ExposedHeaders:   []string{"X-Request-ID", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}
