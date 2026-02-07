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
	apperrors "ai-repo-insights/internal/errors"
	"ai-repo-insights/internal/models"
)

// Client interfaces with external LLM API for natural language analysis
type Client struct {
	config config.LLMConfig
	apiKey string
	client *http.Client
	logger zerolog.Logger
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
	}
}

// GenerateAnalysis calls LLM API to generate report analysis
func (c *Client) GenerateAnalysis(summary models.SummaryJSON, reportLanguage string) (models.LLMOutput, error) {
	prompt := c.buildPrompt(summary, reportLanguage)

	c.logger.Info().Msg("calling LLM API for analysis")

	responseText, err := c.callAPIWithRetry(prompt)
	if err != nil {
		return models.LLMOutput{}, apperrors.NewLLMError("failed to call LLM API after retries", err)
	}

	output, err := c.parseResponse(responseText)
	if err != nil {
		return models.LLMOutput{}, apperrors.NewLLMError("failed to parse LLM response", err)
	}

	c.logger.Info().Msg("LLM analysis completed successfully")
	return output, nil
}

// buildPrompt constructs prompt with role, data, and instructions
func (c *Client) buildPrompt(summary models.SummaryJSON, reportLanguage string) string {
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")

	prompt := fmt.Sprintf(`Role: %s

Task: Analyze the following GitHub repository trending data and provide insights in %s.

Data:
%s

Instructions:
1. Write a brief introduction (2-3 sentences) summarizing the overall trends
2. For each category, provide 1-2 sentences of analytical commentary
3. Comment on dark horse projects (high score)
4. Comment on repeater projects (consecutive appearances)
5. Select 3-5 highlight repositories and provide specific insights for each
6. Maintain a %s tone
7. Do NOT fabricate numbers - only interpret the provided data
8. Output valid JSON in this structure:
{
  "intro": "...",
  "category_notes": {"category_name": "..."},
  "dark_horse_notes": "...",
  "repeaters_notes": "...",
  "highlights": [
    {"repo": "owner/repo", "comment": "...", "tone": "neutral-analytical"}
  ]
}`,
		c.config.RoleDescription,
		reportLanguage,
		string(summaryJSON),
		c.config.OutputTone,
	)

	return prompt
}

// parseResponse parses LLM response into structured format
func (c *Client) parseResponse(responseText string) (models.LLMOutput, error) {
	var output models.LLMOutput

	// Clean response text - remove markdown code blocks if present
	cleanedText := cleanJSONResponse(responseText)

	if err := json.Unmarshal([]byte(cleanedText), &output); err != nil {
		return models.LLMOutput{}, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	// Validate required fields
	if output.Intro == "" {
		return models.LLMOutput{}, fmt.Errorf("missing required field: intro")
	}
	if output.CategoryNotes == nil {
		output.CategoryNotes = make(map[string]string)
	}

	return output, nil
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

// callAPIWithRetry calls API with exponential backoff retry
func (c *Client) callAPIWithRetry(prompt string) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			c.logger.Warn().
				Int("attempt", attempt).
				Float64("backoff_seconds", backoffDuration.Seconds()).
				Msg("retrying LLM API call")
			time.Sleep(backoffDuration)
		}

		response, err := c.callAPI(prompt)
		if err == nil {
			return response, nil
		}

		lastErr = err
		c.logger.Warn().
			Int("attempt", attempt).
			Err(err).
			Msg("LLM API call failed")
	}

	return "", fmt.Errorf("all retry attempts exhausted: %w", lastErr)
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
