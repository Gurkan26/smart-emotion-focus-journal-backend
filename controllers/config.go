package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetConfig retrieves user preferences config
func GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// UpdateConfig updates user preferences config
func UpdateConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
