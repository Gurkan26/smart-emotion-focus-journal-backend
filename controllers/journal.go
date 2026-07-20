package controllers

import (
	"backend/config"
	"backend/models"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	// Thread-safe in-memory fallback database for journals
	memJournals   = []models.Journal{}
	memJournalsMu sync.RWMutex
	memJournalID  uint = 1
)

type JournalInput struct {
	Content     string `json:"content" binding:"required"`
	LlmResponse string `json:"llm_response" binding:"required"`
}

// CreateJournal creates a new journal entry
func CreateJournal(c *gin.Context) {
	var input JournalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input fields"})
		return
	}

	// Retrieve authenticated User ID
	userID, _ := GetUserIDFromRequest(c)

	// Parse decision score from LlmResponse (e.g. "Cognitive Load Score: 75% - Take a break!")
	decisionScore := 50.0 // default fallback
	re := regexp.MustCompile(`Cognitive\s+Load\s+Score:\s*(\d+)`)
	match := re.FindStringSubmatch(input.LlmResponse)
	if len(match) > 1 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			decisionScore = val
		}
	} else {
		// Fallback for general percentage match
		rePercent := regexp.MustCompile(`(\d+)%`)
		matchPercent := rePercent.FindStringSubmatch(input.LlmResponse)
		if len(matchPercent) > 1 {
			if val, err := strconv.ParseFloat(matchPercent[1], 64); err == nil {
				decisionScore = val
			}
		}
	}

	db := config.DB
	if db != nil {
		// PostgreSQL mode
		journal := models.Journal{
			UserID:        userID,
			Content:       input.Content,
			DecisionScore: decisionScore,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := db.Create(&journal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save journal to database"})
			return
		}

		c.JSON(http.StatusCreated, journal)
		return
	}

	// In-memory fallback mode
	memJournalsMu.Lock()
	defer memJournalsMu.Unlock()

	journal := models.Journal{
		ID:            memJournalID,
		UserID:        userID,
		Content:       input.Content,
		DecisionScore: decisionScore,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	memJournalID++
	memJournals = append(memJournals, journal)

	c.JSON(http.StatusCreated, journal)
}

// GetJournals retrieves journal entries
func GetJournals(c *gin.Context) {
	userID, _ := GetUserIDFromRequest(c)

	db := config.DB
	if db != nil {
		// PostgreSQL mode
		var journals []models.Journal
		if err := db.Where("user_id = ?", userID).Order("created_at desc").Find(&journals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch journals"})
			return
		}
		c.JSON(http.StatusOK, journals)
		return
	}

	// In-memory fallback mode
	memJournalsMu.RLock()
	defer memJournalsMu.RUnlock()

	var userJournals []models.Journal
	for i := len(memJournals) - 1; i >= 0; i-- { // Reverse chronological order
		if memJournals[i].UserID == userID {
			userJournals = append(userJournals, memJournals[i])
		}
	}

	c.JSON(http.StatusOK, userJournals)
}
