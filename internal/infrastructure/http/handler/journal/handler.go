package journal

import (
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
}

func NewHandler() *Handler {
	return &Handler{
		analyzer: llm.NewAnalyzer(),
	}
}

func generateSessionToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func getUserIDFromRequest(r *http.Request) uint {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) >= 8 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
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

	memUsersMu.Lock()
	defer memUsersMu.Unlock()

	if _, ok := memUsers[input.Email]; ok {
		sendError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already in use")
		return
	}

	user := UserRecord{
		ID:           memUserCounter,
		Email:        input.Email,
		PasswordHash: string(hashed),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	memUserCounter++
	memUsers[input.Email] = user

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

	memUsersMu.RLock()
	user, ok := memUsers[input.Email]
	memUsersMu.RUnlock()

	if !ok {
		sendError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		sendError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	token := generateSessionToken()
	ActiveSessionsMu.Lock()
	ActiveSessions[token] = user.ID
	ActiveSessionsMu.Unlock()

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) >= 8 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
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
	ActiveSessionsMu.Lock()
	userID, ok := ActiveSessions[oldToken]
	if ok {
		delete(ActiveSessions, oldToken)
		newToken := generateSessionToken()
		ActiveSessions[newToken] = userID
		ActiveSessionsMu.Unlock()
		response.JSON(w, http.StatusOK, map[string]string{"token": newToken})
		return
	}
	ActiveSessionsMu.Unlock()

	sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token session not found")
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"id":         userID,
		"email":      "demo@masterfabric.co",
		"created_at": time.Now(),
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
	response.JSON(w, http.StatusOK, map[string]string{"message": "Account deleted successfully"})
}

// --- User Config Endpoints ---

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)

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
	userID := getUserIDFromRequest(r)

	type ConfigInput struct {
		Theme         string `json:"theme"`
		Notifications bool   `json:"notifications"`
	}

	var input ConfigInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid configuration parameters")
		return
	}

	memConfigsMu.Lock()
	userConfig := entity.UserConfig{
		UserID:        userID,
		Theme:         input.Theme,
		Notifications: input.Notifications,
	}
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

	userID := getUserIDFromRequest(r)

	result, err := h.analyzer.Analyze(r.Context(), input.Content)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "ANALYSIS_FAILED", "Failed to analyze journal entry")
		return
	}

	// Automatically record metric telemetry
	memMetricsMu.Lock()
	metric := entity.LlmMetric{
		ID:         memMetricID,
		UserID:     userID,
		LatencyMs:  result.Metrics.LatencyMs,
		TokenCount: result.Metrics.TotalTokens,
		ErrorLog:   fmt.Sprintf("%d%%", result.CognitiveLoad),
		CreatedAt:  time.Now(),
	}
	memMetricID++
	memMetrics = append(memMetrics, metric)
	memMetricsMu.Unlock()

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

	userID := getUserIDFromRequest(r)

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

	memJournalsMu.Lock()
	defer memJournalsMu.Unlock()

	j := entity.Journal{
		ID:            memJournalID,
		UserID:        userID,
		Content:       input.Content,
		DecisionScore: decisionScore,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	memJournalID++
	memJournals = append(memJournals, j)

	response.JSON(w, http.StatusCreated, j)
}

func (h *Handler) GetJournals(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)

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

	userID := getUserIDFromRequest(r)

	memMetricsMu.Lock()
	defer memMetricsMu.Unlock()

	metric := entity.LlmMetric{
		ID:         memMetricID,
		UserID:     userID,
		LatencyMs:  input.LatencyMs,
		TokenCount: input.TokenCount,
		ErrorLog:   input.DecisionScore,
		CreatedAt:  time.Now(),
	}
	memMetricID++
	memMetrics = append(memMetrics, metric)

	response.JSON(w, http.StatusCreated, metric)
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)

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
	userID := getUserIDFromRequest(r)

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
	userID := getUserIDFromRequest(r)

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
