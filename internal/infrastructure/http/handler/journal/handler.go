package journal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/domain/journal/entity"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/infrastructure/agent"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/infrastructure/learning"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/infrastructure/llm"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/shared/response"
	"golang.org/x/crypto/bcrypt"
)

var (
	// Thread-safe in-memory databases for fallback/demo mode
	memUsers       = make(map[string]UserRecord)
	memUsersMu     sync.RWMutex
	memUserCounter uint = 1

	ActiveSessions   = make(map[string]uint)
	ActiveSessionsMu sync.RWMutex

	memJournals   = []entity.Journal{}
	memJournalsMu sync.RWMutex
	memJournalID  uint = 1

	memConfigs   = make(map[uint]entity.UserConfig)
	memConfigsMu sync.RWMutex

	memMetrics   = []entity.LlmMetric{}
	memMetricsMu sync.RWMutex
	memMetricID  uint = 1
)

type UserRecord struct {
	ID           uint      `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Handler struct {
	analyzer  *llm.Analyzer
	collector *learning.Collector
	harness   *agent.Harness
	db        *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	h := &Handler{
		analyzer:  llm.NewAnalyzer(),
		collector: learning.NewCollector("", 25),
		db:        db,
	}
	h.seedAdminAccount()
	if db != nil {
		h.ensureTables(context.Background())
	}
	return h
}

func (h *Handler) SetHarness(harness *agent.Harness) {
	h.harness = harness
}

func (h *Handler) GetCollector() *learning.Collector {
	return h.collector
}

func (h *Handler) seedAdminAccount() {
	adminEmail := "gurkansenturk@admin.com"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	memUsersMu.Lock()
	if _, exists := memUsers[adminEmail]; !exists {
		memUsers[adminEmail] = UserRecord{
			ID:           memUserCounter,
			Email:        adminEmail,
			PasswordHash: string(hashedPassword),
			IsAdmin:      true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		memUserCounter++
	} else {
		u := memUsers[adminEmail]
		u.IsAdmin = true
		u.PasswordHash = string(hashedPassword)
		memUsers[adminEmail] = u
	}
	memUsersMu.Unlock()
}

func (h *Handler) ensureTables(ctx context.Context) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS journal_users (
			id              BIGSERIAL PRIMARY KEY,
			email           VARCHAR(255) NOT NULL UNIQUE,
			password_hash   VARCHAR(255) NOT NULL,
			is_admin        BOOLEAN NOT NULL DEFAULT false,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at      TIMESTAMPTZ DEFAULT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS active_sessions (
			token           VARCHAR(255) PRIMARY KEY,
			user_id         BIGINT NOT NULL,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS journals (
			id              BIGSERIAL PRIMARY KEY,
			user_id         BIGINT NOT NULL,
			content         TEXT NOT NULL,
			decision_score  DOUBLE PRECISION NOT NULL DEFAULT 50.0,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS user_configs (
			id              BIGSERIAL PRIMARY KEY,
			user_id         BIGINT NOT NULL UNIQUE,
			theme           VARCHAR(50) NOT NULL DEFAULT 'dark',
			notifications   BOOLEAN NOT NULL DEFAULT true
		);`,
		`CREATE TABLE IF NOT EXISTS llm_metrics (
			id              BIGSERIAL PRIMARY KEY,
			user_id         BIGINT NOT NULL,
			latency_ms      BIGINT NOT NULL DEFAULT 0,
			token_count     INT NOT NULL DEFAULT 0,
			error_log       TEXT DEFAULT '',
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_journal_users_email ON journal_users(email);`,
		`CREATE INDEX IF NOT EXISTS idx_journals_user_id ON journals(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_llm_metrics_user_id ON llm_metrics(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_active_sessions_user_id ON active_sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_journal_users_deleted_at ON journal_users(deleted_at);`,
		`ALTER TABLE journal_users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ DEFAULT NULL;`,
		`ALTER TABLE journal_users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT false;`,
	}
	for _, q := range queries {
		_, _ = h.db.Exec(ctx, q)
	}

	// Seed hardcoded admin account into DB
	adminEmail := "gurkansenturk@admin.com"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err == nil {
		var exists bool
		_ = h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM journal_users WHERE email = $1)`, adminEmail).Scan(&exists)
		if !exists {
			_, _ = h.db.Exec(ctx,
				`INSERT INTO journal_users (email, password_hash, is_admin, created_at, updated_at) VALUES ($1, $2, true, NOW(), NOW())`,
				adminEmail, string(hashedPassword))
		} else {
			_, _ = h.db.Exec(ctx,
				`UPDATE journal_users SET is_admin = true, password_hash = $1 WHERE email = $2`,
				string(hashedPassword), adminEmail)
		}
	}
}

func generateSessionToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Handler) getUserIDFromRequest(r *http.Request) (uint, bool) {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) >= 8 && strings.HasPrefix(authHeader, "Bearer ") {
		token := authHeader[7:]
		if h.db != nil {
			var userID uint
			err := h.db.QueryRow(r.Context(), `SELECT user_id FROM active_sessions WHERE token = $1`, token).Scan(&userID)
			if err == nil && userID > 0 {
				return userID, true
			}
		}
		ActiveSessionsMu.RLock()
		userID, ok := ActiveSessions[token]
		ActiveSessionsMu.RUnlock()
		if ok && userID > 0 {
			return userID, true
		}
	}
	return 0, false
}

func (h *Handler) isUserAdmin(r *http.Request) (uint, bool) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok || userID == 0 {
		return 0, false
	}

	if h.db != nil {
		var isAdmin bool
		err := h.db.QueryRow(r.Context(), `SELECT is_admin FROM journal_users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&isAdmin)
		if err == nil && isAdmin {
			return userID, true
		}
		return userID, false
	}

	memUsersMu.RLock()
	defer memUsersMu.RUnlock()
	for _, u := range memUsers {
		if u.ID == userID {
			return userID, u.IsAdmin
		}
	}
	return userID, false
}

func sendError(w http.ResponseWriter, status int, codeStr, msg string) {
	response.JSON(w, status, map[string]interface{}{
		"error":   codeStr,
		"message": msg,
		"code":    status,
	})
}

// --- Common Endpoints ---

func (h *Handler) RootIndex(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"name":        "Smart Emotion & Focus Journal Go Backend API (masterfabric architecture)",
		"status":      "operational",
		"description": "Next-Gen AI Journal & Performance Monitoring REST service with Hexagonal Architecture.",
		"health":      "/health",
		"version":     "/version",
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"message": "success",
	})
}

func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"version": "1.2.5",
		"release": "stable",
		"build":   "2026-07-23",
	})
}

type FeedbackInput struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

func (h *Handler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	var input FeedbackInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Rating < 1 || input.Rating > 5 {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Feedback format invalid. Rating must be 1 to 5.")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Feedback submitted successfully! Thank you.",
		"rating":  input.Rating,
	})
}

// --- Auth Endpoints ---

type AuthInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input AuthInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || len(input.Password) < 6 || input.Email == "" {
		sendError(w, http.StatusBadRequest, "INVALID_FIELDS", "Invalid fields. Password must be at least 6 characters.")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to encrypt password")
		return
	}

	emailClean := strings.TrimSpace(strings.ToLower(input.Email))
	isAdmin := (emailClean == "gurkansenturk@admin.com")

	if h.db != nil {
		// Check if email exists and is active (not soft-deleted)
		var exists bool
		_ = h.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM journal_users WHERE email = $1 AND deleted_at IS NULL)`, emailClean).Scan(&exists)
		if exists {
			sendError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already in use")
			return
		}

		// If a soft-deleted record exists with this email, permanently remove it first
		_, _ = h.db.Exec(r.Context(), `DELETE FROM journal_users WHERE email = $1 AND deleted_at IS NOT NULL`, emailClean)

		var newID uint
		err := h.db.QueryRow(r.Context(),
			`INSERT INTO journal_users (email, password_hash, is_admin, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`,
			emailClean, string(hashed), isAdmin).Scan(&newID)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to register user in database")
			return
		}

		roleStr := "user"
		if isAdmin {
			roleStr = "admin"
		}

		response.JSON(w, http.StatusCreated, map[string]interface{}{
			"message": "User registered successfully",
			"user": map[string]interface{}{
				"id":       newID,
				"email":    emailClean,
				"is_admin": isAdmin,
				"role":     roleStr,
			},
		})
		return
	}

	memUsersMu.Lock()
	defer memUsersMu.Unlock()

	if _, ok := memUsers[emailClean]; ok {
		sendError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already in use")
		return
	}

	user := UserRecord{
		ID:           memUserCounter,
		Email:        emailClean,
		PasswordHash: string(hashed),
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	memUserCounter++
	memUsers[emailClean] = user

	roleStr := "user"
	if isAdmin {
		roleStr = "admin"
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user": map[string]interface{}{
			"id":       user.ID,
			"email":    user.Email,
			"is_admin": isAdmin,
			"role":     roleStr,
		},
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input AuthInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Email == "" || input.Password == "" {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Email and password are required")
		return
	}

	emailClean := strings.TrimSpace(strings.ToLower(input.Email))

	var userID uint
	var dbPasswordHash string
	var isAdmin bool

	if h.db != nil {
		// Only allow login for active (non-deleted) accounts
		err := h.db.QueryRow(r.Context(), `SELECT id, password_hash, is_admin FROM journal_users WHERE email = $1 AND deleted_at IS NULL`, emailClean).Scan(&userID, &dbPasswordHash, &isAdmin)
		if err != nil {
			sendError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
			return
		}
	} else {
		memUsersMu.RLock()
		user, ok := memUsers[emailClean]
		memUsersMu.RUnlock()

		if !ok {
			sendError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
			return
		}
		userID = user.ID
		dbPasswordHash = user.PasswordHash
		isAdmin = user.IsAdmin
	}

	if emailClean == "gurkansenturk@admin.com" {
		isAdmin = true
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbPasswordHash), []byte(input.Password)); err != nil {
		sendError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	token := generateSessionToken()

	if h.db != nil {
		_, _ = h.db.Exec(r.Context(), `INSERT INTO active_sessions (token, user_id, created_at) VALUES ($1, $2, NOW())`, token, userID)
	}

	ActiveSessionsMu.Lock()
	ActiveSessions[token] = userID
	ActiveSessionsMu.Unlock()

	roleStr := "user"
	if isAdmin {
		roleStr = "admin"
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       userID,
			"email":    emailClean,
			"is_admin": isAdmin,
			"role":     roleStr,
		},
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) >= 8 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
		if h.db != nil {
			_, _ = h.db.Exec(r.Context(), `DELETE FROM active_sessions WHERE token = $1`, token)
		}
		ActiveSessionsMu.Lock()
		delete(ActiveSessions, token)
		ActiveSessionsMu.Unlock()
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Logout successful"})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
		return
	}

	oldToken := authHeader[7:]
	userID, ok := h.getUserIDFromRequest(r)

	if !ok || userID == 0 {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token session not found")
		return
	}

	newToken := generateSessionToken()

	if h.db != nil {
		_, _ = h.db.Exec(r.Context(), `DELETE FROM active_sessions WHERE token = $1`, oldToken)
		_, _ = h.db.Exec(r.Context(), `INSERT INTO active_sessions (token, user_id, created_at) VALUES ($1, $2, NOW())`, newToken, userID)
	}

	ActiveSessionsMu.Lock()
	delete(ActiveSessions, oldToken)
	ActiveSessions[newToken] = userID
	ActiveSessionsMu.Unlock()

	response.JSON(w, http.StatusOK, map[string]string{"token": newToken})
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	email := "demo@masterfabric.co"
	createdAt := time.Now()
	isAdmin := false

	if h.db != nil {
		_ = h.db.QueryRow(r.Context(), `SELECT email, is_admin, created_at FROM journal_users WHERE id = $1`, userID).Scan(&email, &isAdmin, &createdAt)
	} else {
		memUsersMu.RLock()
		for _, u := range memUsers {
			if u.ID == userID {
				email = u.Email
				isAdmin = u.IsAdmin
				createdAt = u.CreatedAt
				break
			}
		}
		memUsersMu.RUnlock()
	}

	if email == "gurkansenturk@admin.com" {
		isAdmin = true
	}

	roleStr := "user"
	if isAdmin {
		roleStr = "admin"
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"id":         userID,
		"email":      email,
		"is_admin":   isAdmin,
		"role":       roleStr,
		"created_at": createdAt,
		"config": map[string]interface{}{
			"theme":         "dark",
			"notifications": true,
		},
	})
}

// --- Admin User Management Endpoints ---

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	_, isAdmin := h.isUserAdmin(r)
	if !isAdmin {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Admin privileges required")
		return
	}

	type UserDTO struct {
		ID        uint      `json:"id"`
		Email     string    `json:"email"`
		IsAdmin   bool      `json:"is_admin"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
	}

	users := []UserDTO{}

	if h.db != nil {
		rows, err := h.db.Query(r.Context(), `SELECT id, email, is_admin, created_at FROM journal_users WHERE deleted_at IS NULL ORDER BY id ASC`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var u UserDTO
				if err := rows.Scan(&u.ID, &u.Email, &u.IsAdmin, &u.CreatedAt); err == nil {
					if u.IsAdmin || u.Email == "gurkansenturk@admin.com" {
						u.IsAdmin = true
						u.Role = "admin"
					} else {
						u.Role = "user"
					}
					users = append(users, u)
				}
			}
			response.JSON(w, http.StatusOK, users)
			return
		}
	}

	memUsersMu.RLock()
	defer memUsersMu.RUnlock()
	for _, u := range memUsers {
		roleStr := "user"
		isAdminVal := u.IsAdmin
		if u.Email == "gurkansenturk@admin.com" {
			isAdminVal = true
		}
		if isAdminVal {
			roleStr = "admin"
		}
		users = append(users, UserDTO{
			ID:        u.ID,
			Email:     u.Email,
			IsAdmin:   isAdminVal,
			Role:      roleStr,
			CreatedAt: u.CreatedAt,
		})
	}

	response.JSON(w, http.StatusOK, users)
}

type UpdateUserRoleInput struct {
	UserID  uint `json:"user_id"`
	IsAdmin bool `json:"is_admin"`
}

func (h *Handler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	_, isAdmin := h.isUserAdmin(r)
	if !isAdmin {
		sendError(w, http.StatusForbidden, "FORBIDDEN", "Admin privileges required")
		return
	}

	var input UpdateUserRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.UserID == 0 {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "User ID and admin status are required")
		return
	}

	if h.db != nil {
		_, err := h.db.Exec(r.Context(), `UPDATE journal_users SET is_admin = $1, updated_at = NOW() WHERE id = $2`, input.IsAdmin, input.UserID)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update user role in database")
			return
		}
	}

	memUsersMu.Lock()
	for email, u := range memUsers {
		if u.ID == input.UserID {
			u.IsAdmin = input.IsAdmin
			u.UpdatedAt = time.Now()
			memUsers[email] = u
			break
		}
	}
	memUsersMu.Unlock()

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":  "User admin role updated successfully",
		"user_id":  input.UserID,
		"is_admin": input.IsAdmin,
	})
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "Profile updated successfully"})
}

func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	authHeader := r.Header.Get("Authorization")
	var token string
	if len(authHeader) >= 8 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	// 1. Delete from PostgreSQL database using a transaction for atomicity
	if h.db != nil {
		ctx := r.Context()

		tx, txErr := h.db.Begin(ctx)
		if txErr != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to start transaction for account deletion")
			return
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		// Delete all dependent rows first
		if _, err := tx.Exec(ctx, `DELETE FROM active_sessions WHERE user_id = $1`, userID); err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to clear sessions")
			return
		}
		if _, err := tx.Exec(ctx, `DELETE FROM journals WHERE user_id = $1`, userID); err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to clear journals")
			return
		}
		if _, err := tx.Exec(ctx, `DELETE FROM user_configs WHERE user_id = $1`, userID); err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to clear config")
			return
		}
		if _, err := tx.Exec(ctx, `DELETE FROM llm_metrics WHERE user_id = $1`, userID); err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to clear metrics")
			return
		}

		// Soft-delete the user (mark as deleted instead of removing the row)
		result, err := tx.Exec(ctx, `UPDATE journal_users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, userID)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete account")
			return
		}
		if result.RowsAffected() == 0 {
			sendError(w, http.StatusNotFound, "NOT_FOUND", "Account not found or already deleted")
			return
		}

		if err := tx.Commit(ctx); err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to commit account deletion")
			return
		}
	}

	// 2. Clear in-memory data structures
	ActiveSessionsMu.Lock()
	if token != "" {
		delete(ActiveSessions, token)
	}
	for t, uid := range ActiveSessions {
		if uid == userID {
			delete(ActiveSessions, t)
		}
	}
	ActiveSessionsMu.Unlock()

	memUsersMu.Lock()
	for e, u := range memUsers {
		if u.ID == userID {
			delete(memUsers, e)
		}
	}
	memUsersMu.Unlock()

	memJournalsMu.Lock()
	filteredJournals := []entity.Journal{}
	for _, j := range memJournals {
		if j.UserID != userID {
			filteredJournals = append(filteredJournals, j)
		}
	}
	memJournals = filteredJournals
	memJournalsMu.Unlock()

	memConfigsMu.Lock()
	delete(memConfigs, userID)
	memConfigsMu.Unlock()

	memMetricsMu.Lock()
	filteredMetrics := []entity.LlmMetric{}
	for _, m := range memMetrics {
		if m.UserID != userID {
			filteredMetrics = append(filteredMetrics, m)
		}
	}
	memMetrics = filteredMetrics
	memMetricsMu.Unlock()

	response.JSON(w, http.StatusOK, map[string]string{"message": "Account deleted successfully"})
}

// --- User Config Endpoints ---

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	if h.db != nil {
		var cfg entity.UserConfig
		err := h.db.QueryRow(r.Context(), `SELECT id, user_id, theme, notifications FROM user_configs WHERE user_id = $1`, userID).Scan(&cfg.ID, &cfg.UserID, &cfg.Theme, &cfg.Notifications)
		if err == nil {
			response.JSON(w, http.StatusOK, cfg)
			return
		}
		cfg = entity.UserConfig{
			UserID:        userID,
			Theme:         "dark",
			Notifications: true,
		}
		_ = h.db.QueryRow(r.Context(), `INSERT INTO user_configs (user_id, theme, notifications) VALUES ($1, $2, $3) RETURNING id`, userID, cfg.Theme, cfg.Notifications).Scan(&cfg.ID)
		response.JSON(w, http.StatusOK, cfg)
		return
	}

	memConfigsMu.RLock()
	userConfig, okConfig := memConfigs[userID]
	memConfigsMu.RUnlock()

	if !okConfig {
		userConfig = entity.UserConfig{
			UserID:        userID,
			Theme:         "dark",
			Notifications: true,
		}
		memConfigsMu.Lock()
		memConfigs[userID] = userConfig
		memConfigsMu.Unlock()
	}

	response.JSON(w, http.StatusOK, userConfig)
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	type ConfigInput struct {
		Theme         string `json:"theme"`
		Notifications bool   `json:"notifications"`
	}

	var input ConfigInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid configuration parameters")
		return
	}

	userConfig := entity.UserConfig{
		UserID:        userID,
		Theme:         input.Theme,
		Notifications: input.Notifications,
	}

	if h.db != nil {
		_, err := h.db.Exec(r.Context(),
			`INSERT INTO user_configs (user_id, theme, notifications) VALUES ($1, $2, $3)
			 ON CONFLICT (user_id) DO UPDATE SET theme = EXCLUDED.theme, notifications = EXCLUDED.notifications`,
			userID, input.Theme, input.Notifications)
		if err == nil {
			response.JSON(w, http.StatusOK, userConfig)
			return
		}
	}

	memConfigsMu.Lock()
	memConfigs[userID] = userConfig
	memConfigsMu.Unlock()

	response.JSON(w, http.StatusOK, userConfig)
}

// --- Journal Endpoints ---

type AnalyzeInput struct {
	Content string `json:"content"`
}

func (h *Handler) AnalyzeJournal(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var input AnalyzeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Content) == "" {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Journal content is required for analysis")
		return
	}

	result, err := h.analyzer.Analyze(r.Context(), input.Content)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "ANALYSIS_FAILED", "Failed to analyze journal entry")
		return
	}

	now := time.Now()
	metric := entity.LlmMetric{
		UserID:     userID,
		LatencyMs:  result.Metrics.LatencyMs,
		TokenCount: result.Metrics.TotalTokens,
		ErrorLog:   fmt.Sprintf("%d%%", result.CognitiveLoad),
		CreatedAt:  now,
	}

	if h.db != nil {
		var newID uint
		_ = h.db.QueryRow(r.Context(),
			`INSERT INTO llm_metrics (user_id, latency_ms, token_count, error_log, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			userID, result.Metrics.LatencyMs, result.Metrics.TotalTokens, metric.ErrorLog, now).Scan(&newID)
		metric.ID = newID
	} else {
		memMetricsMu.Lock()
		metric.ID = memMetricID
		memMetricID++
		memMetrics = append(memMetrics, metric)
		memMetricsMu.Unlock()
	}

	// Record interaction in continuous learning collector
	if h.collector != nil {
		h.collector.RecordAnalysis(userID, input.Content, result.CognitiveLoad, result.Suggestion)
	}

	response.JSON(w, http.StatusOK, result)
}

type JournalInput struct {
	Content     string `json:"content"`
	LlmResponse string `json:"llm_response"`
}

func (h *Handler) CreateJournal(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var input JournalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Content == "" {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid input fields")
		return
	}

	decisionScore := 50.0
	re := regexp.MustCompile(`Cognitive\s+Load\s+Score:\s*(\d+)`)
	match := re.FindStringSubmatch(input.LlmResponse)
	if len(match) > 1 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			decisionScore = val
		}
	} else {
		rePercent := regexp.MustCompile(`(\d+)%`)
		matchPercent := rePercent.FindStringSubmatch(input.LlmResponse)
		if len(matchPercent) > 1 {
			if val, err := strconv.ParseFloat(matchPercent[1], 64); err == nil {
				decisionScore = val
			}
		}
	}

	now := time.Now()
	j := entity.Journal{
		UserID:        userID,
		Content:       input.Content,
		DecisionScore: decisionScore,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if h.db != nil {
		var newID uint
		err := h.db.QueryRow(r.Context(),
			`INSERT INTO journals (user_id, content, decision_score, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			userID, input.Content, decisionScore, now, now).Scan(&newID)
		if err == nil {
			j.ID = newID
			response.JSON(w, http.StatusCreated, j)
			return
		}
	}

	memJournalsMu.Lock()
	defer memJournalsMu.Unlock()

	j.ID = memJournalID
	memJournalID++
	memJournals = append(memJournals, j)

	response.JSON(w, http.StatusCreated, j)
}

func (h *Handler) GetJournals(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	if h.db != nil {
		rows, err := h.db.Query(r.Context(),
			`SELECT id, user_id, content, decision_score, created_at, updated_at FROM journals WHERE user_id = $1 ORDER BY created_at DESC`, userID)
		if err == nil {
			defer rows.Close()
			userJournals := []entity.Journal{}
			for rows.Next() {
				var j entity.Journal
				if err := rows.Scan(&j.ID, &j.UserID, &j.Content, &j.DecisionScore, &j.CreatedAt, &j.UpdatedAt); err == nil {
					userJournals = append(userJournals, j)
				}
			}
			response.JSON(w, http.StatusOK, userJournals)
			return
		}
	}

	memJournalsMu.RLock()
	defer memJournalsMu.RUnlock()

	userJournals := []entity.Journal{}
	for i := len(memJournals) - 1; i >= 0; i-- {
		if memJournals[i].UserID == userID {
			userJournals = append(userJournals, memJournals[i])
		}
	}

	response.JSON(w, http.StatusOK, userJournals)
}

// --- Monitoring & Telemetry Endpoints ---

type MetricInput struct {
	LatencyMs     int64  `json:"latency_ms"`
	TokenCount    int    `json:"token_count"`
	DecisionScore string `json:"decision_score"`
	Status        string `json:"status"`
}

func (h *Handler) CreateMetric(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var input MetricInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid metrics format")
		return
	}

	now := time.Now()
	metric := entity.LlmMetric{
		UserID:     userID,
		LatencyMs:  input.LatencyMs,
		TokenCount: input.TokenCount,
		ErrorLog:   input.DecisionScore,
		CreatedAt:  now,
	}

	if h.db != nil {
		var newID uint
		err := h.db.QueryRow(r.Context(),
			`INSERT INTO llm_metrics (user_id, latency_ms, token_count, error_log, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			userID, input.LatencyMs, input.TokenCount, input.DecisionScore, now).Scan(&newID)
		if err == nil {
			metric.ID = newID
			response.JSON(w, http.StatusCreated, metric)
			return
		}
	}

	memMetricsMu.Lock()
	defer memMetricsMu.Unlock()

	metric.ID = memMetricID
	memMetricID++
	memMetrics = append(memMetrics, metric)

	response.JSON(w, http.StatusCreated, metric)
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	if h.db != nil {
		rows, err := h.db.Query(r.Context(),
			`SELECT id, user_id, latency_ms, token_count, error_log, created_at FROM llm_metrics WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
		if err == nil {
			defer rows.Close()
			userMetrics := []entity.LlmMetric{}
			for rows.Next() {
				var m entity.LlmMetric
				if err := rows.Scan(&m.ID, &m.UserID, &m.LatencyMs, &m.TokenCount, &m.ErrorLog, &m.CreatedAt); err == nil {
					userMetrics = append(userMetrics, m)
				}
			}
			response.JSON(w, http.StatusOK, userMetrics)
			return
		}
	}

	memMetricsMu.RLock()
	defer memMetricsMu.RUnlock()

	userMetrics := []entity.LlmMetric{}
	for i := len(memMetrics) - 1; i >= 0; i-- {
		if memMetrics[i].UserID == userID {
			userMetrics = append(userMetrics, memMetrics[i])
			if len(userMetrics) >= 50 {
				break
			}
		}
	}

	response.JSON(w, http.StatusOK, userMetrics)
}

func (h *Handler) GetScores(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	if h.db != nil {
		var avg float64
		err := h.db.QueryRow(r.Context(),
			`SELECT COALESCE(AVG(decision_score), 50.0) FROM journals WHERE user_id = $1`, userID).Scan(&avg)
		if err == nil {
			response.JSON(w, http.StatusOK, map[string]interface{}{
				"user_id":            userID,
				"avg_cognitive_load": avg,
			})
			return
		}
	}

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

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user_id":            userID,
		"avg_cognitive_load": avg,
	})
}

func (h *Handler) CreateErrorLog(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "error logged"})
}

func (h *Handler) ClearMetrics(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	if h.db != nil {
		_, _ = h.db.Exec(r.Context(), `DELETE FROM llm_metrics WHERE user_id = $1`, userID)
	}

	memMetricsMu.Lock()
	defer memMetricsMu.Unlock()

	filtered := []entity.LlmMetric{}
	for _, m := range memMetrics {
		if m.UserID != userID {
			filtered = append(filtered, m)
		}
	}
	memMetrics = filtered

	response.JSON(w, http.StatusOK, map[string]string{"message": "Metrics cleared"})
}

// --- Prompt Optimizer Endpoints ---

type OptimizePromptInput struct {
	Prompt            string `json:"prompt"`
	Template          string `json:"template"`
	CustomInstruction string `json:"custom_instruction,omitempty"`
}

func (i *OptimizePromptInput) UnmarshalJSON(data []byte) error {
	type rawInput struct {
		Prompt                 string `json:"prompt"`
		Template               string `json:"template"`
		CustomInstruction      string `json:"custom_instruction"`
		CamelCustomInstruction string `json:"customInstruction"`
	}
	var raw rawInput
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	i.Prompt = raw.Prompt
	i.Template = raw.Template
	if raw.CustomInstruction != "" {
		i.CustomInstruction = raw.CustomInstruction
	} else {
		i.CustomInstruction = raw.CamelCustomInstruction
	}
	return nil
}

func (h *Handler) OptimizePrompt(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var input OptimizePromptInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Prompt) == "" {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Prompt text is required for optimization")
		return
	}

	if input.Template == "" {
		input.Template = "accurate"
	}

	result, err := h.analyzer.OptimizePrompt(r.Context(), input.Prompt, input.Template, input.CustomInstruction)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "OPTIMIZATION_FAILED", "Failed to optimize prompt")
		return
	}

	// Persist optimization to DB if pool exists
	if h.db != nil {
		_, _ = h.db.Exec(r.Context(),
			`INSERT INTO prompt_optimizations (user_id, original_prompt, optimized_prompt, template, custom_instruction, original_tokens, optimized_tokens, latency_ms, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
			userID, result.OriginalPrompt, result.OptimizedPrompt, result.Template, result.CustomInstruction, result.OriginalTokens, result.OptimizedTokens, result.Metrics.LatencyMs)
	}

	// Record interaction in continuous learning collector
	if h.collector != nil {
		h.collector.RecordOptimization(userID, result.OriginalPrompt, result.OptimizedPrompt, result.Template, result.ThinkingProcess)
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetPromptHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	type HistoryRecord struct {
		ID                uint      `json:"id"`
		OriginalPrompt    string    `json:"original_prompt"`
		OptimizedPrompt   string    `json:"optimized_prompt"`
		Template          string    `json:"template"`
		CustomInstruction string    `json:"custom_instruction,omitempty"`
		OriginalTokens    int       `json:"original_tokens"`
		OptimizedTokens   int       `json:"optimized_tokens"`
		LatencyMs         int64     `json:"latency_ms"`
		CreatedAt         time.Time `json:"created_at"`
	}

	history := []HistoryRecord{}

	if h.db != nil {
		rows, err := h.db.Query(r.Context(),
			`SELECT id, original_prompt, optimized_prompt, template, COALESCE(custom_instruction, ''), original_tokens, optimized_tokens, latency_ms, created_at 
			 FROM prompt_optimizations WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var rec HistoryRecord
				if err := rows.Scan(&rec.ID, &rec.OriginalPrompt, &rec.OptimizedPrompt, &rec.Template, &rec.CustomInstruction, &rec.OriginalTokens, &rec.OptimizedTokens, &rec.LatencyMs, &rec.CreatedAt); err == nil {
					history = append(history, rec)
				}
			}
		}
	}

	response.JSON(w, http.StatusOK, history)
}

func (h *Handler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	templates := []map[string]interface{}{
		{
			"id":          "accurate",
			"name":        "En Doğru Sonuç",
			"description": "Prompt'u en hassas, net ve eksiksiz yanıt verecek şekilde yapılandırır.",
			"icon":        "Target",
			"badge":       "High Precision",
		},
		{
			"id":          "minimal",
			"name":        "Minimum Token",
			"description": "Gereksiz kelimeleri çıkartıp token tüketimini ve maliyeti en aza indirir.",
			"icon":        "Zap",
			"badge":       "Token Saver",
		},
		{
			"id":          "creative",
			"name":        "En Yaratıcı",
			"description": "Modelin zengin, hayal gücü yüksek ve detaylı çıktılar üretmesini sağlar.",
			"icon":        "Sparkles",
			"badge":       "High Creativity",
		},
		{
			"id":          "code",
			"name":        "Kod Odaklı",
			"description": "Yazılım geliştirme görevleri için üretim kalitesinde kod spesifikasyonuna çevirir.",
			"icon":        "Code",
			"badge":       "Developer Pack",
		},
		{
			"id":          "academic",
			"name":        "Akademik & Araştırma",
			"description": "Resmi terminoloji ve metodolojik analiz yapısına dönüştürür.",
			"icon":        "GraduationCap",
			"badge":       "Research Ready",
		},
		{
			"id":          "custom",
			"name":        "Özel Talimat",
			"description": "Sizin belirteceğiniz özel optimizasyon kuralına göre prompt'u yeniden biçimlendirir.",
			"icon":        "Sliders",
			"badge":       "Custom Rule",
		},
	}

	response.JSON(w, http.StatusOK, templates)
}

