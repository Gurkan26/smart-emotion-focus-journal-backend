package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/infrastructure/mcp"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/shared/response"
)

type FineTuneTelemetryDTO struct {
	ID           int64     `json:"id,omitempty"`
	Status       string    `json:"status"`        // "IDLE", "RUNNING", "COMPLETED", "FAILED"
	CurrentEpoch int       `json:"current_epoch"`
	TotalEpochs  int       `json:"total_epochs"`
	Step         int       `json:"step"`
	TotalSteps   int       `json:"total_steps"`
	ProgressPct  float64   `json:"progress_pct"`
	Loss         float64   `json:"loss"`
	LearningRate float64   `json:"learning_rate"`
	VramGB       float64   `json:"vram_gb"`
	AdapterName  string    `json:"adapter_name"`
	Message      string    `json:"message"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Handler struct {
	db              *pgxpool.Pool
	wikiMCP         *mcp.DeepWikiMCP
	mcpSuite        *mcp.MCPServerSuite
	telemetryMu     sync.RWMutex
	latestTelemetry FineTuneTelemetryDTO
}

func NewHandler(db *pgxpool.Pool) *Handler {
	h := &Handler{
		db:       db,
		wikiMCP:  mcp.NewDeepWikiMCP(""),
		mcpSuite: mcp.NewMCPServerSuite(),
		latestTelemetry: FineTuneTelemetryDTO{
			Status:       "IDLE",
			CurrentEpoch: 0,
			TotalEpochs:  5,
			ProgressPct:  0.0,
			Loss:         0.0,
			LearningRate: 0.0002,
			VramGB:       0.0,
			AdapterName:  "gemma-journal-custom-lora",
			Message:      "No fine-tuning job running currently",
			UpdatedAt:    time.Now(),
		},
	}
	if db != nil {
		h.ensureAdminTables(context.Background())
	}
	return h
}

func (h *Handler) ensureAdminTables(ctx context.Context) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admin_users (
			id              BIGSERIAL PRIMARY KEY,
			user_id         BIGINT NOT NULL UNIQUE,
			admin_level     VARCHAR(50) NOT NULL DEFAULT 'superadmin',
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS llm_configs (
			id              BIGSERIAL PRIMARY KEY,
			system_prompt   TEXT NOT NULL DEFAULT '',
			max_tokens      INT NOT NULL DEFAULT 2048,
			temperature     DOUBLE PRECISION NOT NULL DEFAULT 0.2,
			top_p           DOUBLE PRECISION NOT NULL DEFAULT 0.9,
			active_adapter  VARCHAR(255) DEFAULT 'gemma-default-lora',
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS peft_adapters (
			id              BIGSERIAL PRIMARY KEY,
			name            VARCHAR(255) NOT NULL UNIQUE,
			description     TEXT DEFAULT '',
			file_path       VARCHAR(512) NOT NULL,
			is_active       BOOLEAN NOT NULL DEFAULT false,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS finetune_jobs (
			id              BIGSERIAL PRIMARY KEY,
			adapter_name    VARCHAR(255) NOT NULL,
			status          VARCHAR(50) NOT NULL DEFAULT 'RUNNING',
			current_epoch   INT NOT NULL DEFAULT 0,
			total_epochs    INT NOT NULL DEFAULT 5,
			progress_pct    DOUBLE PRECISION NOT NULL DEFAULT 0.0,
			loss            DOUBLE PRECISION DEFAULT 0.0,
			vram_gb         DOUBLE PRECISION DEFAULT 0.0,
			message         TEXT DEFAULT '',
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`INSERT INTO llm_configs (system_prompt, max_tokens, temperature, top_p, active_adapter)
		 SELECT 'You are an expert AI prompt optimizer and cognitive load analyst.', 2048, 0.2, 0.9, 'gemma-default-lora'
		 WHERE NOT EXISTS (SELECT 1 FROM llm_configs);`,
	}
	for _, q := range queries {
		_, _ = h.db.Exec(ctx, q)
	}
}

type LlmConfigDTO struct {
	SystemPrompt  string  `json:"system_prompt"`
	MaxTokens     int     `json:"max_tokens"`
	Temperature   float64 `json:"temperature"`
	TopP          float64 `json:"top_p"`
	ActiveAdapter string  `json:"active_adapter"`
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := LlmConfigDTO{
		SystemPrompt:  "You are an expert AI prompt optimizer and cognitive load analyst.",
		MaxTokens:     2048,
		Temperature:   0.2,
		TopP:          0.9,
		ActiveAdapter: "gemma-default-lora",
	}

	if h.db != nil {
		_ = h.db.QueryRow(r.Context(),
			`SELECT system_prompt, max_tokens, temperature, top_p, active_adapter FROM llm_configs ORDER BY id DESC LIMIT 1`).
			Scan(&cfg.SystemPrompt, &cfg.MaxTokens, &cfg.Temperature, &cfg.TopP, &cfg.ActiveAdapter)
	}

	response.JSON(w, http.StatusOK, cfg)
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var input LlmConfigDTO
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid configuration input"})
		return
	}

	if h.db != nil {
		_, _ = h.db.Exec(r.Context(),
			`INSERT INTO llm_configs (system_prompt, max_tokens, temperature, top_p, active_adapter, updated_at)
			 VALUES ($1, $2, $3, $4, $5, NOW())`,
			input.SystemPrompt, input.MaxTokens, input.Temperature, input.TopP, input.ActiveAdapter)
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "LLM Configuration updated successfully",
		"config":  input,
	})
}

type AdapterDTO struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handler) ListAdapters(w http.ResponseWriter, r *http.Request) {
	adapters := []AdapterDTO{
		{ID: 1, Name: "gemma-default-lora", Description: "Default fine-tuned LoRA adapter for general prompt optimization", FilePath: "/adapters/gemma-default-lora.safetensors", IsActive: true, CreatedAt: time.Now()},
		{ID: 2, Name: "code-optimizer-v2", Description: "LoRA adapter fine-tuned on 50k software engineering specs", FilePath: "/adapters/code-optimizer-v2.safetensors", IsActive: false, CreatedAt: time.Now()},
		{ID: 3, Name: "gemma-journal-custom-lora", Description: "Custom PEFT adapter trained on user emotion & focus journals", FilePath: "/adapters/gemma-journal-custom-lora.safetensors", IsActive: false, CreatedAt: time.Now()},
	}

	if h.db != nil {
		rows, err := h.db.Query(r.Context(), `SELECT id, name, description, file_path, is_active, created_at FROM peft_adapters ORDER BY id ASC`)
		if err == nil {
			defer rows.Close()
			dbAdapters := []AdapterDTO{}
			for rows.Next() {
				var a AdapterDTO
				if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.FilePath, &a.IsActive, &a.CreatedAt); err == nil {
					dbAdapters = append(dbAdapters, a)
				}
			}
			if len(dbAdapters) > 0 {
				adapters = dbAdapters
			}
		}
	}

	response.JSON(w, http.StatusOK, adapters)
}

func (h *Handler) ActivateAdapter(w http.ResponseWriter, r *http.Request) {
	type ActivateInput struct {
		Name string `json:"name"`
	}
	var input ActivateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Name == "" {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "Adapter name required"})
		return
	}

	if h.db != nil {
		_, _ = h.db.Exec(r.Context(), `UPDATE peft_adapters SET is_active = false`)
		_, _ = h.db.Exec(r.Context(), `UPDATE peft_adapters SET is_active = true WHERE name = $1`, input.Name)
		_, _ = h.db.Exec(r.Context(), `UPDATE llm_configs SET active_adapter = $1 WHERE id = (SELECT max(id) FROM llm_configs)`, input.Name)
	}

	h.telemetryMu.Lock()
	h.latestTelemetry.AdapterName = input.Name
	h.telemetryMu.Unlock()

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":        "PEFT LoRA Adapter activated hot-swap successfully",
		"active_adapter": input.Name,
	})
}

// ReceiveTelemetry receives telemetry logs from PEFT python training script and persists to DB.
func (h *Handler) ReceiveTelemetry(w http.ResponseWriter, r *http.Request) {
	var input FineTuneTelemetryDTO
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid telemetry payload"})
		return
	}
	input.UpdatedAt = time.Now()

	h.telemetryMu.Lock()
	h.latestTelemetry = input
	h.telemetryMu.Unlock()

	if h.db != nil {
		_, _ = h.db.Exec(r.Context(),
			`INSERT INTO finetune_jobs (adapter_name, status, current_epoch, total_epochs, progress_pct, loss, vram_gb, message, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
			input.AdapterName, input.Status, input.CurrentEpoch, input.TotalEpochs, input.ProgressPct, input.Loss, input.VramGB, input.Message)

		// Auto-register adapter if completed
		if input.Status == "COMPLETED" {
			filePath := fmt.Sprintf("/adapters/%s.safetensors", input.AdapterName)
			desc := fmt.Sprintf("PEFT LoRA adapter fine-tuned locally (Final Loss: %.4f)", input.Loss)
			_, _ = h.db.Exec(r.Context(),
				`INSERT INTO peft_adapters (name, description, file_path, is_active, created_at)
				 VALUES ($1, $2, $3, false, NOW())
				 ON CONFLICT (name) DO UPDATE SET description = $2, file_path = $3`,
				input.AdapterName, desc, filePath)
		}
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "telemetry_recorded"})
}

// GetFineTuneStatus returns the current fine-tuning progress and loss metrics.
func (h *Handler) GetFineTuneStatus(w http.ResponseWriter, r *http.Request) {
	h.telemetryMu.RLock()
	defer h.telemetryMu.RUnlock()

	response.JSON(w, http.StatusOK, h.latestTelemetry)
}

// GetFineTuneHistory fetches historical fine-tuning jobs recorded in PostgreSQL.
func (h *Handler) GetFineTuneHistory(w http.ResponseWriter, r *http.Request) {
	history := []FineTuneTelemetryDTO{
		{
			ID:           1,
			AdapterName:  "gemma-journal-custom-lora",
			Status:       "COMPLETED",
			CurrentEpoch: 5,
			TotalEpochs:  5,
			ProgressPct:  100.0,
			Loss:         0.3412,
			VramGB:       4.1,
			Message:      "Training completed in 42s. Adapter ready for hot-swap.",
			UpdatedAt:    time.Now().Add(-10 * time.Minute),
		},
	}

	if h.db != nil {
		rows, err := h.db.Query(r.Context(),
			`SELECT id, adapter_name, status, current_epoch, total_epochs, progress_pct, loss, vram_gb, message, updated_at
			 FROM finetune_jobs ORDER BY id DESC LIMIT 20`)
		if err == nil {
			defer rows.Close()
			dbHistory := []FineTuneTelemetryDTO{}
			for rows.Next() {
				var item FineTuneTelemetryDTO
				if err := rows.Scan(&item.ID, &item.AdapterName, &item.Status, &item.CurrentEpoch, &item.TotalEpochs, &item.ProgressPct, &item.Loss, &item.VramGB, &item.Message, &item.UpdatedAt); err == nil {
					dbHistory = append(dbHistory, item)
				}
			}
			if len(dbHistory) > 0 {
				history = dbHistory
			}
		}
	}

	response.JSON(w, http.StatusOK, history)
}

// StartFineTune initializes a training run and saves to DB.
func (h *Handler) StartFineTune(w http.ResponseWriter, r *http.Request) {
	type StartInput struct {
		AdapterName  string  `json:"adapter_name"`
		Epochs       int     `json:"epochs"`
		LoraRank     int     `json:"lora_r"`
		LearningRate float64 `json:"learning_rate"`
	}
	var input StartInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.AdapterName == "" {
		input.AdapterName = "gemma-journal-custom-lora"
		input.Epochs = 5
		input.LoraRank = 16
		input.LearningRate = 0.0002
	}

	job := FineTuneTelemetryDTO{
		Status:       "RUNNING",
		CurrentEpoch: 1,
		TotalEpochs:  input.Epochs,
		Step:         1,
		TotalSteps:   input.Epochs * 10,
		ProgressPct:  10.0,
		Loss:         2.45,
		LearningRate: input.LearningRate,
		VramGB:       4.1,
		AdapterName:  input.AdapterName,
		Message:      "Fine-tuning job started on local GPU workstation...",
		UpdatedAt:    time.Now(),
	}

	h.telemetryMu.Lock()
	h.latestTelemetry = job
	h.telemetryMu.Unlock()

	if h.db != nil {
		var jobID int64
		_ = h.db.QueryRow(r.Context(),
			`INSERT INTO finetune_jobs (adapter_name, status, current_epoch, total_epochs, progress_pct, loss, vram_gb, message, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) RETURNING id`,
			job.AdapterName, job.Status, job.CurrentEpoch, job.TotalEpochs, job.ProgressPct, job.Loss, job.VramGB, job.Message).Scan(&jobID)
		job.ID = jobID
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Fine-tuning job initialized successfully",
		"job":     job,
	})
}

// QueryDeepWiki handles legacy single DeepWiki queries.
func (h *Handler) QueryDeepWiki(w http.ResponseWriter, r *http.Request) {
	type QueryInput struct {
		Query string `json:"query"`
	}
	var input QueryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Query == "" {
		input.Query = "React component best practices"
	}

	res, err := h.wikiMCP.SearchKnowledge(r.Context(), mcp.MCPPayload{Query: input.Query})
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]string{"error": "DeepWiki MCP query failed"})
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// ExecuteMCPSuite handles requests across Render MCP, Vercel MCP, MF Academy MCP, and DeepWiki MCP.
func (h *Handler) ExecuteMCPSuite(w http.ResponseWriter, r *http.Request) {
	var req mcp.GenericMCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid MCP Suite request"})
		return
	}

	res, err := h.mcpSuite.ExecuteTool(r.Context(), req)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response.JSON(w, http.StatusOK, res)
}
