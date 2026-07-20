package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns server status
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GetVersion returns current application version
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// SubmitFeedback handles user feedback submission
func SubmitFeedback(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
