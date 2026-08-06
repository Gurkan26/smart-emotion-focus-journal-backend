package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/gurkanfikretgunak/masterfabric-go/internal/infrastructure/llm"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/infrastructure/mcp"
)

// ---------------------------------------------------------------------------
// Agent Tools — Wrappers around existing backend components
// ---------------------------------------------------------------------------
// These tools allow the Agent Harness to use existing infrastructure:
//   - OptimizePromptTool: Prompt optimization engine
//   - QueryKnowledgeTool: DeepWiki MCP for knowledge retrieval
// ---------------------------------------------------------------------------

// === OptimizePromptTool ===

// OptimizePromptTool wraps the Prompt Optimization engine.
type OptimizePromptTool struct {
	analyzer *llm.Analyzer
}

func NewOptimizePromptTool(analyzer *llm.Analyzer) *OptimizePromptTool {
	return &OptimizePromptTool{analyzer: analyzer}
}

func (t *OptimizePromptTool) Name() string { return "optimize_prompt" }
func (t *OptimizePromptTool) Description() string {
	return "Optimizes a raw prompt for an AI model using a specified template. Input: prompt=<text>, template=<accurate|minimal|creative|code|academic>"
}

func (t *OptimizePromptTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	prompt, _ := args["prompt"].(string)
	template, _ := args["template"].(string)
	if strings.TrimSpace(prompt) == "" {
		return &ToolResult{Success: false, Error: "Missing 'prompt' argument"}, nil
	}
	if template == "" {
		template = "accurate"
	}

	result, err := t.analyzer.OptimizePrompt(ctx, prompt, template, "")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	data := result.OptimizedPrompt
	return &ToolResult{Success: true, Data: data}, nil
}

// === QueryKnowledgeTool ===

// QueryKnowledgeTool wraps DeepWiki MCP for knowledge retrieval.
type QueryKnowledgeTool struct {
	mcpSuite *mcp.MCPServerSuite
}

func NewQueryKnowledgeTool(mcpSuite *mcp.MCPServerSuite) *QueryKnowledgeTool {
	return &QueryKnowledgeTool{mcpSuite: mcpSuite}
}

func (t *QueryKnowledgeTool) Name() string { return "query_knowledge" }
func (t *QueryKnowledgeTool) Description() string {
	return "Searches the DeepWiki knowledge base for relevant information. Input: query=<search text>"
}

func (t *QueryKnowledgeTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	query, _ := args["query"].(string)
	if strings.TrimSpace(query) == "" {
		return &ToolResult{Success: false, Error: "Missing 'query' argument"}, nil
	}

	req := mcp.GenericMCPRequest{
		Server: "deepwiki",
		Action: "knowledge_query",
		Query:  query,
	}

	resp, err := t.mcpSuite.ExecuteTool(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	data := fmt.Sprintf("Knowledge result (latency: %dms): %s", resp.LatencyMs, resp.Message)
	return &ToolResult{Success: resp.Success, Data: data}, nil
}
