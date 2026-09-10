package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ActiveModel represents an Ollama model currently loaded in memory.
type ActiveModel struct {
	Name      string    `json:"name"`
	Model     string    `json:"model"`
	Size      int64     `json:"size"`
	Digest    string    `json:"digest"`
	ExpiresAt time.Time `json:"expires_at"`
	SizeVRAM  int64     `json:"size_vram"`
}

// OllamaClient interfaces with the local Ollama instance.
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOllamaClient creates a client pointing to the Ollama daemon.
func NewOllamaClient(baseURL string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434"
	}
	return &OllamaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Ping checks if the Ollama daemon is reachable.
func (c *OllamaClient) Ping(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status: %d", resp.StatusCode)
	}

	var payload struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "unknown", nil
	}
	return payload.Version, nil
}

// ListRunningModels returns models currently loaded in RAM/VRAM.
func (c *OllamaClient) ListRunningModels(ctx context.Context) ([]ActiveModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/ps", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query running models: %d", resp.StatusCode)
	}

	var payload struct {
		Models []ActiveModel `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Models, nil
}

// UnloadModel frees memory occupied by a loaded model by sending keep_alive: 0.
func (c *OllamaClient) UnloadModel(ctx context.Context, modelID string) error {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":      modelID,
		"keep_alive": 0,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to unload model %s: HTTP %d", modelID, resp.StatusCode)
	}
	return nil
}
