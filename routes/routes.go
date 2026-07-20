package routes

import (
	"backend/controllers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes registers all 20 required API endpoints with the Gin engine
func SetupRoutes(r *gin.Engine) {
	// Auth (8 Endpoints)
	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)
	r.POST("/logout", controllers.Logout)
	r.POST("/refresh", controllers.RefreshToken)
	r.GET("/profile", controllers.GetProfile)
	r.PUT("/profile", controllers.UpdateProfile)
	r.PUT("/password", controllers.UpdatePassword)
	r.DELETE("/delete", controllers.DeleteAccount)

	// Config (2 Endpoints)
	r.GET("/config", controllers.GetConfig)
	r.PUT("/config", controllers.UpdateConfig)

	// Web MLC-LLM (7 Endpoints)
	api := r.Group("/api")
	{
		// Journal routes
		api.POST("/journal", controllers.CreateJournal)
		api.GET("/journal", controllers.GetJournals)

		// Monitoring and metrics routes
		api.POST("/monitor/metrics", controllers.CreateMetric)
		api.GET("/monitor/metrics", controllers.GetMetrics)
		api.GET("/monitor/scores", controllers.GetScores)
		api.POST("/monitor/error", controllers.CreateErrorLog)
		api.DELETE("/monitor/clear", controllers.ClearMetrics)
	}

	// Common (3 Endpoints)
	r.GET("/health", controllers.HealthCheck)
	r.GET("/version", controllers.GetVersion)
	r.POST("/feedback", controllers.SubmitFeedback)
}
