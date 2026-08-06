package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// OllamaLLMClient — Adapter connecting the Agent Harness to Ollama API
// ---------------------------------------------------------------------------
// Implements the LLMClient interface so the harness can use the local
// Ollama (Docker) instance with the fine-tuned Gemma GGUF model.
// ---------------------------------------------------------------------------

type OllamaLLMClient struct {
	client  *http.Client
	apiBase string
	model   string
}

// NewOllamaLLMClient creates an LLMClient that talks to the local Ollama instance.
func NewOllamaLLMClient() *OllamaLLMClient {
	apiBase := os.Getenv("LLM_API_BASE")
	if apiBase == "" {
		apiBase = os.Getenv("OLLAMA_API_BASE")
	}
	if apiBase == "" {
		apiBase = "http://localhost:11434/v1"
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gemma-journal:latest"
	}

	return &OllamaLLMClient{
		client:  &http.Client{Timeout: 120 * time.Second},
		apiBase: strings.TrimSuffix(apiBase, "/"),
		model:   model,
	}
}

type ollamaChatReq struct {
	Model       string          `json:"model"`
	Messages    []ollamaMsg     `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type ollamaMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ChatComplete sends a conversation to Ollama and returns the model's response.
func (o *OllamaLLMClient) ChatComplete(ctx context.Context, systemPrompt string, messages []ChatMessage) (string, error) {
	var url string
	if strings.HasSuffix(o.apiBase, "/v1") {
		url = fmt.Sprintf("%s/chat/completions", o.apiBase)
	} else {
		url = fmt.Sprintf("%s/v1/chat/completions", o.apiBase)
	}

	ollamaMsgs := make([]ollamaMsg, 0, len(messages)+1)
	ollamaMsgs = append(ollamaMsgs, ollamaMsg{Role: "system", Content: systemPrompt})
	for _, m := range messages {
		role := m.Role
		if role == "tool" {
			role = "user" // Ollama doesn't have a "tool" role; send as user
		}
		ollamaMsgs = append(ollamaMsgs, ollamaMsg{Role: role, Content: m.Content})
	}

	reqBody := ollamaChatReq{
		Model:       o.model,
		Messages:    ollamaMsgs,
		Temperature: 0.3,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	fmt.Printf("[AGENT-LLM] Calling %s model=%s msgs=%d\n", url, o.model, len(ollamaMsgs))

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, buf.String())
	}

	var chatResp ollamaChatResp
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode error: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from model")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
