package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the askbox server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New creates a new askbox client.
func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// HealthResponse represents the /health response.
type HealthResponse struct {
	Status       string `json:"status"`
	ArchHubReady bool   `json:"arch_hub_ready"`
	ArchHubPath  string `json:"arch_hub_path"`
	ArchHubRepos int    `json:"arch_hub_repos"`
	JobsTotal    int    `json:"jobs_total"`
	JobsRunning  int    `json:"jobs_running"`
	Uptime       float64 `json:"uptime_seconds"`
}

// AskRequest is the body for POST /ask.
type AskRequest struct {
	Question string   `json:"question"`
	Repos    []string `json:"repos,omitempty"`
	Adapter  string   `json:"adapter,omitempty"`
	Model    string   `json:"model,omitempty"`
}

// AskResponse is the response from POST /ask.
type AskResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// JobResponse is the response from GET /ask/{id}.
type JobResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Question  string `json:"question"`
	Answer    string `json:"answer,omitempty"`
	Error     string `json:"error,omitempty"`
	ToolCalls int    `json:"tool_calls"`
	Adapter   string `json:"adapter,omitempty"`
	Model     string `json:"model,omitempty"`
	CreatedAt float64 `json:"created_at,omitempty"`
	Duration  float64 `json:"duration_seconds,omitempty"`
}

// Health checks the askbox server health.
func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("cannot reach askbox at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("askbox returned %d", resp.StatusCode)
	}

	var h HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return nil, fmt.Errorf("decode health: %w", err)
	}
	return &h, nil
}

// Ask submits a question to the askbox.
func (c *Client) Ask(req *AskRequest) (*AskResponse, error) {
	body, _ := json.Marshal(req)
	resp, err := c.HTTPClient.Post(c.BaseURL+"/ask", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot reach askbox at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 503 {
		return nil, fmt.Errorf("arch-hub not loaded — run: ask refresh --url <arch-hub-git-url>")
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("askbox returned %d: %s", resp.StatusCode, string(data))
	}

	var a AskResponse
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &a, nil
}

// GetJob polls a job by ID.
func (c *Client) GetJob(id string) (*JobResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/ask/" + id)
	if err != nil {
		return nil, fmt.Errorf("poll failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("job %s not found", id)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("askbox returned %d", resp.StatusCode)
	}

	var j JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		return nil, fmt.Errorf("decode job: %w", err)
	}
	return &j, nil
}

// ListJobs returns all jobs, optionally filtered by status.
func (c *Client) ListJobs(status string, limit int) ([]JobResponse, error) {
	url := c.BaseURL + "/ask"
	sep := "?"
	if status != "" {
		url += sep + "status=" + status
		sep = "&"
	}
	if limit > 0 {
		url += fmt.Sprintf("%slimit=%d", sep, limit)
	}

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("askbox returned %d", resp.StatusCode)
	}

	var jobs []JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, fmt.Errorf("decode jobs: %w", err)
	}
	return jobs, nil
}

// Refresh triggers an arch-hub refresh.
func (c *Client) Refresh(url, branch string) error {
	reqURL := c.BaseURL + "/arch-hub/refresh"
	sep := "?"
	if url != "" {
		reqURL += sep + "url=" + url
		sep = "&"
	}
	if branch != "" {
		reqURL += sep + "branch=" + branch
	}

	resp, err := c.HTTPClient.Post(reqURL, "", nil)
	if err != nil {
		return fmt.Errorf("refresh failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, string(data))
	}
	return nil
}
