package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateMetric registers new LLM usage metric
func CreateMetric(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GetMetrics retrieves LLM monitoring metrics
func GetMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GetScores retrieves decision score statistics
func GetScores(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// CreateErrorLog registers a new LLM error log
func CreateErrorLog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// ClearMetrics deletes LLM metrics history
func ClearMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
