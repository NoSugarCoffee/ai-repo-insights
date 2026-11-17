package models

import (
	"time"
)

// RepoMetadata represents raw repository metadata from trending
type RepoMetadata struct {
	Owner          string    `json:"owner"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Description    string    `json:"description"`
	Language       string    `json:"language"`
	Topics         []string  `json:"topics"`
	Stars          int       `json:"stars"`
	Forks          int       `json:"forks"`
	StarsToday     int       `json:"stars_today"`
	StarsThisWeek  int       `json:"stars_this_week"`
	StarsThisMonth int       `json:"stars_this_month"`
	CreatedAt      time.Time `json:"created_at"`
}

// Key returns the repository key in format "owner/repo"
func (r *RepoMetadata) Key() string {
	return r.Owner + "/" + r.Name
}

// ClassifiedRepo represents a repository with category assignments
type ClassifiedRepo struct {
	Metadata        RepoMetadata `json:"metadata"`
	Categories      []string     `json:"categories"`
	PrimaryCategory string       `json:"primary_category"`
	MatchScore      int          `json:"match_score"`
}

// Key returns the repository key
func (c *ClassifiedRepo) Key() string {
	return c.Metadata.Key()
}

// RepoHistory represents historical tracking for a single repository
type RepoHistory struct {
	WeeksInTop      int    `json:"weeks_in_top"`
	LastSeenReport  string `json:"last_seen_report"`
	LastSeenDate    string `json:"last_seen_date"`
	FirstSeenReport string `json:"first_seen_report"`
	FirstSeenDate   string `json:"first_seen_date"`
}

// History represents complete historical tracking
type History struct {
	LatestReport string                 `json:"latest_report"`
	History      map[string]RepoHistory `json:"history"`
}

// NewHistory creates a new empty history
func NewHistory() *History {
	return &History{
		History: make(map[string]RepoHistory),
	}
}
