package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// NewRepoMetadata creates a new RepoMetadata instance
func NewRepoMetadata(owner string, name string, url string, description string, language string, topics []string, stars int, forks int, createdAt time.Time) *RepoMetadata {
	return &RepoMetadata{
		Owner:       owner,
		Name:        name,
		URL:         url,
		Description: description,
		Language:    language,
		Topics:      topics,
		Stars:       stars,
		Forks:       forks,
		CreatedAt:   createdAt,
	}
}

// NewClassifiedRepo creates a new ClassifiedRepo instance
func NewClassifiedRepo(metadata RepoMetadata, categories []string, primaryCategory string, matchScore int) *ClassifiedRepo {
	return &ClassifiedRepo{
		Metadata:        metadata,
		Categories:      categories,
		PrimaryCategory: primaryCategory,
		MatchScore:      matchScore,
	}
}

// NewRepoHistory creates a new RepoHistory instance
func NewRepoHistory(weeksInTop int, lastSeenReport string, lastSeenDate string, firstSeenReport string, firstSeenDate string) *RepoHistory {
	return &RepoHistory{
		WeeksInTop:      weeksInTop,
		LastSeenReport:  lastSeenReport,
		LastSeenDate:    lastSeenDate,
		FirstSeenReport: firstSeenReport,
		FirstSeenDate:   firstSeenDate,
	}
}

// FormatRepoURL formats a repository URL from owner and name
func FormatRepoURL(owner string, name string) string {
	return fmt.Sprintf("https://github.com/%s/%s", owner, name)
}

// FormatRepoKey formats a repository key from owner and name
func FormatRepoKey(owner string, name string) string {
	return fmt.Sprintf("%s/%s", owner, name)
}

// ParseRepoKey parses a repository key into owner and name
func ParseRepoKey(key string) (owner string, name string, err error) {
	var parsed [2]string
	n := 0
	start := 0

	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			if n >= 2 {
				return "", "", fmt.Errorf("invalid repo key format: %s", key)
			}
			parsed[n] = key[start:i]
			n++
			start = i + 1
		}
	}

	if n != 1 {
		return "", "", fmt.Errorf("invalid repo key format: %s", key)
	}

	parsed[n] = key[start:]

	return parsed[0], parsed[1], nil
}

// MarshalJSON marshals any value to JSON with indentation
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// Seen reports whether the repository key has ever appeared in a report.
func (h *History) Seen(key string) bool {
	_, ok := h.History[key]
	return ok
}
