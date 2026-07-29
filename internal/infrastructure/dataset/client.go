package dataset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PromptRow represents an individual prompt dataset record returned by Hugging Face.
type PromptRow struct {
	OriginalPrompt  string `json:"original_prompt"`
	Template        string `json:"template"`
	OptimizedPrompt string `json:"optimized_prompt"`
}

// DatasetRowWrapper represents the wrapper structure around each row in HF API response.
type DatasetRowWrapper struct {
	RowIdx int       `json:"row_idx"`
	Row    PromptRow `json:"row"`
}

// DatasetResponse represents the root JSON structure returned by Hugging Face Datasets Server API.
type DatasetResponse struct {
	Features []struct {
		FeatureIdx int    `json:"feature_idx"`
		Name       string `json:"name"`
		Type       struct {
			Dtype string `json:"dtype"`
		} `json:"type"`
	} `json:"features"`
	Rows []DatasetRowWrapper `json:"rows"`
}

// Client handles REST HTTP communication with Hugging Face Datasets Server API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient initializes a new Hugging Face Datasets Server API client instance.
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    "https://datasets-server.huggingface.co",
		token:      token,
	}
}

// FetchFirstRows fetches the initial dataset rows from Hugging Face Datasets Server REST API.
func (c *Client) FetchFirstRows(ctx context.Context, datasetRepo string) (*DatasetResponse, error) {
	url := fmt.Sprintf("%s/first-rows?dataset=%s&config=default&split=train", c.baseURL, datasetRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	if c.token != "" {
		req.Header.Add("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to HF Datasets Server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HF Datasets Server returned error status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var datasetResp DatasetResponse
	if err := json.Unmarshal(body, &datasetResp); err != nil {
		return nil, fmt.Errorf("failed to decode dataset JSON response: %w", err)
	}

	return &datasetResp, nil
}
