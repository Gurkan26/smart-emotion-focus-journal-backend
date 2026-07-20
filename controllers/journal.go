package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateJournal creates a new journal entry
func CreateJournal(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GetJournals retrieves journal entries
func GetJournals(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
