package main

import (
	"log"
	"net/http"
	"os"

	"backend/config"
	"backend/models"
	"backend/routes"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware sets up basic cross-origin headers for frontend communication
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func main() {
	log.Println("Starting Smart Emotion & Focus Journal Backend...")

	// Initialize Database Connection
	db := config.InitDB()
	if db != nil {
		log.Println("Running database migrations...")
		err := db.AutoMigrate(
			&models.User{},
			&models.UserConfig{},
			&models.Journal{},
			&models.LlmMetric{},
		)
		if err != nil {
			log.Printf("Failed to run migrations: %v", err)
		} else {
			log.Println("Migrations completed successfully.")
		}
	}

	// Initialize Gin engine
	r := gin.Default()

	// Use Middlewares
	r.Use(CORSMiddleware())
	r.Use(gin.Recovery())

	// Register Routes
	routes.SetupRoutes(r)

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Server is running on port %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
