// Package llm provides LLM integration for Koji's personality decisions.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with an LLM backend.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// Config holds LLM client configuration.
type Config struct {
	BaseURL string        // Ollama API URL (default: http://localhost:11434)
	Model   string        // Model name (default: phi3:mini)
	Timeout time.Duration // Request timeout (default: 30s)
}

// DefaultConfig returns sensible defaults for local Ollama.
func DefaultConfig() Config {
	return Config{
		BaseURL: "http://localhost:11434",
		Model:   "phi3:mini",
		Timeout: 30 * time.Second,
	}
}

// NewClient creates a new LLM client.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "phi3:mini"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// ollamaRequest is the request format for Ollama's /api/generate endpoint.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"` // "json" for JSON output
}

// ollamaResponse is the response format from Ollama's /api/generate endpoint.
type ollamaResponse struct {
	Model      string `json:"model"`
	Response   string `json:"response"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason,omitempty"`
}

// Generate sends a prompt to the LLM and returns the response.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	return c.generate(ctx, prompt, false)
}

// GenerateJSON sends a prompt and requests JSON-formatted output.
func (c *Client) GenerateJSON(ctx context.Context, prompt string) (string, error) {
	return c.generate(ctx, prompt, true)
}

func (c *Client) generate(ctx context.Context, prompt string, jsonFormat bool) (string, error) {
	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}
	if jsonFormat {
		reqBody.Format = "json"
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return result.Response, nil
}

// tagsResponse is the response format from Ollama's /api/tags endpoint.
type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// Ping checks if the LLM backend is available.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// CheckModel verifies the configured model is available and returns available models if not.
func (c *Client) CheckModel(ctx context.Context) (bool, []string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false, nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("connecting to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var tags tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return false, nil, fmt.Errorf("decoding response: %w", err)
	}

	var available []string
	found := false
	for _, m := range tags.Models {
		available = append(available, m.Name)
		if m.Name == c.model {
			found = true
		}
	}

	return found, available, nil
}

// Model returns the configured model name.
func (c *Client) Model() string {
	return c.model
}
