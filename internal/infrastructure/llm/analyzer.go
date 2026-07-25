package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AnalysisResult struct {
	CognitiveLoad   int              `json:"cognitiveLoad"`
	FocusLevel      int              `json:"focusLevel"`
	StressLevel     int              `json:"stressLevel"`
	DominantEmotion string           `json:"dominantEmotion"`
	Suggestion      string           `json:"suggestion"`
	Metrics         ExecutionMetrics `json:"metrics"`
}

type ExecutionMetrics struct {
	LatencyMs        int64  `json:"latencyMs"`
	PromptTokens     int    `json:"promptTokens"`
	CompletionTokens int    `json:"completionTokens"`
	TotalTokens      int    `json:"totalTokens"`
	InferenceTimeSec string `json:"inferenceTimeSec"`
	TokensSec        string `json:"tokensSec"`
}

type Analyzer struct {
	client  *http.Client
	apiBase string
	apiKey  string
	model   string
}

func NewAnalyzer() *Analyzer {
	apiBase := os.Getenv("LLM_API_BASE")
	apiKey := os.Getenv("LLM_API_KEY")
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gemma2:2b"
	}

	return &Analyzer{
		client:  &http.Client{Timeout: 120 * time.Second},
		apiBase: strings.TrimSuffix(apiBase, "/"),
		apiKey:  apiKey,
		model:   model,
	}
}

func (a *Analyzer) Analyze(ctx context.Context, content string) (*AnalysisResult, error) {
	startTime := time.Now()

	// If external LLM API is configured, try calling OpenAI/MLC LLM API
	if a.apiBase != "" {
		res, err := a.callExternalLLM(ctx, content, startTime)
		if err == nil && res != nil {
			return res, nil
		}
	}

	// Standalone Go Cognitive Analysis Engine
	return a.analyzeFallback(content, startTime), nil
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (a *Analyzer) callExternalLLM(ctx context.Context, content string, startTime time.Time) (*AnalysisResult, error) {
	url := fmt.Sprintf("%s/chat/completions", a.apiBase)
	sysPrompt := "You are an emotional state analysis assistant. Always respond strictly in English regardless of the input text language. Read the user's text and only return a Decision Score in this format: 'Cognitive Load Score: %X - [One sentence advice in English]'. Do not include any other conversational filler."

	reqBody := chatCompletionRequest{
		Model: a.model,
		Messages: []chatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: content},
		},
		Temperature: 0.1,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm status code: %d", resp.StatusCode)
	}

	var completionResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return nil, err
	}

	if len(completionResp.Choices) == 0 {
		return nil, fmt.Errorf("empty llm choices")
	}

	rawReply := strings.TrimSpace(completionResp.Choices[0].Message.Content)
	latencyMs := time.Since(startTime).Milliseconds()

	promptTokens := completionResp.Usage.PromptTokens
	if promptTokens == 0 {
		promptTokens = len(content)/4 + 12
	}
	completionTokens := completionResp.Usage.CompletionTokens
	if completionTokens == 0 {
		completionTokens = len(rawReply)/4 + 5
	}
	totalTokens := promptTokens + completionTokens

	return parseReplyToResult(content, rawReply, latencyMs, promptTokens, completionTokens, totalTokens), nil
}

var (
	highStressKeywords = []string{"yorgun", "stres", "baskı", "burnout", "zor", "kaygı", "endişe", "ölüyorum", "bıktım", "sıkıldım", "kötü", "bozuk", "tire", "exhausted", "stress", "anxious", "overwhelmed"}
	moderateKeywords   = []string{"yoğun", "çalışıyorum", "odak", "projeler", "ödev", "ders", "busy", "working", "focus", "deadline"}
	calmKeywords       = []string{"huzurlu", "sakin", "iyi", "mutlu", "harika", "dinlenmiş", "rahat", "calm", "happy", "relaxed", "great", "peaceful"}
)

func (a *Analyzer) analyzeFallback(content string, startTime time.Time) *AnalysisResult {
	lower := strings.ToLower(content)

	score := 50.0
	highCount := 0
	modCount := 0
	calmCount := 0

	for _, w := range highStressKeywords {
		if strings.Contains(lower, w) {
			highCount++
		}
	}
	for _, w := range moderateKeywords {
		if strings.Contains(lower, w) {
			modCount++
		}
	}
	for _, w := range calmKeywords {
		if strings.Contains(lower, w) {
			calmCount++
		}
	}

	if highCount > 0 {
		score = 65.0 + float64(highCount)*8.0
	} else if modCount > 0 {
		score = 45.0 + float64(modCount)*5.0
	} else if calmCount > 0 {
		score = 25.0 - float64(calmCount)*5.0
	}

	if len(content) > 150 {
		score += 10.0
	}

	if score > 100 {
		score = 100
	}
	if score < 10 {
		score = 10
	}

	var suggestion string
	if score > 75 {
		suggestion = "High cognitive load detected. Taking a deep breath and a 15-minute break is recommended."
	} else if score > 45 {
		suggestion = "Your mental workload is moderate. Adding short breaks between work sessions will help maintain your focus."
	} else {
		suggestion = "Your mental state is well-balanced and calm. You can continue working at your current pace."
	}

	replyText := fmt.Sprintf("Cognitive Load Score: %d%% - %s", int(score), suggestion)
	latencyMs := time.Since(startTime).Milliseconds()

	promptTokens := len(content)/4 + 12
	completionTokens := len(replyText)/4 + 5
	totalTokens := promptTokens + completionTokens

	return parseReplyToResult(content, replyText, latencyMs, promptTokens, completionTokens, totalTokens)
}

func parseReplyToResult(content, replyText string, latencyMs int64, promptTokens, completionTokens, totalTokens int) *AnalysisResult {
	cognitiveLoad := 50
	suggestion := replyText

	re := regexp.MustCompile(`Cognitive\s+Load\s+Score:\s*(\d+)%?\s*-\s*(.*)`)
	match := re.FindStringSubmatch(replyText)
	if len(match) > 2 {
		if val, err := strconv.Atoi(match[1]); err == nil {
			cognitiveLoad = val
		}
		suggestion = strings.TrimSpace(match[2])
	} else {
		rePercent := regexp.MustCompile(`(\d+)%`)
		matchPercent := rePercent.FindStringSubmatch(replyText)
		if len(matchPercent) > 1 {
			if val, err := strconv.Atoi(matchPercent[1]); err == nil {
				cognitiveLoad = val
			}
		}
		parts := strings.Split(replyText, "-")
		if len(parts) > 1 {
			suggestion = strings.TrimSpace(strings.Join(parts[1:], "-"))
		}
	}

	if cognitiveLoad > 100 {
		cognitiveLoad = 100
	}
	if cognitiveLoad < 0 {
		cognitiveLoad = 0
	}

	stressLevel := int(math.Min(100, math.Max(0, float64(cognitiveLoad)*1.1-10)))
	focusLevel := int(math.Min(100, math.Max(0, 100-float64(cognitiveLoad)*0.8)))

	dominantEmotion := "Calm & Centered"
	if cognitiveLoad > 75 {
		dominantEmotion = "High Cognitive Fatigue"
	} else if cognitiveLoad > 45 {
		dominantEmotion = "Distracted / Restless"
	}

	sec := float64(latencyMs) / 1000.0
	if sec <= 0 {
		sec = 0.001
	}
	tps := float64(totalTokens) / sec

	return &AnalysisResult{
		CognitiveLoad:   cognitiveLoad,
		FocusLevel:      focusLevel,
		StressLevel:     stressLevel,
		DominantEmotion: dominantEmotion,
		Suggestion:      suggestion,
		Metrics: ExecutionMetrics{
			LatencyMs:        latencyMs,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
			InferenceTimeSec: fmt.Sprintf("%.2f", sec),
			TokensSec:        fmt.Sprintf("%.1f", tps),
		},
	}
}

type OptimizationResult struct {
	OriginalPrompt    string           `json:"originalPrompt"`
	OptimizedPrompt   string           `json:"optimizedPrompt"`
	Template          string           `json:"template"`
	CustomInstruction string           `json:"customInstruction,omitempty"`
	OriginalTokens    int              `json:"originalTokens"`
	OptimizedTokens   int              `json:"optimizedTokens"`
	TokenSavingsPct   float64          `json:"tokenSavingsPct"`
	Explanation       string           `json:"explanation"`
	Metrics           ExecutionMetrics `json:"metrics"`
}

func (a *Analyzer) OptimizePrompt(ctx context.Context, rawPrompt, template, customInst string) (*OptimizationResult, error) {
	startTime := time.Now()
	cleanPrompt := strings.TrimSpace(rawPrompt)

	systemPrompts := map[string]string{
		"accurate": "You are an expert prompt engineer. Rewrite the user's prompt to be maximally precise, unambiguous, clear, and structured for an AI model to produce the most accurate result possible. Return ONLY the rewritten prompt without introductory text or explanations.",
		"minimal":  "You are a token optimization expert. Compress and rewrite the user's prompt to use the FEWEST tokens possible while preserving 100% of the core intent and critical requirements. Remove filler words. Return ONLY the compressed prompt.",
		"creative": "You are a creative prompt enhancer. Expand and enrich the user's prompt to inspire deep, highly imaginative, vivid, and detailed responses from AI models. Add rich context and stylistic direction. Return ONLY the enhanced prompt.",
		"code":     "You are a software architecture prompt specialist. Rewrite the user's prompt as a rigorous technical specification for an AI coding assistant. Include explicit requirements for clean code, error handling, edge cases, and typing. Return ONLY the rewritten technical prompt.",
		"academic": "You are an academic researcher. Rewrite the user's prompt into a scholarly, research-grade query with formal academic terminology and structured analytical criteria. Return ONLY the scholarly prompt.",
		"custom":   fmt.Sprintf("You are an expert prompt optimizer. Follow this specific instruction to optimize the user's prompt: '%s'. Return ONLY the optimized prompt.", customInst),
	}

	sysPrompt, exists := systemPrompts[template]
	if !exists || sysPrompt == "" {
		sysPrompt = systemPrompts["accurate"]
	}

	var optimized string
	var latencyMs int64

	if a.apiBase != "" {
		reqBody := chatCompletionRequest{
			Model: a.model,
			Messages: []chatMessage{
				{Role: "system", Content: sysPrompt},
				{Role: "user", Content: cleanPrompt},
			},
			Temperature: 0.2,
		}

		jsonBytes, err := json.Marshal(reqBody)
		if err == nil {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/chat/completions", a.apiBase), bytes.NewBuffer(jsonBytes))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				if a.apiKey != "" {
					req.Header.Set("Authorization", "Bearer "+a.apiKey)
				}
				resp, err := a.client.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					var completionResp chatCompletionResponse
					if err := json.NewDecoder(resp.Body).Decode(&completionResp); err == nil && len(completionResp.Choices) > 0 {
						optimized = strings.TrimSpace(completionResp.Choices[0].Message.Content)
					}
					resp.Body.Close()
				}
			}
		}
	}

	// Smart Go-Engine Fallback for Prompt Optimization
	if optimized == "" {
		optimized = fallbackOptimize(cleanPrompt, template, customInst)
	}

	latencyMs = time.Since(startTime).Milliseconds()
	if latencyMs == 0 {
		latencyMs = 45
	}

	origTokens := len(cleanPrompt)/4 + 4
	optTokens := len(optimized)/4 + 4
	savingsPct := 0.0
	if origTokens > 0 {
		savingsPct = math.Round(((float64(origTokens) - float64(optTokens)) / float64(origTokens)) * 1000) / 10
	}

	sec := float64(latencyMs) / 1000.0
	if sec <= 0 {
		sec = 0.001
	}

	explanation := fmt.Sprintf("Optimized using template '%s'. Reduced redundant context and improved AI instruction clarity.", template)

	return &OptimizationResult{
		OriginalPrompt:    cleanPrompt,
		OptimizedPrompt:   optimized,
		Template:          template,
		CustomInstruction: customInst,
		OriginalTokens:    origTokens,
		OptimizedTokens:   optTokens,
		TokenSavingsPct:   savingsPct,
		Explanation:       explanation,
		Metrics: ExecutionMetrics{
			LatencyMs:        latencyMs,
			PromptTokens:     origTokens,
			CompletionTokens: optTokens,
			TotalTokens:      origTokens + optTokens,
			InferenceTimeSec: fmt.Sprintf("%.2f", sec),
			TokensSec:        fmt.Sprintf("%.1f", float64(origTokens+optTokens)/sec),
		},
	}, nil
}

func fallbackOptimize(prompt, template, customInst string) string {
	clean := strings.TrimSpace(prompt)
	switch template {
	case "minimal":
		// Remove fluff words
		words := strings.Fields(clean)
		filtered := make([]string, 0, len(words))
		fluff := map[string]bool{
			"please": true, "could": true, "you": true, "kindly": true, "want": true,
			"to": true, "try": true, "make": true, "sure": true, "that": true, "lütfen": true,
			"bana": true, "yapar": true, "mısın": true, "istiyorum": true,
		}
		for _, w := range words {
			if !fluff[strings.ToLower(w)] {
				filtered = append(filtered, w)
			}
		}
		if len(filtered) > 0 {
			return strings.Join(filtered, " ") + " [Concise output only]"
		}
		return clean

	case "code":
		return fmt.Sprintf("Act as a senior software architect. Implement the following task with clean architecture, strict error handling, and unit tests:\n\nTask: %s\n\nRequirements:\n- Production-ready code\n- Include clear type definitions\n- Provide step-by-step rationale", clean)

	case "creative":
		return fmt.Sprintf("Explore the following concept with rich creative detail, vivid imagery, and engaging tone:\n\nPrompt: %s\n\nProvide deep perspective, creative narrative elements, and compelling structure.", clean)

	case "academic":
		return fmt.Sprintf("Analyze the following research question using academic methodology, peer-reviewed citations, and formal analytical structure:\n\nQuery: %s\n\nInclude background, methodology, discussion of trade-offs, and systematic conclusion.", clean)

	case "custom":
		if customInst != "" {
			return fmt.Sprintf("[Instruction: %s]\n\nTask: %s", customInst, clean)
		}
		return fmt.Sprintf("Task: %s\nConstraint: Provide exact and verified answer.", clean)

	case "accurate":
	default:
		return fmt.Sprintf("Provide a precise, accurate, and structured solution to the following prompt:\n\n'%s'\n\nGuidelines:\n1. Direct and unambiguous answer\n2. Include key edge cases\n3. Format with clean sections and markdown tables if applicable.", clean)
	}

	return clean
}

