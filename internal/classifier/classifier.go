package classifier

import (
	"strings"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/models"
)

// Classifier filters and categorizes repositories based on keyword rules
type Classifier struct {
	keywords config.KeywordConfig
}

// New creates a new Classifier with the given keyword configuration
func New(keywords config.KeywordConfig) *Classifier {
	return &Classifier{
		keywords: keywords,
	}
}

// Classify filters and categorizes repositories based on keyword rules
// Returns only repositories that pass the filter (match include, don't match exclude)
func (c *Classifier) Classify(repos []models.RepoMetadata) []models.ClassifiedRepo {
	var classified []models.ClassifiedRepo

	for _, repo := range repos {
		if c.matchesInclude(repo) && !c.matchesExclude(repo) {
			categories := c.assignCategories(repo)
			primaryCategory := c.selectPrimaryCategory(repo, categories)
			matchScore := c.calculateMatchScore(repo)

			classified = append(classified, models.ClassifiedRepo{
				Metadata:        repo,
				Categories:      categories,
				PrimaryCategory: primaryCategory,
				MatchScore:      matchScore,
			})
		}
	}

	return classified
}

// matchesInclude checks if the repository matches any include keyword
func (c *Classifier) matchesInclude(repo models.RepoMetadata) bool {
	searchText := c.buildSearchText(repo)

	for _, keyword := range c.keywords.Include {
		if c.matchesKeyword(searchText, keyword) {
			return true
		}
	}

	return false
}

// matchesExclude checks if the repository matches any exclude keyword
func (c *Classifier) matchesExclude(repo models.RepoMetadata) bool {
	searchText := c.buildSearchText(repo)

	for _, keyword := range c.keywords.Exclude {
		if c.matchesKeyword(searchText, keyword) {
			return true
		}
	}

	return false
}

// matchesKeyword checks if a keyword matches with word boundaries
func (c *Classifier) matchesKeyword(text string, keyword string) bool {
	keyword = strings.ToLower(keyword)
	text = strings.ToLower(text)
	
	// For hyphenated keywords like "machine-learning", match as-is
	if strings.Contains(keyword, "-") {
		return strings.Contains(text, keyword)
	}
	
	// For single words, check if keyword appears as whole word or as part of related words
	// This handles: exact match, plurals, and word stems (e.g., "agent" matches "agentic")
	words := strings.Fields(text)
	for _, word := range words {
		// Remove common punctuation
		word = strings.Trim(word, ".,;:!?()[]{}\"'")
		
		// Exact match or plural match
		if word == keyword || word == keyword+"s" {
			return true
		}
		
		// Stem matching: if word starts with keyword (handles "agentic" from "agent")
		// Only for keywords >= 4 chars to avoid false positives
		if len(keyword) >= 4 && strings.HasPrefix(word, keyword) {
			return true
		}
	}
	
	return false
}

// assignCategories assigns all matching categories to the repository
func (c *Classifier) assignCategories(repo models.RepoMetadata) []string {
	searchText := c.buildSearchText(repo)
	var categories []string

	for categoryName, keywords := range c.keywords.Categories {
		for _, keyword := range keywords {
			if c.matchesKeyword(searchText, keyword) {
				categories = append(categories, categoryName)
				break // Only add category once even if multiple keywords match
			}
		}
	}

	return categories
}

// selectPrimaryCategory selects the primary category based on keyword match frequency
func (c *Classifier) selectPrimaryCategory(repo models.RepoMetadata, categories []string) string {
	if len(categories) == 0 {
		return ""
	}

	searchText := c.buildSearchText(repo)
	maxMatches := 0
	primaryCategory := categories[0] // Default to first category

	for _, categoryName := range categories {
		matches := 0
		keywords := c.keywords.Categories[categoryName]

		for _, keyword := range keywords {
			if c.matchesKeyword(searchText, keyword) {
				matches++
			}
		}

		if matches > maxMatches {
			maxMatches = matches
			primaryCategory = categoryName
		}
	}

	return primaryCategory
}

// calculateMatchScore counts the total number of keyword matches
func (c *Classifier) calculateMatchScore(repo models.RepoMetadata) int {
	searchText := c.buildSearchText(repo)
	score := 0

	// Count include keyword matches
	for _, keyword := range c.keywords.Include {
		if c.matchesKeyword(searchText, keyword) {
			score++
		}
	}

	// Count category keyword matches
	for _, keywords := range c.keywords.Categories {
		for _, keyword := range keywords {
			if c.matchesKeyword(searchText, keyword) {
				score++
			}
		}
	}

	return score
}

// buildSearchText concatenates repo name, description, and topics into a lowercase search string
func (c *Classifier) buildSearchText(repo models.RepoMetadata) string {
	parts := []string{
		repo.Name,
		repo.Description,
	}
	parts = append(parts, repo.Topics...)

	return strings.ToLower(strings.Join(parts, " "))
}
