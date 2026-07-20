package controllers

import (
	"github.com/gurkanfikretgunak/masterfabric-go/config"
	"github.com/gurkanfikretgunak/masterfabric-go/models"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	// In-memory config storage: maps UserID -> UserConfig
	memConfigs   = make(map[uint]models.UserConfig)
	memConfigsMu sync.RWMutex
)

// GetConfig retrieves user preferences config
func GetConfig(c *gin.Context) {
	userID, _ := GetUserIDFromRequest(c)

	db := config.DB
	if db != nil {
		var userConfig models.UserConfig
		if err := db.Where("user_id = ?", userID).First(&userConfig).Error; err != nil {
			// If not found, create default config
			userConfig = models.UserConfig{
				UserID:        userID,
				Theme:         "dark",
				Notifications: true,
			}
			_ = db.Create(&userConfig)
		}
		c.JSON(http.StatusOK, userConfig)
		return
	}

	// In-memory fallback
	memConfigsMu.RLock()
	userConfig, ok := memConfigs[userID]
	memConfigsMu.RUnlock()

	if !ok {
		userConfig = models.UserConfig{
			UserID:        userID,
			Theme:         "dark",
			Notifications: true,
		}
		memConfigsMu.Lock()
		memConfigs[userID] = userConfig
		memConfigsMu.Unlock()
	}

	c.JSON(http.StatusOK, userConfig)
}

// UpdateConfig updates user preferences config
func UpdateConfig(c *gin.Context) {
	userID, _ := GetUserIDFromRequest(c)

	type ConfigInput struct {
		Theme         string `json:"theme"`
		Notifications bool   `json:"notifications"`
	}

	var input ConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration parameters"})
		return
	}

	db := config.DB
	if db != nil {
		var userConfig models.UserConfig
		if err := db.Where("user_id = ?", userID).First(&userConfig).Error; err != nil {
			userConfig = models.UserConfig{
				UserID:        userID,
				Theme:         input.Theme,
				Notifications: input.Notifications,
			}
			db.Create(&userConfig)
		} else {
			userConfig.Theme = input.Theme
			userConfig.Notifications = input.Notifications
			db.Save(&userConfig)
		}
		c.JSON(http.StatusOK, userConfig)
		return
	}

	// In-memory fallback
	memConfigsMu.Lock()
	userConfig := models.UserConfig{
		UserID:        userID,
		Theme:         input.Theme,
		Notifications: input.Notifications,
	}
	memConfigs[userID] = userConfig
	memConfigsMu.Unlock()

	c.JSON(http.StatusOK, userConfig)
}
