package middleware

import (
	"github.com/go-chi/cors"
)

// CORSOptions builds chi/cors options with safe credential handling.
func CORSOptions(origins []string) cors.Options {
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	hasWildcard := false
	for _, origin := range origins {
		if origin == "*" {
			hasWildcard = true
			break
		}
	}

	opts := cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Organization-ID", "X-App-ID", "X-Requested-With"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: !hasWildcard,
		MaxAge:           300,
	}

	return opts
}
