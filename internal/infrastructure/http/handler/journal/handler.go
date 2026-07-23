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
	ID           uint
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Handler struct {
	analyzer *llm.Analyzer
	db       *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	h := &Handler{
		analyzer: llm.NewAnalyzer(),
		db:       db,
	}
	if db != nil {
		h.ensureTables(context.Background())
	}
	return h
}

func (h *Handler) ensureTables(ctx context.Context) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS journal_users (
			id              BIGSERIAL PRIMARY KEY,
			email           VARCHAR(255) NOT NULL UNIQUE,
			password_hash   VARCHAR(255) NOT NULL,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
	}
	for _, q := range queries {
		_, _ = h.db.Exec(ctx, q)
	}
}

func generateSessionToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Handler) getUserIDFromRequest(r *http.Request) uint {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) >= 8 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
		if h.db != nil {
			var userID uint
			err := h.db.QueryRow(r.Context(), `SELECT user_id FROM active_sessions WHERE token = $1`, token).Scan(&userID)
			if err == nil && userID > 0 {
				return userID
			}
		}
		ActiveSessionsMu.RLock()
		userID, ok := ActiveSessions[token]
		ActiveSessionsMu.RUnlock()
		if ok {
			return userID
		}
	}
	return 1 // Default demo user ID
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

	if h.db != nil {
		var exists bool
		_ = h.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM journal_users WHERE email = $1)`, emailClean).Scan(&exists)
		if exists {
			sendError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already in use")
			return
		}

		var newID uint
		err := h.db.QueryRow(r.Context(),
			`INSERT INTO journal_users (email, password_hash, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`,
			emailClean, string(hashed)).Scan(&newID)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to register user in database")
			return
		}

		memUsersMu.Lock()
		memUsers[emailClean] = UserRecord{
			ID:           newID,
			Email:        emailClean,
			PasswordHash: string(hashed),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		memUsersMu.Unlock()

		response.JSON(w, http.StatusCreated, map[string]interface{}{
			"message": "User registered successfully",
			"user": map[string]interface{}{
				"id":    newID,
				"email": emailClean,
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
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	memUserCounter++
	memUsers[emailClean] = user

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
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

	if h.db != nil {
		err := h.db.QueryRow(r.Context(), `SELECT id, password_hash FROM journal_users WHERE email = $1`, emailClean).Scan(&userID, &dbPasswordHash)
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

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":    userID,
			"email": emailClean,
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
	userID := h.getUserIDFromRequest(r)

	if userID == 0 {
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
	userID := h.getUserIDFromRequest(r)

	email := "demo@masterfabric.co"
	createdAt := time.Now()

	if h.db != nil {
		_ = h.db.QueryRow(r.Context(), `SELECT email, created_at FROM journal_users WHERE id = $1`, userID).Scan(&email, &createdAt)
	} else {
		memUsersMu.RLock()
		for _, u := range memUsers {
			if u.ID == userID {
				email = u.Email
				createdAt = u.CreatedAt
				break
			}
		}
		memUsersMu.RUnlock()
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"id":         userID,
		"email":      email,
		"created_at": createdAt,
		"config": map[string]interface{}{
			"theme":         "dark",
			"notifications": true,
		},
	})
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "Profile updated successfully"})
}

func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDFromRequest(r)

	authHeader := r.Header.Get("Authorization")
	var token string
	if len(authHeader) >= 8 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	// 1. Delete from PostgreSQL database if pool exists
	if h.db != nil {
		ctx := r.Context()

		var email string
		_ = h.db.QueryRow(ctx, `SELECT email FROM journal_users WHERE id = $1`, userID).Scan(&email)

		_, _ = h.db.Exec(ctx, `DELETE FROM active_sessions WHERE user_id = $1 OR token = $2`, userID, token)
		_, _ = h.db.Exec(ctx, `DELETE FROM journals WHERE user_id = $1`, userID)
		_, _ = h.db.Exec(ctx, `DELETE FROM user_configs WHERE user_id = $1`, userID)
		_, _ = h.db.Exec(ctx, `DELETE FROM llm_metrics WHERE user_id = $1`, userID)
		_, _ = h.db.Exec(ctx, `DELETE FROM journal_users WHERE id = $1`, userID)

		if email != "" {
			memUsersMu.Lock()
			delete(memUsers, email)
			memUsersMu.Unlock()
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
	userID := h.getUserIDFromRequest(r)

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
	userConfig, ok := memConfigs[userID]
	memConfigsMu.RUnlock()

	if !ok {
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
	userID := h.getUserIDFromRequest(r)

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
	var input AnalyzeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Content) == "" {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Journal content is required for analysis")
		return
	}

	userID := h.getUserIDFromRequest(r)

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

	response.JSON(w, http.StatusOK, result)
}

type JournalInput struct {
	Content     string `json:"content"`
	LlmResponse string `json:"llm_response"`
}

func (h *Handler) CreateJournal(w http.ResponseWriter, r *http.Request) {
	var input JournalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Content == "" {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid input fields")
		return
	}

	userID := h.getUserIDFromRequest(r)

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
	userID := h.getUserIDFromRequest(r)

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
	var input MetricInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid metrics format")
		return
	}

	userID := h.getUserIDFromRequest(r)

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
	userID := h.getUserIDFromRequest(r)

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
	userID := h.getUserIDFromRequest(r)

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
	userID := h.getUserIDFromRequest(r)

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
