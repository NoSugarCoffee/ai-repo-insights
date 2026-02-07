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

// NewScoredRepo creates a new ScoredRepo instance
func NewScoredRepo(repo ClassifiedRepo, totalStars int, heat7 int, heat30 int, prev30 int, score int) *ScoredRepo {
	return &ScoredRepo{
		Repo:       repo,
		TotalStars: totalStars,
		Heat7:      heat7,
		Heat30:     heat30,
		Prev30:     prev30,
		Score:      score,
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


// NewMetaInfo creates a new MetaInfo instance
func NewMetaInfo(runDate string, windowDays int, shortWindowDays int, topN int, filterDomain string) *MetaInfo {
	return &MetaInfo{
		RunDate:         runDate,
		WindowDays:      windowDays,
		ShortWindowDays: shortWindowDays,
		TopN:            topN,
		FilterDomain:    filterDomain,
	}
}

// NewCategoryStats creates a new CategoryStats instance
func NewCategoryStats(name string, count int, avgHeat7 float64, avgScore float64) *CategoryStats {
	return &CategoryStats{
		Name:     name,
		Count:    count,
		AvgHeat7: avgHeat7,
		AvgScore: avgScore,
	}
}

// NewLanguageStats creates a new LanguageStats instance
func NewLanguageStats(name string, count int) *LanguageStats {
	return &LanguageStats{
		Name:  name,
		Count: count,
	}
}

// NewNewReposInfo creates a new NewReposInfo instance
func NewNewReposInfo(count int, thresholdDays int, repos []string) *NewReposInfo {
	return &NewReposInfo{
		Count:         count,
		ThresholdDays: thresholdDays,
		Repos:         repos,
	}
}

// NewDarkHorseInfo creates a new DarkHorseInfo instance
func NewDarkHorseInfo(repoKey string, repoName string, url string, score int, heat30 int, heat7 int, category string) *DarkHorseInfo {
	return &DarkHorseInfo{
		RepoKey:  repoKey,
		RepoName: repoName,
		URL:      url,
		Score:    score,
		Heat30:   heat30,
		Heat7:    heat7,
		Category: category,
	}
}

// NewRepeaterInfo creates a new RepeaterInfo instance
func NewRepeaterInfo(repoKey string, repoName string, url string, weeksInTop int, currentHeat7 int, category string) *RepeaterInfo {
	return &RepeaterInfo{
		RepoKey:      repoKey,
		RepoName:     repoName,
		URL:          url,
		WeeksInTop:   weeksInTop,
		CurrentHeat7: currentHeat7,
		Category:     category,
	}
}

// NewTopRepoInfo creates a new TopRepoInfo instance
func NewTopRepoInfo(rank int, repoKey string, repoName string, url string, category string, language string, heat7 int, heat30 int, score int, description string) *TopRepoInfo {
	return &TopRepoInfo{
		Rank:        rank,
		RepoKey:     repoKey,
		RepoName:    repoName,
		URL:         url,
		Category:    category,
		Language:    language,
		Heat7:       heat7,
		Heat30:      heat30,
		Score:       score,
		Description: description,
	}
}

// NewSummaryJSON creates a new SummaryJSON instance
func NewSummaryJSON(meta MetaInfo, categories []CategoryStats, languages []LanguageStats, newRepos NewReposInfo, darkHorses []DarkHorseInfo, repeaters []RepeaterInfo, topRepos []TopRepoInfo) *SummaryJSON {
	return &SummaryJSON{
		Meta:       meta,
		Categories: categories,
		Languages:  languages,
		NewRepos:   newRepos,
		DarkHorses: darkHorses,
		Repeaters:  repeaters,
		TopRepos:   topRepos,
	}
}

// NewHighlightComment creates a new HighlightComment instance
func NewHighlightComment(repo string, comment string, tone string) *HighlightComment {
	return &HighlightComment{
		Repo:    repo,
		Comment: comment,
		Tone:    tone,
	}
}

// NewLLMOutput creates a new LLMOutput instance
func NewLLMOutput(intro string, categoryNotes map[string]string, darkHorseNotes string, repeatersNotes string, highlights []HighlightComment) *LLMOutput {
	return &LLMOutput{
		Intro:          intro,
		CategoryNotes:  categoryNotes,
		DarkHorseNotes: darkHorseNotes,
		RepeatersNotes: repeatersNotes,
		Highlights:     highlights,
	}
}

// NewPipelineResult creates a new PipelineResult instance
func NewPipelineResult(success bool, reportID string, errorMsg string) *PipelineResult {
	return &PipelineResult{
		Success:  success,
		ReportID: reportID,
		Error:    errorMsg,
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
