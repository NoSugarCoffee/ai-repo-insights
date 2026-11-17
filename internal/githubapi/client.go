// Package githubapi reads repository details from the GitHub REST API.
// The trending pages carry no creation date or README, and the radar needs
// both: the date to tell new projects apart, the README so the LLM writes
// from what the project does rather than from its name.
package githubapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ReadmeRuneLimit caps the README excerpt passed to the LLM.
const ReadmeRuneLimit = 4000

// RepoDetails is the subset of GET /repos/{owner}/{repo} the radar uses.
type RepoDetails struct {
	CreatedAt   time.Time
	Stars       int
	Topics      []string
	Description string
}

// APIError is a non-success response from the GitHub API.
type APIError struct {
	FullName string
	Endpoint string
	Status   int
	Body     string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("GitHub API %s for %s returned HTTP %d: %s; check that GITHUB_TOKEN is set and valid and that the rate limit is not exhausted",
		e.Endpoint, e.FullName, e.Status, strings.TrimSpace(e.Body))
}

// Client calls the GitHub REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a client; baseURL is https://api.github.com in production.
func NewClient(baseURL string, token string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, httpClient: httpClient}
}

// Repo fetches creation time, stars, topics and description for "owner/repo".
func (c *Client) Repo(ctx context.Context, fullName string) (RepoDetails, error) {
	var body struct {
		CreatedAt       time.Time `json:"created_at"`
		StargazersCount int       `json:"stargazers_count"`
		Topics          []string  `json:"topics"`
		Description     string    `json:"description"`
	}
	if _, err := c.getJSON(ctx, fullName, "/repos/"+fullName, &body); err != nil {
		return RepoDetails{}, err
	}
	return RepoDetails{CreatedAt: body.CreatedAt, Stars: body.StargazersCount, Topics: body.Topics, Description: body.Description}, nil
}

// Readme returns the first ReadmeRuneLimit runes of the repository README,
// or "" when the repository has none.
func (c *Client) Readme(ctx context.Context, fullName string) (string, error) {
	var body struct {
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	status, err := c.getJSON(ctx, fullName, "/repos/"+fullName+"/readme", &body)
	if err != nil {
		return "", err
	}
	// GitHub omits the content of files over 1 MB (encoding "none"); such a
	// README is treated like a missing one rather than failing the run.
	if status == http.StatusNotFound || body.Encoding == "none" {
		return "", nil
	}
	if body.Encoding != "base64" {
		return "", fmt.Errorf("README of %s has unsupported encoding %q", fullName, body.Encoding)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body.Content, "\n", ""))
	if err != nil {
		return "", fmt.Errorf("decode README of %s: %w", fullName, err)
	}
	return truncateRunes(string(decoded), ReadmeRuneLimit), nil
}

// getJSON returns 404 as a status for the caller to interpret; any other
// non-200 response is an *APIError.
func (c *Client) getJSON(ctx context.Context, fullName string, path string, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("GitHub API %s for %s: %w", path, fullName, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read GitHub API %s for %s: %w", path, fullName, err)
	}
	if resp.StatusCode == http.StatusNotFound && strings.HasSuffix(path, "/readme") {
		return resp.StatusCode, nil
	}
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, &APIError{FullName: fullName, Endpoint: path, Status: resp.StatusCode, Body: string(data)}
	}
	if err := json.Unmarshal(data, out); err != nil {
		return resp.StatusCode, fmt.Errorf("parse GitHub API %s for %s: %w", path, fullName, err)
	}
	return resp.StatusCode, nil
}

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}
