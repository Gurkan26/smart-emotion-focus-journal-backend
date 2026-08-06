package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Agent Harness Engine — ReAct Loop (Reason + Act + Observe + Reflect)
// ---------------------------------------------------------------------------
// This is the core orchestration layer that wraps around the LLM (fine-tuned
// Gemma model). Instead of single-shot inference, the agent iterates through
// a Think → Act → Observe → Reflect cycle until the task is complete or
// bounded limits are reached.
// ---------------------------------------------------------------------------

// AgentTask represents a user goal submitted to the agent.
type AgentTask struct {
	ID        string    `json:"id"`
	UserID    uint      `json:"userId"`
	Goal      string    `json:"goal"`
	Status    string    `json:"status"` // RUNNING, COMPLETED, FAILED
	Steps     []Step    `json:"steps"`
	Result    string    `json:"result,omitempty"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"createdAt"`
	Duration  int64     `json:"durationMs"`
}

// Step records one iteration of the ReAct loop.
type Step struct {
	Index     int                    `json:"index"`
	Phase     string                 `json:"phase"` // THINK, ACT, OBSERVE, REFLECT
	ToolName  string                 `json:"toolName,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    string                 `json:"output"`
	TokensUsed int                   `json:"tokensUsed"`
	LatencyMs int64                  `json:"latencyMs"`
	CreatedAt time.Time              `json:"createdAt"`
}

// ToolResult is what an AgentTool returns after execution.
type ToolResult struct {
	Success bool   `json:"success"`
	Data    string `json:"data"`
	Error   string `json:"error,omitempty"`
}

// AgentTool is the interface that all tools available to the agent must implement.
type AgentTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

// LLMClient abstracts the LLM call so the harness doesn't depend on a specific model.
type LLMClient interface {
	// ChatComplete sends messages and returns the model's text response.
	ChatComplete(ctx context.Context, systemPrompt string, messages []ChatMessage) (string, error)
}

// ChatMessage is a simple role+content pair for LLM conversation.
type ChatMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant", "tool"
	Content string `json:"content"`
}

// HarnessConfig controls the agent's bounded autonomy limits.
type HarnessConfig struct {
	MaxIterations int           // Max ReAct loop cycles (default: 8)
	Timeout       time.Duration // Hard timeout for entire task (default: 120s)
	EnableReflect bool          // Enable self-repair on errors (default: true)
}

// DefaultConfig returns safe, production-ready defaults.
func DefaultConfig() HarnessConfig {
	return HarnessConfig{
		MaxIterations: 8,
		Timeout:       120 * time.Second,
		EnableReflect: true,
	}
}

// Harness is the Agent Harness Engine — the core orchestrator.
type Harness struct {
	mu       sync.RWMutex
	tools    map[string]AgentTool
	llm      LLMClient
	config   HarnessConfig
	history  []AgentTask // Recent completed tasks
	db       *pgxpool.Pool
}

// NewHarness creates a new Agent Harness with the given LLM client and config.
func NewHarness(llm LLMClient, cfg HarnessConfig) *Harness {
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 8
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 120 * time.Second
	}
	return &Harness{
		tools:   make(map[string]AgentTool),
		llm:     llm,
		config:  cfg,
		history: make([]AgentTask, 0, 32),
	}
}

func (h *Harness) SetDB(db *pgxpool.Pool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.db = db
}

// RegisterTool adds a tool to the agent's available toolset.
func (h *Harness) RegisterTool(tool AgentTool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tools[tool.Name()] = tool
	fmt.Printf("[AGENT] Registered tool: %s — %s\n", tool.Name(), tool.Description())
}

// ListTools returns descriptions of all registered tools for the LLM system prompt.
func (h *Harness) ListTools() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("Available tools:\n")
	for name, tool := range h.tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", name, tool.Description()))
	}
	return sb.String()
}

// Execute runs the ReAct loop for a given user goal.
func (h *Harness) Execute(ctx context.Context, userID uint, goal string) *AgentTask {
	taskCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	task := AgentTask{
		ID:        fmt.Sprintf("task_%d_%d", userID, time.Now().UnixNano()),
		UserID:    userID,
		Goal:      goal,
		Status:    "RUNNING",
		Steps:     make([]Step, 0, h.config.MaxIterations*4),
		CreatedAt: time.Now(),
	}

	fmt.Printf("[AGENT] Starting task %s: %q (max %d iterations, timeout %s)\n",
		task.ID, goal, h.config.MaxIterations, h.config.Timeout)

	systemPrompt := h.buildSystemPrompt(taskCtx)
	messages := []ChatMessage{
		{Role: "user", Content: fmt.Sprintf("Goal: %s", goal)},
	}

	for i := 0; i < h.config.MaxIterations; i++ {
		select {
		case <-taskCtx.Done():
			task.Status = "FAILED"
			task.Result = "Task timed out"
			task.Duration = time.Since(task.CreatedAt).Milliseconds()
			h.recordTask(task)
			return &task
		default:
		}

		// === THINK Phase ===
		thinkStart := time.Now()
		thought, err := h.llm.ChatComplete(taskCtx, systemPrompt, messages)
		if err != nil {
			task.Steps = append(task.Steps, Step{
				Index: len(task.Steps), Phase: "THINK",
				Output: fmt.Sprintf("LLM error: %v", err),
				LatencyMs: time.Since(thinkStart).Milliseconds(),
				CreatedAt: time.Now(),
			})
			task.Status = "FAILED"
			task.Result = fmt.Sprintf("LLM call failed: %v", err)
			break
		}

		task.Steps = append(task.Steps, Step{
			Index: len(task.Steps), Phase: "THINK",
			Output:    thought,
			LatencyMs: time.Since(thinkStart).Milliseconds(),
			CreatedAt: time.Now(),
		})
		messages = append(messages, ChatMessage{Role: "assistant", Content: thought})

		fmt.Printf("[AGENT] Step %d THINK: %s\n", i, truncate(thought, 120))

		// Check if the agent decided to finish (no tool call)
		toolName, toolArgs := h.parseToolCall(thought)
		if toolName == "" {
			// Agent produced a final answer
			task.Status = "COMPLETED"
			task.Result = thought
			break
		}

		// === ACT Phase ===
		actStart := time.Now()
		h.mu.RLock()
		toolNameLower := strings.ToLower(toolName)
		var tool AgentTool
		var exists bool
		for name, t := range h.tools {
			if strings.ToLower(name) == toolNameLower {
				tool = t
				exists = true
				break
			}
		}
		h.mu.RUnlock()

		var toolResult *ToolResult
		if !exists {
			toolResult = &ToolResult{
				Success: false,
				Error:   fmt.Sprintf("Unknown tool: %s. Available: %s", toolName, h.toolNames()),
			}
		} else {
			toolResult, err = tool.Execute(taskCtx, toolArgs)
			if err != nil {
				toolResult = &ToolResult{Success: false, Error: err.Error()}
			}
		}

		task.Steps = append(task.Steps, Step{
			Index: len(task.Steps), Phase: "ACT",
			ToolName:  toolName,
			Input:     toolArgs,
			Output:    toolResult.Data,
			LatencyMs: time.Since(actStart).Milliseconds(),
			CreatedAt: time.Now(),
		})

		fmt.Printf("[AGENT] Step %d ACT: tool=%s success=%v\n", i, toolName, toolResult.Success)

		// === OBSERVE Phase ===
		var observation string
		if toolResult.Success {
			observation = fmt.Sprintf("Tool '%s' returned: %s", toolName, truncate(toolResult.Data, 500))
		} else {
			observation = fmt.Sprintf("Tool '%s' FAILED: %s", toolName, toolResult.Error)
		}

		task.Steps = append(task.Steps, Step{
			Index: len(task.Steps), Phase: "OBSERVE",
			Output:    observation,
			CreatedAt: time.Now(),
		})
		messages = append(messages, ChatMessage{Role: "tool", Content: observation})

		// If prompt optimization tool succeeded, complete task immediately with clean output
		if toolResult.Success && (strings.ToLower(toolName) == "optimize_prompt" || strings.TrimSpace(toolResult.Data) != "") {
			task.Status = "COMPLETED"
			task.Result = toolResult.Data
			break
		}

		// === REFLECT Phase (self-repair on errors) ===
		if !toolResult.Success && h.config.EnableReflect {
			reflectMsg := fmt.Sprintf(
				"The previous tool call failed. Error: %s. Please try a different approach or correct the parameters.",
				toolResult.Error,
			)
			task.Steps = append(task.Steps, Step{
				Index: len(task.Steps), Phase: "REFLECT",
				Output:    reflectMsg,
				CreatedAt: time.Now(),
			})
			messages = append(messages, ChatMessage{Role: "user", Content: reflectMsg})
			fmt.Printf("[AGENT] Step %d REFLECT: self-repair triggered\n", i)
		}
	}

	if task.Status == "RUNNING" {
		task.Status = "COMPLETED"
		task.Result = "Max iterations reached. Last output used as result."
		if len(task.Steps) > 0 {
			task.Result = task.Steps[len(task.Steps)-1].Output
		}
	}

	task.Duration = time.Since(task.CreatedAt).Milliseconds()
	task.Score = h.evaluateTrajectory(task)
	h.recordTask(task)

	fmt.Printf("[AGENT] Task %s completed: status=%s steps=%d duration=%dms score=%.2f\n",
		task.ID, task.Status, len(task.Steps), task.Duration, task.Score)

	return &task
}

// ExecuteDirect runs a deterministic agent task — directly calls a specific tool
// without the LLM THINK step. This is used for well-known tasks (like prompt
// optimization) where we already know which tool to call and with what args.
// The Harness still logs all [AGENT] steps and records history.
func (h *Harness) ExecuteDirect(ctx context.Context, userID uint, goal string, toolName string, toolArgs map[string]interface{}) *AgentTask {
	taskCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	task := AgentTask{
		ID:        fmt.Sprintf("task_%d_%d", userID, time.Now().UnixNano()),
		UserID:    userID,
		Goal:      goal,
		Status:    "RUNNING",
		Steps:     make([]Step, 0, 4),
		CreatedAt: time.Now(),
	}

	fmt.Printf("[AGENT] Starting direct task %s: tool=%s goal=%q\n",
		task.ID, toolName, truncate(goal, 100))

	// === THINK Phase (deterministic — no LLM call) ===
	thinkOutput := fmt.Sprintf("Deterministic routing: directly calling tool '%s' with args: prompt=[user input], template=%v", toolName, toolArgs["template"])
	task.Steps = append(task.Steps, Step{
		Index: 0, Phase: "THINK",
		Output:    thinkOutput,
		LatencyMs: 0,
		CreatedAt: time.Now(),
	})
	fmt.Printf("[AGENT] Step 0 THINK: %s\n", truncate(thinkOutput, 120))

	// === ACT Phase (direct tool execution) ===
	actStart := time.Now()
	h.mu.RLock()
	toolNameLower := strings.ToLower(toolName)
	var tool AgentTool
	var exists bool
	for name, t := range h.tools {
		if strings.ToLower(name) == toolNameLower {
			tool = t
			exists = true
			break
		}
	}
	h.mu.RUnlock()

	var toolResult *ToolResult
	if !exists {
		toolResult = &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Unknown tool: %s. Available: %s", toolName, h.toolNames()),
		}
	} else {
		var err error
		toolResult, err = tool.Execute(taskCtx, toolArgs)
		if err != nil {
			toolResult = &ToolResult{Success: false, Error: err.Error()}
		}
	}

	task.Steps = append(task.Steps, Step{
		Index: 1, Phase: "ACT",
		ToolName:  toolName,
		Input:     toolArgs,
		Output:    truncate(toolResult.Data, 500),
		LatencyMs: time.Since(actStart).Milliseconds(),
		CreatedAt: time.Now(),
	})
	fmt.Printf("[AGENT] Step 1 ACT: tool=%s success=%v latency=%dms\n", toolName, toolResult.Success, time.Since(actStart).Milliseconds())

	// === OBSERVE Phase ===
	var observation string
	if toolResult.Success {
		observation = fmt.Sprintf("Tool '%s' returned optimized result (%d chars)", toolName, len(toolResult.Data))
		task.Status = "COMPLETED"
		task.Result = toolResult.Data
	} else {
		observation = fmt.Sprintf("Tool '%s' FAILED: %s", toolName, toolResult.Error)
		task.Status = "FAILED"
		task.Result = toolResult.Error
	}

	task.Steps = append(task.Steps, Step{
		Index: 2, Phase: "OBSERVE",
		Output:    observation,
		CreatedAt: time.Now(),
	})
	fmt.Printf("[AGENT] Step 2 OBSERVE: %s\n", truncate(observation, 120))

	task.Duration = time.Since(task.CreatedAt).Milliseconds()
	task.Score = h.evaluateTrajectory(task)
	h.recordTask(task)

	fmt.Printf("[AGENT] Task %s completed: status=%s steps=%d duration=%dms score=%.2f\n",
		task.ID, task.Status, len(task.Steps), task.Duration, task.Score)

	return &task
}

// GetHistory returns recent completed tasks.
func (h *Harness) GetHistory() []AgentTask {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]AgentTask, len(h.history))
	copy(result, h.history)
	return result
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (h *Harness) buildSystemPrompt(ctx context.Context) string {
	basePrompt := "You are MasterFabric AI, an expert prompt engineering specialist and cognitive load analyst."
	if h.db != nil && ctx != nil {
		var dbPrompt string
		err := h.db.QueryRow(ctx, `SELECT system_prompt FROM llm_configs ORDER BY id DESC LIMIT 1`).Scan(&dbPrompt)
		if err == nil && strings.TrimSpace(dbPrompt) != "" {
			basePrompt = strings.TrimSpace(dbPrompt)
		}
	}

	return fmt.Sprintf(`%s

%s

## Mandatory Role & Directive
- You are a PROMPT OPTIMIZATION & ENGINEERING AGENT.
- Your MANDATE is to REWRITE, EXPAND, and OPTIMIZE raw user input prompts into comprehensive, structured, professional prompt specifications.
- DO NOT answer or execute the user's raw prompt directly! DO NOT output code or solutions! Instead, transform their input into a superior prompt specification for another LLM.

## How to use tools
When you need to use a tool, respond EXACTLY in this format:
TOOL_CALL: tool_name
ARG: key1=value1
ARG: key2=value2

## When you have the final answer
When you have enough information to answer the user's goal, respond normally WITHOUT any TOOL_CALL. Your response MUST be the final optimized prompt text.

## Rules
- Think step by step before acting.
- Use tools only when necessary.
- If a tool fails, try a different approach.
- Always provide a final answer in the user's language.`, basePrompt, h.ListTools())
}

// parseToolCall extracts tool name and args from the LLM response.
// Robustly handles markdown symbols (##, **), whitespace, and case differences.
func (h *Harness) parseToolCall(thought string) (string, map[string]interface{}) {
	lines := strings.Split(thought, "\n")
	var toolName string
	args := make(map[string]interface{})

	for _, line := range lines {
		cleaned := strings.TrimSpace(line)
		cleaned = strings.TrimLeft(cleaned, "#* \t")

		upper := strings.ToUpper(cleaned)
		if strings.HasPrefix(upper, "TOOL_CALL:") {
			toolName = strings.TrimSpace(cleaned[len("TOOL_CALL:"):])
			toolName = strings.ToLower(toolName)
		} else if strings.HasPrefix(upper, "ARG:") {
			argStr := strings.TrimSpace(cleaned[len("ARG:"):])
			parts := strings.SplitN(argStr, "=", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, "\"'")
				args[key] = val
			}
		}
	}

	return toolName, args
}

func (h *Harness) toolNames() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.tools))
	for name := range h.tools {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// evaluateTrajectory scores the agent's execution path (0.0 - 1.0).
func (h *Harness) evaluateTrajectory(task AgentTask) float64 {
	score := 0.5

	// Completed tasks get a bonus
	if task.Status == "COMPLETED" {
		score += 0.2
	}

	// Fewer steps = more efficient
	stepCount := len(task.Steps)
	if stepCount <= 4 {
		score += 0.15
	} else if stepCount <= 8 {
		score += 0.05
	} else {
		score -= 0.1 // Too many steps = inefficient
	}

	// Count self-repairs
	reflectCount := 0
	for _, s := range task.Steps {
		if s.Phase == "REFLECT" {
			reflectCount++
		}
	}
	if reflectCount == 0 {
		score += 0.1 // No errors needed
	} else if reflectCount >= 3 {
		score -= 0.15 // Too many failures
	}

	// Fast execution bonus
	if task.Duration < 5000 {
		score += 0.05
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}
	return score
}

func (h *Harness) recordTask(task AgentTask) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history = append(h.history, task)
	if len(h.history) > 50 {
		h.history = h.history[len(h.history)-50:]
	}
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
