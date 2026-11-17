package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/config"
)

// Client interfaces with external LLM API for natural language analysis
type Client struct {
	config config.LLMConfig
	apiKey string
	client *http.Client
	logger zerolog.Logger
	// backoff is the wait before retry attempt n (n >= 1).
	backoff func(attempt int) time.Duration
}

// NewClient creates a new LLM client
func NewClient(cfg config.LLMConfig, apiKey string, logger zerolog.Logger) *Client {
	return &Client{
		config: cfg,
		apiKey: apiKey,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
		logger: logger,
		backoff: func(attempt int) time.Duration {
			return time.Duration(math.Pow(2, float64(attempt))) * time.Second
		},
	}
}

// cleanJSONResponse removes markdown code blocks and whitespace from response
func cleanJSONResponse(text string) string {
	// Remove leading/trailing whitespace
	text = strings.TrimSpace(text)

	// Remove markdown code blocks (```json ... ``` or ``` ... ```)
	if strings.HasPrefix(text, "```") {
		// Find the first newline after opening ```
		firstNewline := strings.Index(text, "\n")
		if firstNewline != -1 {
			text = text[firstNewline+1:]
		}
		// Remove closing ```
		if strings.HasSuffix(text, "```") {
			text = text[:len(text)-3]
		}
		text = strings.TrimSpace(text)
	}

	return text
}

// callAPI makes a single API call to the LLM service
func (c *Client) callAPI(prompt string) (string, error) {
	if c.config.Provider == "gemini" {
		return c.callGeminiAPI(prompt)
	}
	return c.callOpenAIAPI(prompt)
}

// callOpenAIAPI makes an API call using OpenAI format
func (c *Client) callOpenAIAPI(prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": c.config.Model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": c.config.Temperature,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := c.config.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("API response contains no choices")
	}

	return apiResponse.Choices[0].Message.Content, nil
}

// callGeminiAPI makes an API call using Gemini format
func (c *Client) callGeminiAPI(prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": c.config.Temperature,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.config.BaseURL, c.config.Model, c.apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	if len(apiResponse.Candidates) == 0 {
		return "", fmt.Errorf("API response contains no candidates")
	}

	if len(apiResponse.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("API response candidate contains no parts")
	}

	return apiResponse.Candidates[0].Content.Parts[0].Text, nil
}
