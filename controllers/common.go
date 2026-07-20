package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns server status
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "success",
	})
}

// GetVersion returns current application version
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": "1.2.5",
		"release": "stable",
		"build":   "2026-07-20",
	})
}

type FeedbackInput struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment" binding:"required"`
}

// SubmitFeedback handles user feedback submission
func SubmitFeedback(c *gin.Context) {
	var input FeedbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback format invalid. Rating must be 1 to 5."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Feedback submitted successfully! Thank you.",
		"rating":  input.Rating,
	})
}

// RootIndex handles root URL check to prevent 404 and return API metadata
func RootIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":        "Smart Emotion & Focus Journal Go Backend API",
		"status":      "operational",
		"description": "Next-Gen AI Journal & Performance Monitoring REST service.",
		"health":      "/health",
		"version":     "/version",
	})
}
