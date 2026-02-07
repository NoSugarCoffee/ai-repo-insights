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

// ScoredRepo represents a repository with calculated metrics
type ScoredRepo struct {
	Repo       ClassifiedRepo `json:"repo"`
	TotalStars int            `json:"total_stars"`
	Heat7      int            `json:"heat_7"`
	Heat30     int            `json:"heat_30"`
	Prev30     int            `json:"prev_30"`
	Score      int            `json:"score"`
}

// Key returns the repository key
func (s *ScoredRepo) Key() string {
	return s.Repo.Key()
}

// SortKey returns a tuple for sorting (heat_30, score)
func (s *ScoredRepo) SortKey() (int, int) {
	return s.Heat30, s.Score
}

// RepoHistory represents historical tracking for a single repository
type RepoHistory struct {
	WeeksInTop       int    `json:"weeks_in_top"`
	LastSeenReport   string `json:"last_seen_report"`
	LastSeenDate     string `json:"last_seen_date"`
	FirstSeenReport  string `json:"first_seen_report"`
	FirstSeenDate    string `json:"first_seen_date"`
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

// MetaInfo represents metadata for the summary
type MetaInfo struct {
	RunDate          string `json:"run_date"`
	WindowDays       int    `json:"window_days"`
	ShortWindowDays  int    `json:"short_window_days"`
	TopN             int    `json:"top_n"`
	FilterDomain     string `json:"filter_domain"`
}

// CategoryStats represents statistics for a category
type CategoryStats struct {
	Name        string  `json:"name"`
	Count       int     `json:"count"`
	AvgHeat7    float64 `json:"avg_heat_7"`
	AvgScore    float64 `json:"avg_score"`
}

// LanguageStats represents statistics for a language
type LanguageStats struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// NewReposInfo represents information about new repositories
type NewReposInfo struct {
	Count         int      `json:"count"`
	ThresholdDays int      `json:"threshold_days"`
	Repos         []string `json:"repos"`
}

// DarkHorseInfo represents a dark horse repository
type DarkHorseInfo struct {
	RepoKey string `json:"repo_key"`
	RepoName string `json:"repo_name"`
	URL     string `json:"url"`
	Score   int    `json:"score"`
	Heat30  int    `json:"heat_30"`
	Heat7   int    `json:"heat_7"`
	Category string `json:"category"`
}

// RepeaterInfo represents a repeater repository
type RepeaterInfo struct {
	RepoKey     string `json:"repo_key"`
	RepoName    string `json:"repo_name"`
	URL         string `json:"url"`
	WeeksInTop  int    `json:"weeks_in_top"`
	CurrentHeat7 int    `json:"current_heat_7"`
	Category    string `json:"category"`
}

// TopRepoInfo represents a top repository with full details
type TopRepoInfo struct {
	Rank     int    `json:"rank"`
	RepoKey  string `json:"repo_key"`
	RepoName string `json:"repo_name"`
	URL      string `json:"url"`
	Category string `json:"category"`
	Language string `json:"language"`
	Heat7    int    `json:"heat_7"`
	Heat30   int    `json:"heat_30"`
	Score    int    `json:"score"`
	Description string `json:"description"`
}

// SummaryJSON represents the complete summary for LLM
type SummaryJSON struct {
	Meta       MetaInfo        `json:"meta"`
	Categories []CategoryStats `json:"categories"`
	Languages  []LanguageStats `json:"languages"`
	NewRepos   NewReposInfo    `json:"new_repos"`
	DarkHorses []DarkHorseInfo `json:"dark_horses"`
	Repeaters  []RepeaterInfo  `json:"repeaters"`
	TopRepos   []TopRepoInfo   `json:"top_repos"`
}

// HighlightComment represents a highlighted repository comment
type HighlightComment struct {
	Repo    string `json:"repo"`
	Comment string `json:"comment"`
	Tone    string `json:"tone"`
}

// LLMOutput represents the output from LLM analysis
type LLMOutput struct {
	Intro           string                      `json:"intro"`
	CategoryNotes   map[string]string           `json:"category_notes"`
	DarkHorseNotes  string                      `json:"dark_horse_notes"`
	RepeatersNotes  string                      `json:"repeaters_notes"`
	Highlights      []HighlightComment          `json:"highlights"`
}

// PipelineResult represents the result of a pipeline execution
type PipelineResult struct {
	Success  bool   `json:"success"`
	ReportID string `json:"report_id,omitempty"`
	Error    string `json:"error,omitempty"`
}
