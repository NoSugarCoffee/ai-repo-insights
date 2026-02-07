package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ai-repo-insights/internal/errors"
	"ai-repo-insights/internal/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog"
)

const (
	maxRetries = 3
	retryDelay = 2 * time.Second
	baseURL    = "https://github.com/trending"
)

// TrendingFetcher fetches trending repositories from GitHub
type TrendingFetcher struct {
	languages []string
	client    *http.Client
	logger    zerolog.Logger
}

// New creates a new TrendingFetcher
func New(languages []string, logger zerolog.Logger) *TrendingFetcher {
	return &TrendingFetcher{
		languages: languages,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// FetchTrending fetches trending repositories for all configured languages
func (f *TrendingFetcher) FetchTrending() ([]models.RepoMetadata, error) {
	f.logger.Info().Strs("languages", f.languages).Msg("Starting trending fetch for languages")

	var allRepos []models.RepoMetadata
	var lastErr error

	// Fetch trending repos for each language
	for _, lang := range f.languages {
		repos, err := f.fetchTrendingForLanguage(lang)
		if err != nil {
			f.logger.Error().Str("language", lang).Err(err).Msg("Failed to fetch trending for language")
			lastErr = err
			continue
		}
		allRepos = append(allRepos, repos...)
	}

	// Check if we got any results
	if len(allRepos) == 0 {
		if lastErr != nil {
			return nil, errors.NewDataFetchError("failed to fetch trending data for any language", lastErr)
		}
		return nil, errors.NewDataFetchError("no trending data retrieved", nil)
	}

	// Deduplicate repositories
	allRepos = f.deduplicateRepos(allRepos)

	f.logger.Info().Int("total_repos", len(allRepos)).Msg("Trending fetch completed")
	return allRepos, nil
}

// fetchTrendingForLanguage fetches trending repositories for a specific language
func (f *TrendingFetcher) fetchTrendingForLanguage(language string) ([]models.RepoMetadata, error) {
	f.logger.Debug().Str("language", language).Msg("Fetching trending for language")

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Fetch today, week, and month data
		reposToday, err := f.scrapeTrendingPage(language, "daily")
		if err != nil {
			lastErr = err
			f.logger.Warn().Str("language", language).Int("attempt", attempt).Err(err).Msg("Fetch attempt failed for today")
			if attempt < maxRetries {
				time.Sleep(retryDelay * time.Duration(attempt))
			}
			continue
		}

		reposWeek, err := f.scrapeTrendingPage(language, "weekly")
		if err != nil {
			lastErr = err
			f.logger.Warn().Str("language", language).Int("attempt", attempt).Err(err).Msg("Fetch attempt failed for week")
			if attempt < maxRetries {
				time.Sleep(retryDelay * time.Duration(attempt))
			}
			continue
		}

		reposMonth, err := f.scrapeTrendingPage(language, "monthly")
		if err != nil {
			lastErr = err
			f.logger.Warn().Str("language", language).Int("attempt", attempt).Err(err).Msg("Fetch attempt failed for month")
			if attempt < maxRetries {
				time.Sleep(retryDelay * time.Duration(attempt))
			}
			continue
		}

		repos := f.mergeRepos(reposToday, reposWeek, reposMonth, language)
		f.logger.Debug().Str("language", language).Int("repos", len(repos)).Msg("Successfully fetched trending")
		return repos, nil
	}

	return nil, errors.NewDataFetchError(fmt.Sprintf("failed to fetch trending for %s after %d attempts", language, maxRetries), lastErr)
}

// scrapeTrendingPage scrapes a GitHub trending page for a specific language and timeframe
func (f *TrendingFetcher) scrapeTrendingPage(language string, since string) (map[string]*models.RepoMetadata, error) {
	url := fmt.Sprintf("%s/%s?since=%s", baseURL, language, since)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	repos := make(map[string]*models.RepoMetadata)
	
	// Parse each repository article
	doc.Find("article.Box-row").Each(func(i int, s *goquery.Selection) {
		// Extract owner and repo name from h2 a[href]
		repoLink := s.Find("h2 a").First()
		href, exists := repoLink.Attr("href")
		if !exists {
			return
		}
		
		// href format: /owner/repo
		parts := strings.Split(strings.Trim(href, "/"), "/")
		if len(parts) != 2 {
			return
		}
		
		owner := parts[0]
		name := parts[1]
		key := owner + "/" + name
		
		// Extract description
		description := strings.TrimSpace(s.Find("p.col-9").Text())
		
		// Extract stars gained
		starsText := s.Find("span.d-inline-block.float-sm-right").Text()
		stars := f.parseStars(starsText)
		
		repos[key] = &models.RepoMetadata{
			Owner:       owner,
			Name:        name,
			URL:         "https://github.com" + href,
			Description: description,
			Language:    language,
			Topics:      []string{},
			Stars:       0,
			Forks:       0,
			CreatedAt:   time.Now(),
		}
		
		// Store stars based on timeframe
		switch since {
		case "daily":
			repos[key].StarsToday = stars
		case "weekly":
			repos[key].StarsThisWeek = stars
		case "monthly":
			repos[key].StarsThisMonth = stars
		}
	})
	
	return repos, nil
}

// parseStars extracts the star count from text like "1,234 stars today"
func (f *TrendingFetcher) parseStars(text string) int {
	// Remove commas and extract numbers
	re := regexp.MustCompile(`[\d,]+`)
	match := re.FindString(text)
	if match == "" {
		return 0
	}
	
	// Remove commas
	match = strings.ReplaceAll(match, ",", "")
	
	stars, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}
	
	return stars
}

// mergeRepos merges today, week, and month data
func (f *TrendingFetcher) mergeRepos(
	todayMap map[string]*models.RepoMetadata,
	weekMap map[string]*models.RepoMetadata,
	monthMap map[string]*models.RepoMetadata,
	language string,
) []models.RepoMetadata {
	// Collect all unique repo keys
	allKeys := make(map[string]bool)
	for key := range todayMap {
		allKeys[key] = true
	}
	for key := range weekMap {
		allKeys[key] = true
	}
	for key := range monthMap {
		allKeys[key] = true
	}
	
	repos := make([]models.RepoMetadata, 0, len(allKeys))
	
	for key := range allKeys {
		repo := models.RepoMetadata{
			Language:  language,
			Topics:    []string{},
			CreatedAt: time.Now(),
		}
		
		// Get data from today
		if todayRepo, exists := todayMap[key]; exists {
			repo.Owner = todayRepo.Owner
			repo.Name = todayRepo.Name
			repo.URL = todayRepo.URL
			repo.Description = todayRepo.Description
			repo.StarsToday = todayRepo.StarsToday
		}
		
		// Get data from week
		if weekRepo, exists := weekMap[key]; exists {
			if repo.Owner == "" {
				repo.Owner = weekRepo.Owner
				repo.Name = weekRepo.Name
				repo.URL = weekRepo.URL
				repo.Description = weekRepo.Description
			}
			repo.StarsThisWeek = weekRepo.StarsThisWeek
		}
		
		// Get data from month
		if monthRepo, exists := monthMap[key]; exists {
			if repo.Owner == "" {
				repo.Owner = monthRepo.Owner
				repo.Name = monthRepo.Name
				repo.URL = monthRepo.URL
				repo.Description = monthRepo.Description
			}
			repo.StarsThisMonth = monthRepo.StarsThisMonth
		}
		
		// Only add if we have at least owner and name
		if repo.Owner != "" && repo.Name != "" {
			repos = append(repos, repo)
		}
	}
	
	return repos
}

// deduplicateRepos removes duplicate repositories based on owner/name
func (f *TrendingFetcher) deduplicateRepos(repos []models.RepoMetadata) []models.RepoMetadata {
	seen := make(map[string]bool)
	var unique []models.RepoMetadata

	for _, repo := range repos {
		key := repo.Key()
		if !seen[key] {
			seen[key] = true
			unique = append(unique, repo)
		}
	}

	if len(repos) != len(unique) {
		f.logger.Debug().Int("original", len(repos)).Int("unique", len(unique)).Msg("Deduplicated repositories")
	}

	return unique
}

// SaveRaw saves trending data to a date-stamped JSON file
func (f *TrendingFetcher) SaveRaw(repos []models.RepoMetadata, date time.Time) error {
	if len(repos) == 0 {
		return fmt.Errorf("cannot save empty repository list")
	}

	// Format date as ISO date (YYYY-MM-DD)
	dateStr := date.Format("2006-01-02")
	dirPath := "data/trending_raw"
	filename := fmt.Sprintf("%s/%s.json", dirPath, dateStr)

	f.logger.Info().Str("filename", filename).Int("repos", len(repos)).Msg("Saving trending data")

	// Create directory if it doesn't exist
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return errors.NewFilesystemError("failed to create directory", dirPath, err)
	}

	// Marshal repos to JSON with indentation for readability
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return errors.NewFilesystemError("failed to marshal trending data", filename, err)
	}

	// Write to file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return errors.NewFilesystemError("failed to write trending data", filename, err)
	}

	f.logger.Info().Str("filename", filename).Msg("Successfully saved trending data")
	return nil
}
