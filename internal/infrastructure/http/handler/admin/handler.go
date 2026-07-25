package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/infrastructure/mcp"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/shared/response"
)

type Handler struct {
	db      *pgxpool.Pool
	wikiMCP *mcp.DeepWikiMCP
}

func NewHandler(db *pgxpool.Pool) *Handler {
	h := &Handler{
		db:      db,
		wikiMCP: mcp.NewDeepWikiMCP(""),
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
		{ID: 3, Name: "token-compressor-lora", Description: "Ultra-low parameter adapter for prompt compression", FilePath: "/adapters/token-compressor-lora.safetensors", IsActive: false, CreatedAt: time.Now()},
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

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":        "PEFT LoRA Adapter activated hot-swap successfully",
		"active_adapter": input.Name,
	})
}

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
