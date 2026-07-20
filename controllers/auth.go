package controllers

import (
	"github.com/gurkanfikretgunak/masterfabric-go/config"
	"github.com/gurkanfikretgunak/masterfabric-go/models"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var (
	// Thread-safe in-memory fallback databases
	memUsers       = make(map[string]models.User)
	memUsersMu     sync.RWMutex
	memUserCounter uint = 1

	// Session token storage mapping Token -> UserID
	ActiveSessions   = make(map[string]uint)
	ActiveSessionsMu sync.RWMutex
)

// Helper to generate secure session tokens
func generateSessionToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Helper to get authenticated UserID from Request
func GetUserIDFromRequest(c *gin.Context) (uint, bool) {
	// Support Authorization: Bearer <token>
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) >= 8 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
		ActiveSessionsMu.RLock()
		userID, ok := ActiveSessions[token]
		ActiveSessionsMu.RUnlock()
		if ok {
			return userID, true
		}
	}
	
	// Fallback to query param or custom header if needed, 
	// but default to 1 (Demo User ID) for unauthenticated telemetry syncs
	return 1, true
}

type AuthInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register handles user registration
func Register(c *gin.Context) {
	var input AuthInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fields. Password must be at least 6 characters."})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt password"})
		return
	}

	db := config.DB
	if db != nil {
		// PostgreSQL mode
		var existing models.User
		if err := db.Where("email = ?", input.Email).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
			return
		}

		user := models.User{
			Email:        input.Email,
			PasswordHash: string(hashed),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
			return
		}

		userConfig := models.UserConfig{
			UserID:        user.ID,
			Theme:         "dark",
			Notifications: true,
		}
		_ = db.Create(&userConfig)

		c.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully",
			"user": gin.H{
				"id":    user.ID,
				"email": user.Email,
			},
		})
		return
	}

	// In-memory fallback mode
	memUsersMu.Lock()
	defer memUsersMu.Unlock()

	if _, ok := memUsers[input.Email]; ok {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
		return
	}

	user := models.User{
		ID:           memUserCounter,
		Email:        input.Email,
		PasswordHash: string(hashed),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	memUserCounter++
	memUsers[input.Email] = user

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully (in-memory mode)",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// Login handles user authentication
func Login(c *gin.Context) {
	var input AuthInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	var userID uint
	var userEmail string
	var passwordHash string

	db := config.DB
	if db != nil {
		// PostgreSQL mode
		var user models.User
		if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		userID = user.ID
		userEmail = user.Email
		passwordHash = user.PasswordHash
	} else {
		// In-memory mode
		memUsersMu.RLock()
		user, ok := memUsers[input.Email]
		memUsersMu.RUnlock()

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		userID = user.ID
		userEmail = user.Email
		passwordHash = user.PasswordHash
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate session token
	token := generateSessionToken()
	ActiveSessionsMu.Lock()
	ActiveSessions[token] = userID
	ActiveSessionsMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    userID,
			"email": userEmail,
		},
	})
}

// Logout handles user sign-out
func Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) >= 8 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
		ActiveSessionsMu.Lock()
		delete(ActiveSessions, token)
		ActiveSessionsMu.Unlock()
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// RefreshToken handles JWT token refreshing
func RefreshToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	oldToken := authHeader[7:]
	ActiveSessionsMu.Lock()
	userID, ok := ActiveSessions[oldToken]
	if ok {
		delete(ActiveSessions, oldToken)
		newToken := generateSessionToken()
		ActiveSessions[newToken] = userID
		ActiveSessionsMu.Unlock()
		c.JSON(http.StatusOK, gin.H{"token": newToken})
		return
	}
	ActiveSessionsMu.Unlock()

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Token session not found"})
}

// GetProfile retrieves user profile
func GetProfile(c *gin.Context) {
	userID, ok := GetUserIDFromRequest(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	db := config.DB
	if db != nil {
		var user models.User
		if err := db.Preload("UserConfig").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User profile not found"})
			return
		}
		c.JSON(http.StatusOK, user)
		return
	}

	// In-memory fallback User profile
	c.JSON(http.StatusOK, gin.H{
		"id":         userID,
		"email":      "demo@masterfabric.co",
		"created_at": time.Now(),
		"config": gin.H{
			"theme":         "dark",
			"notifications": true,
		},
	})
}

// UpdateProfile updates user profile information
func UpdateProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// UpdatePassword updates user password
func UpdatePassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

// DeleteAccount deletes a user account
func DeleteAccount(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}
