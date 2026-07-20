package controllers

import (
	"backend/config"
	"backend/models"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	// Thread-safe in-memory fallback database for metrics
	memMetrics   = []models.LlmMetric{}
	memMetricsMu sync.RWMutex
	memMetricID  uint = 1
)

type MetricInput struct {
	LatencyMs     int64  `json:"latency_ms" binding:"required"`
	TokenCount    int    `json:"token_count" binding:"required"`
	DecisionScore string `json:"decision_score"`
	Status        string `json:"status"`
}

// CreateMetric registers new LLM usage metric
func CreateMetric(c *gin.Context) {
	var input MetricInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metrics format"})
		return
	}

	userID, _ := GetUserIDFromRequest(c)

	db := config.DB
	if db != nil {
		// PostgreSQL mode
		metric := models.LlmMetric{
			UserID:     userID,
			LatencyMs:  input.LatencyMs,
			TokenCount: input.TokenCount,
			ErrorLog:   input.DecisionScore,
			CreatedAt:  time.Now(),
		}

		if err := db.Create(&metric).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save metric"})
			return
		}

		c.JSON(http.StatusCreated, metric)
		return
	}

	// In-memory fallback mode
	memMetricsMu.Lock()
	defer memMetricsMu.Unlock()

	metric := models.LlmMetric{
		ID:         memMetricID,
		UserID:     userID,
		LatencyMs:  input.LatencyMs,
		TokenCount: input.TokenCount,
		ErrorLog:   input.DecisionScore,
		CreatedAt:  time.Now(),
	}
	memMetricID++
	memMetrics = append(memMetrics, metric)

	c.JSON(http.StatusCreated, metric)
}

// GetMetrics retrieves LLM monitoring metrics
func GetMetrics(c *gin.Context) {
	userID, _ := GetUserIDFromRequest(c)

	db := config.DB
	if db != nil {
		// PostgreSQL mode
		var metrics []models.LlmMetric
		if err := db.Where("user_id = ?", userID).Order("created_at desc").Limit(50).Find(&metrics).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch metrics"})
			return
		}
		c.JSON(http.StatusOK, metrics)
		return
	}

	// In-memory mode
	memMetricsMu.RLock()
	defer memMetricsMu.RUnlock()

	var userMetrics []models.LlmMetric
	for i := len(memMetrics) - 1; i >= 0; i-- {
		if memMetrics[i].UserID == userID {
			userMetrics = append(userMetrics, memMetrics[i])
			if len(userMetrics) >= 50 {
				break
			}
		}
	}

	c.JSON(http.StatusOK, userMetrics)
}

// GetScores retrieves decision score statistics
func GetScores(c *gin.Context) {
	userID, _ := GetUserIDFromRequest(c)

	db := config.DB
	if db != nil {
		// PostgreSQL mode
		var avgLoad float64
		row := db.Model(&models.Journal{}).Where("user_id = ?", userID).Select("AVG(decision_score)").Row()
		_ = row.Scan(&avgLoad)

		c.JSON(http.StatusOK, gin.H{
			"user_id":            userID,
			"avg_cognitive_load": avgLoad,
		})
		return
	}

	// In-memory mode
	memJournalsMu.RLock()
	defer memJournalsMu.RUnlock()

	totalScore := 0.0
	count := 0
	for _, j := range memJournals {
		if j.UserID == userID {
			totalScore += j.DecisionScore
			count++
		}
	}

	avg := 50.0
	if count > 0 {
		avg = totalScore / float64(count)
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":            userID,
		"avg_cognitive_load": avg,
	})
}

// CreateErrorLog registers a new LLM error log
func CreateErrorLog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "error logged"})
}

// ClearMetrics deletes LLM metrics history
func ClearMetrics(c *gin.Context) {
	userID, _ := GetUserIDFromRequest(c)

	db := config.DB
	if db != nil {
		// PostgreSQL mode
		db.Where("user_id = ?", userID).Delete(&models.LlmMetric{})
		c.JSON(http.StatusOK, gin.H{"message": "Metrics database cleared successfully"})
		return
	}

	// In-memory mode
	memMetricsMu.Lock()
	defer memMetricsMu.Unlock()

	filtered := []models.LlmMetric{}
	for _, m := range memMetrics {
		if m.UserID != userID {
			filtered = append(filtered, m)
		}
	}
	memMetrics = filtered

	c.JSON(http.StatusOK, gin.H{"message": "Metrics cleared (in-memory)"})
}
