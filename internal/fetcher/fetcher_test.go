package fetcher

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"ai-repo-insights/internal/logging"
	"ai-repo-insights/internal/models"

	"github.com/PuerkitoBio/goquery"
)

// TestNew tests the creation of a new TrendingFetcher
func TestNew(t *testing.T) {
	logger := logging.NewLogger("info")
	languages := []string{"go", "python"}

	fetcher := New(languages, logger)

	if fetcher == nil {
		t.Fatal("Expected non-nil fetcher")
	}

	if len(fetcher.languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(fetcher.languages))
	}

	if fetcher.httpClient == nil {
		t.Error("Expected non-nil HTTP client")
	}
}

// TestDeduplicateRepos tests repository deduplication
func TestDeduplicateRepos(t *testing.T) {
	logger := logging.NewLogger("info")
	fetcher := New([]string{"go"}, logger)

	repos := []models.RepoMetadata{
		{Owner: "owner1", Name: "repo1"},
		{Owner: "owner1", Name: "repo1"}, // duplicate
		{Owner: "owner2", Name: "repo2"},
		{Owner: "owner1", Name: "repo1"}, // duplicate
		{Owner: "owner3", Name: "repo3"},
	}

	unique := fetcher.deduplicateRepos(repos)

	if len(unique) != 3 {
		t.Errorf("Expected 3 unique repos, got %d", len(unique))
	}

	// Verify the unique repos are correct
	keys := make(map[string]bool)
	for _, repo := range unique {
		keys[repo.Key()] = true
	}

	expectedKeys := []string{"owner1/repo1", "owner2/repo2", "owner3/repo3"}
	for _, key := range expectedKeys {
		if !keys[key] {
			t.Errorf("Expected key %s not found in unique repos", key)
		}
	}
}

// TestParseStarCount tests parsing star counts from text
func TestParseStarCount(t *testing.T) {
	logger := logging.NewLogger("info")
	fetcher := New([]string{"go"}, logger)

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"simple number", "123", 123},
		{"with comma", "1,234", 1234},
		{"with spaces", "  456  ", 456},
		{"with comma and spaces", " 12,345 ", 12345},
		{"empty string", "", 0},
		{"non-numeric", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fetcher.parseStarCount(tt.input)
			if result != tt.expected {
				t.Errorf("parseStarCount(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExtractRepoMetadata tests extracting metadata from HTML
func TestExtractRepoMetadata(t *testing.T) {
	logger := logging.NewLogger("info")
	fetcher := New([]string{"go"}, logger)

	// Sample HTML structure similar to GitHub trending
	html := `
	<article class="Box-row">
		<h2 class="h3">
			<a href="/owner/repo">owner / repo</a>
		</h2>
		<p class="col-9">This is a test repository description</p>
		<a class="topic-tag">golang</a>
		<a class="topic-tag">testing</a>
		<span class="d-inline-block float-sm-right">1,234</span>
	</article>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	selection := doc.Find("article.Box-row").First()
	repo, err := fetcher.extractRepoMetadata(selection, "go")

	if err != nil {
		t.Fatalf("extractRepoMetadata failed: %v", err)
	}

	if repo.Owner != "owner" {
		t.Errorf("Expected owner 'owner', got '%s'", repo.Owner)
	}

	if repo.Name != "repo" {
		t.Errorf("Expected name 'repo', got '%s'", repo.Name)
	}

	if repo.URL != "https://github.com/owner/repo" {
		t.Errorf("Expected URL 'https://github.com/owner/repo', got '%s'", repo.URL)
	}

	if repo.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", repo.Language)
	}

	if repo.Description != "This is a test repository description" {
		t.Errorf("Expected description 'This is a test repository description', got '%s'", repo.Description)
	}

	if len(repo.Topics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(repo.Topics))
	}

	if repo.Stars != 1234 {
		t.Errorf("Expected 1234 stars, got %d", repo.Stars)
	}
}

// TestExtractRepoMetadataInvalidHref tests error handling for invalid href
func TestExtractRepoMetadataInvalidHref(t *testing.T) {
	logger := logging.NewLogger("info")
	fetcher := New([]string{"go"}, logger)

	tests := []struct {
		name string
		html string
	}{
		{
			"missing link",
			`<article class="Box-row"><h2 class="h3"></h2></article>`,
		},
		{
			"invalid href format",
			`<article class="Box-row"><h2 class="h3"><a href="/invalid">test</a></h2></article>`,
		},
		{
			"no href attribute",
			`<article class="Box-row"><h2 class="h3"><a>test</a></h2></article>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			selection := doc.Find("article.Box-row").First()
			_, err = fetcher.extractRepoMetadata(selection, "go")

			if err == nil {
				t.Error("Expected error for invalid HTML, got nil")
			}
		})
	}
}

// TestSaveRaw tests saving trending data to JSON files
func TestSaveRaw(t *testing.T) {
	logger := logging.NewLogger("info")
	fetcher := New([]string{"go"}, logger)

	// Create test data
	testDate := time.Date(2024, 2, 7, 0, 0, 0, 0, time.UTC)
	repos := []models.RepoMetadata{
		{
			Owner:       "owner1",
			Name:        "repo1",
			URL:         "https://github.com/owner1/repo1",
			Description: "Test repo 1",
			Language:    "go",
			Topics:      []string{"testing", "golang"},
			Stars:       1234,
			Forks:       56,
			CreatedAt:   testDate,
		},
		{
			Owner:       "owner2",
			Name:        "repo2",
			URL:         "https://github.com/owner2/repo2",
			Description: "Test repo 2",
			Language:    "python",
			Topics:      []string{"ai", "ml"},
			Stars:       5678,
			Forks:       123,
			CreatedAt:   testDate,
		},
	}

	// Save the data
	err := fetcher.SaveRaw(repos, testDate)
	if err != nil {
		t.Fatalf("SaveRaw failed: %v", err)
	}

	// Verify the file was created with correct name
	expectedFilename := "data/trending_raw/2024-02-07.json"
	if _, err := os.Stat(expectedFilename); os.IsNotExist(err) {
		t.Fatalf("Expected file %s was not created", expectedFilename)
	}

	// Read the file back and verify content
	data, err := os.ReadFile(expectedFilename)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	var loadedRepos []models.RepoMetadata
	err = json.Unmarshal(data, &loadedRepos)
	if err != nil {
		t.Fatalf("Failed to unmarshal saved data: %v", err)
	}

	if len(loadedRepos) != len(repos) {
		t.Errorf("Expected %d repos, got %d", len(repos), len(loadedRepos))
	}

	// Verify first repo
	if loadedRepos[0].Owner != "owner1" {
		t.Errorf("Expected owner 'owner1', got '%s'", loadedRepos[0].Owner)
	}
	if loadedRepos[0].Name != "repo1" {
		t.Errorf("Expected name 'repo1', got '%s'", loadedRepos[0].Name)
	}
	if loadedRepos[0].Stars != 1234 {
		t.Errorf("Expected 1234 stars, got %d", loadedRepos[0].Stars)
	}

	// Clean up
	os.Remove(expectedFilename)
}

// TestSaveRawEmptyList tests error handling for empty repository list
func TestSaveRawEmptyList(t *testing.T) {
	logger := logging.NewLogger("info")
	fetcher := New([]string{"go"}, logger)

	testDate := time.Date(2024, 2, 7, 0, 0, 0, 0, time.UTC)
	err := fetcher.SaveRaw([]models.RepoMetadata{}, testDate)

	if err == nil {
		t.Error("Expected error for empty repository list, got nil")
	}
}

// TestSaveRawISODateFormat tests that filenames use ISO date format
func TestSaveRawISODateFormat(t *testing.T) {
	logger := logging.NewLogger("info")
	fetcher := New([]string{"go"}, logger)

	tests := []struct {
		name         string
		date         time.Time
		expectedFile string
	}{
		{
			"standard date",
			time.Date(2024, 2, 7, 0, 0, 0, 0, time.UTC),
			"data/trending_raw/2024-02-07.json",
		},
		{
			"single digit month and day",
			time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
			"data/trending_raw/2024-01-05.json",
		},
		{
			"end of year",
			time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			"data/trending_raw/2023-12-31.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := []models.RepoMetadata{
				{Owner: "test", Name: "repo", URL: "https://github.com/test/repo"},
			}

			err := fetcher.SaveRaw(repos, tt.date)
			if err != nil {
				t.Fatalf("SaveRaw failed: %v", err)
			}

			// Verify file exists
			if _, err := os.Stat(tt.expectedFile); os.IsNotExist(err) {
				t.Errorf("Expected file %s was not created", tt.expectedFile)
			}

			// Clean up
			os.Remove(tt.expectedFile)
		})
	}
}
