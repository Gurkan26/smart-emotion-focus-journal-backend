package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Continuous Learning Collector
// ---------------------------------------------------------------------------
// Collects user interactions (journal analyses, prompt optimizations) and
// converts successful ones into fine-tuning training data. When enough new
// examples accumulate, the dataset is auto-appended to the JSONL file so
// the next fine-tuning run benefits from real user data.
//
// This creates a feedback loop:
//   User interaction → Collector buffer → JSONL dataset → Fine-tune → Better model
// ---------------------------------------------------------------------------

// TrainingExample represents a single instruction/input/output triple
// in the format expected by peft_finetune.py.
type TrainingExample struct {
	Instruction string `json:"instruction"`
	Input       string `json:"input"`
	Output      string `json:"output"`
}

// InteractionRecord captures a raw user interaction before it becomes
// a training example. Includes metadata for quality filtering.
type InteractionRecord struct {
	ID              string    `json:"id"`
	Source          string    `json:"source"`           // "journal_analyze" or "prompt_optimize"
	UserID          uint      `json:"user_id"`
	UserInput       string    `json:"user_input"`       // What the user sent
	ModelOutput     string    `json:"model_output"`     // What the model returned
	ThinkingProcess string    `json:"thinking_process"` // Model's internal reasoning
	Template        string    `json:"template"`         // Template used (for prompt optimize)
	QualityScore    float64   `json:"quality_score"`    // Auto-assessed quality (0.0-1.0)
	UserRating      int       `json:"user_rating"`      // 0=unrated, 1=thumbs_down, 2=thumbs_up
	CreatedAt       time.Time `json:"created_at"`
}

// CollectorStats reports the current state of the learning pipeline.
type CollectorStats struct {
	BufferSize           int       `json:"bufferSize"`
	TotalCollected       int       `json:"totalCollected"`
	TotalFlushed         int       `json:"totalFlushed"`
	ApprovedExamples     int       `json:"approvedExamples"`
	PendingExamples      int       `json:"pendingExamples"`
	AutoFlushThreshold   int       `json:"autoFlushThreshold"`
	DatasetPath          string    `json:"datasetPath"`
	LastFlushAt          time.Time `json:"lastFlushAt,omitempty"`
	ContinuousLearning   bool      `json:"continuousLearning"`
}

// Collector manages the continuous learning data pipeline.
type Collector struct {
	mu sync.RWMutex

	// Buffer of unprocessed interactions
	buffer []InteractionRecord

	// Counters
	totalCollected int
	totalFlushed   int

	// Configuration
	datasetPath        string // Path to journal_finetune_dataset.jsonl
	autoFlushThreshold int    // Auto-flush to JSONL after this many approved examples
	lastFlushAt        time.Time

	// Minimum quality thresholds
	minQualityScore float64 // Only flush examples above this score
}

// NewCollector creates a new continuous learning collector.
// datasetPath should point to the JSONL file used by peft_finetune.py.
// autoFlushThreshold controls how many approved examples trigger an auto-flush.
func NewCollector(datasetPath string, autoFlushThreshold int) *Collector {
	if autoFlushThreshold <= 0 {
		autoFlushThreshold = 25
	}
	if datasetPath == "" {
		datasetPath = filepath.Join("scripts", "journal_finetune_dataset.jsonl")
	}
	return &Collector{
		buffer:             make([]InteractionRecord, 0, 64),
		datasetPath:        datasetPath,
		autoFlushThreshold: autoFlushThreshold,
		minQualityScore:    0.6,
	}
}

// RecordAnalysis captures a journal analysis interaction for potential training.
func (c *Collector) RecordAnalysis(userID uint, content string, cognitiveLoad int, suggestion string) {
	quality := assessAnalysisQuality(content, cognitiveLoad, suggestion)

	record := InteractionRecord{
		ID:           fmt.Sprintf("analyze_%d_%d", userID, time.Now().UnixNano()),
		Source:       "journal_analyze",
		UserID:       userID,
		UserInput:    content,
		ModelOutput:  fmt.Sprintf("Cognitive Load Score: %d%% - %s", cognitiveLoad, suggestion),
		QualityScore: quality,
		UserRating:   0, // Unrated by default
		CreatedAt:    time.Now(),
	}

	c.mu.Lock()
	c.buffer = append(c.buffer, record)
	c.totalCollected++
	c.mu.Unlock()

	fmt.Printf("[LEARNING] Recorded analysis interaction (quality: %.2f, buffer: %d)\n", quality, len(c.buffer))
}

// RecordOptimization captures a prompt optimization interaction for training.
func (c *Collector) RecordOptimization(userID uint, originalPrompt, optimizedPrompt, template, thinking string) {
	quality := assessOptimizationQuality(originalPrompt, optimizedPrompt, template)

	record := InteractionRecord{
		ID:              fmt.Sprintf("optimize_%d_%d", userID, time.Now().UnixNano()),
		Source:          "prompt_optimize",
		UserID:          userID,
		UserInput:       originalPrompt,
		ModelOutput:     optimizedPrompt,
		ThinkingProcess: thinking,
		Template:        template,
		QualityScore:    quality,
		UserRating:      0,
		CreatedAt:       time.Now(),
	}

	c.mu.Lock()
	c.buffer = append(c.buffer, record)
	c.totalCollected++
	c.mu.Unlock()

	fmt.Printf("[LEARNING] Recorded optimization interaction (quality: %.2f, template: %s, buffer: %d)\n", quality, template, len(c.buffer))
}

// RateInteraction allows users to rate a model response (thumbs up/down).
// Positive ratings (2) boost an example's chance of being flushed to the dataset.
func (c *Collector) RateInteraction(interactionID string, rating int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.buffer {
		if c.buffer[i].ID == interactionID {
			c.buffer[i].UserRating = rating
			if rating == 2 { // thumbs up
				c.buffer[i].QualityScore = min(1.0, c.buffer[i].QualityScore+0.3)
			} else if rating == 1 { // thumbs down
				c.buffer[i].QualityScore = max(0.0, c.buffer[i].QualityScore-0.4)
			}
			fmt.Printf("[LEARNING] Interaction %s rated: %d (new quality: %.2f)\n", interactionID, rating, c.buffer[i].QualityScore)

			// Check if we should auto-flush
			approvedCount := 0
			for _, rec := range c.buffer {
				if rec.QualityScore >= c.minQualityScore {
					approvedCount++
				}
			}
			if approvedCount >= c.autoFlushThreshold {
				go c.FlushToDataset()
			}
			return true
		}
	}
	return false
}

// FlushToDataset appends high-quality interactions to the JSONL training file.
// Only examples with quality >= minQualityScore are flushed.
func (c *Collector) FlushToDataset() (int, error) {
	c.mu.Lock()

	// Separate approved and pending examples
	var approved []InteractionRecord
	var remaining []InteractionRecord
	for _, rec := range c.buffer {
		if rec.QualityScore >= c.minQualityScore {
			approved = append(approved, rec)
		} else {
			remaining = append(remaining, rec)
		}
	}

	if len(approved) == 0 {
		c.mu.Unlock()
		return 0, nil
	}

	// Convert to training examples
	examples := make([]TrainingExample, 0, len(approved))
	for _, rec := range approved {
		example := interactionToTrainingExample(rec)
		examples = append(examples, example)
	}

	// Update buffer — keep only unapproved records
	c.buffer = remaining
	c.totalFlushed += len(examples)
	c.lastFlushAt = time.Now()
	c.mu.Unlock()

	// Append to JSONL file
	if err := appendToJSONL(c.datasetPath, examples); err != nil {
		return 0, fmt.Errorf("failed to append to dataset: %w", err)
	}

	fmt.Printf("[LEARNING] ✅ Flushed %d approved examples to %s (total flushed: %d)\n", len(examples), c.datasetPath, c.totalFlushed)
	return len(examples), nil
}

// GetStats returns current learning pipeline statistics.
func (c *Collector) GetStats() CollectorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	approved := 0
	pending := 0
	for _, rec := range c.buffer {
		if rec.QualityScore >= c.minQualityScore {
			approved++
		} else {
			pending++
		}
	}

	return CollectorStats{
		BufferSize:         len(c.buffer),
		TotalCollected:     c.totalCollected,
		TotalFlushed:       c.totalFlushed,
		ApprovedExamples:   approved,
		PendingExamples:    pending,
		AutoFlushThreshold: c.autoFlushThreshold,
		DatasetPath:        c.datasetPath,
		LastFlushAt:        c.lastFlushAt,
		ContinuousLearning: true,
	}
}

// GetPendingInteractions returns buffered interactions for admin review.
func (c *Collector) GetPendingInteractions() []InteractionRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]InteractionRecord, len(c.buffer))
	copy(result, c.buffer)
	return result
}

// ---------------------------------------------------------------------------
// Quality Assessment Functions
// ---------------------------------------------------------------------------

// assessAnalysisQuality evaluates whether a journal analysis interaction
// is worth adding to the training dataset.
func assessAnalysisQuality(content string, cognitiveLoad int, suggestion string) float64 {
	score := 0.5 // Base score

	// Longer content = more nuanced analysis opportunity
	if len(content) > 50 {
		score += 0.1
	}
	if len(content) > 150 {
		score += 0.1
	}

	// Reasonable cognitive load range (not edge cases)
	if cognitiveLoad >= 20 && cognitiveLoad <= 90 {
		score += 0.1
	}

	// Non-trivial suggestion
	if len(suggestion) > 30 {
		score += 0.1
	}

	// Penalty for very short/trivial content
	if len(content) < 10 {
		score -= 0.3
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}
	return score
}

// assessOptimizationQuality evaluates prompt optimization quality.
func assessOptimizationQuality(original, optimized, template string) float64 {
	score := 0.5

	// Optimization should produce meaningfully different output
	if len(optimized) > len(original)*2 {
		score += 0.15 // Good expansion
	}

	// Template-specific quality signals
	if template == "minimal" && len(optimized) < len(original) {
		score += 0.15 // Minimal template should compress
	}

	// Non-trivial input
	if len(original) > 10 {
		score += 0.1
	}

	// Non-trivial output
	if len(optimized) > 30 {
		score += 0.1
	}

	// Penalty for trivially short
	if len(original) < 5 || len(optimized) < 10 {
		score -= 0.3
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}
	return score
}

// ---------------------------------------------------------------------------
// Conversion & Persistence
// ---------------------------------------------------------------------------

// interactionToTrainingExample converts a user interaction into a JSONL
// training example compatible with peft_finetune.py.
func interactionToTrainingExample(rec InteractionRecord) TrainingExample {
	switch rec.Source {
	case "prompt_optimize":
		// Build structured output matching CSV dataset format
		outputObj := map[string]string{
			"original_prompt":  rec.UserInput,
			"optimized_prompt": rec.ModelOutput,
			"template_used":    rec.Template,
		}
		outputJSON, _ := json.Marshal(outputObj)
		return TrainingExample{
			Instruction: "Optimize the given prompt for maximum clarity, structure, and model effectiveness.",
			Input:       rec.UserInput,
			Output:      string(outputJSON),
		}

	case "journal_analyze":
		return TrainingExample{
			Instruction: "Analyze the user's emotional state and provide a cognitive load assessment with actionable advice.",
			Input:       rec.UserInput,
			Output:      rec.ModelOutput,
		}

	default:
		return TrainingExample{
			Instruction: "Process the following input and provide an optimized response.",
			Input:       rec.UserInput,
			Output:      rec.ModelOutput,
		}
	}
}

// appendToJSONL appends training examples to the JSONL dataset file.
func appendToJSONL(path string, examples []TrainingExample) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetEscapeHTML(false)
	for _, ex := range examples {
		if err := encoder.Encode(ex); err != nil {
			return err
		}
	}

	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
