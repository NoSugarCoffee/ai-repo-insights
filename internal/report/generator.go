package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-repo-insights/internal/config"
	apperrors "ai-repo-insights/internal/errors"
	"ai-repo-insights/internal/models"
)

// Generator combines data and LLM output into formatted Markdown report
type Generator struct {
	settings config.Settings
	keywords config.KeywordConfig
}

// NewGenerator creates a new report generator
func NewGenerator(settings config.Settings, keywords config.KeywordConfig) *Generator {
	return &Generator{
		settings: settings,
		keywords: keywords,
	}
}

// GenerateReport generates complete Markdown report
func (g *Generator) GenerateReport(
	summary models.SummaryJSON,
	llmOutput models.LLMOutput,
	reportID string,
	languages []string,
) string {
	var sb strings.Builder

	// Header
	sb.WriteString(g.formatHeader(summary, reportID, languages))
	sb.WriteString("\n\n")

	// Overview
	sb.WriteString("## Overview\n\n")
	sb.WriteString(SanitizeMarkdown(llmOutput.Intro))
	sb.WriteString("\n\n")

	// Top N table
	sb.WriteString(g.formatTopTable(summary.TopRepos, summary.Meta.TopN))
	sb.WriteString("\n\n")

	// Category breakdown
	sb.WriteString(g.formatCategoryBreakdown(summary, llmOutput))
	sb.WriteString("\n\n")

	// Dark horses
	if len(summary.DarkHorses) > 0 {
		sb.WriteString(g.formatDarkHorses(summary.DarkHorses, llmOutput.DarkHorseNotes))
		sb.WriteString("\n\n")
	}

	// Repeaters
	if len(summary.Repeaters) > 0 {
		sb.WriteString(g.formatRepeaters(summary.Repeaters, llmOutput.RepeatersNotes))
		sb.WriteString("\n\n")
	}

	// Highlights
	if len(llmOutput.Highlights) > 0 {
		sb.WriteString(g.formatHighlights(llmOutput.Highlights))
		sb.WriteString("\n\n")
	}

	// Methodology
	sb.WriteString(g.formatMethodology(summary.Meta))

	return sb.String()
}

// formatHeader generates report title and metadata
func (g *Generator) formatHeader(summary models.SummaryJSON, reportID string, languages []string) string {
	languageList := strings.Join(languages, ", ")

	return fmt.Sprintf(`# %s GitHub Trending Report - %s

**Report Date**: %s  
**Analysis Window**: %d days  
**Languages Tracked**: %s  
**Top N**: %d`,
		summary.Meta.FilterDomain,
		reportID,
		summary.Meta.RunDate,
		summary.Meta.WindowDays,
		languageList,
		summary.Meta.TopN,
	)
}

// formatTopTable generates top N ranking table
func (g *Generator) formatTopTable(repos []models.TopRepoInfo, topN int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Top %d Repositories\n\n", topN))
	sb.WriteString("| Rank | Repository | Category | Language | Heat_7 | Heat_30 | Score |\n")
	sb.WriteString("|------|-----------|----------|----------|---------|---------|-------|\n")

	for _, repo := range repos {
		sb.WriteString(fmt.Sprintf("| %d | [%s](%s) | %s | %s | %s | %s | %s |\n",
			repo.Rank,
			SanitizeRepoName(repo.RepoName),
			SanitizeURL(repo.URL),
			repo.Category,
			repo.Language,
			formatNumber(repo.Heat7),
			formatNumber(repo.Heat30),
			formatNumber(repo.Score),
		))
	}

	return sb.String()
}

// formatCategoryBreakdown generates per-category sections
func (g *Generator) formatCategoryBreakdown(summary models.SummaryJSON, llmOutput models.LLMOutput) string {
	var sb strings.Builder

	sb.WriteString("## Category Breakdown\n\n")

	// Group repos by category
	reposByCategory := make(map[string][]models.TopRepoInfo)
	for _, repo := range summary.TopRepos {
		reposByCategory[repo.Category] = append(reposByCategory[repo.Category], repo)
	}

	// Format each category
	for _, catStats := range summary.Categories {
		sb.WriteString(fmt.Sprintf("### %s (%d projects)\n\n", catStats.Name, catStats.Count))

		// LLM notes
		if note, exists := llmOutput.CategoryNotes[catStats.Name]; exists && note != "" {
			sb.WriteString(SanitizeMarkdown(note))
			sb.WriteString("\n\n")
		}

		// Statistics
		sb.WriteString(fmt.Sprintf("**Average Heat_7**: %s  \n", formatNumber(int(catStats.AvgHeat7))))
		sb.WriteString(fmt.Sprintf("**Average Score**: %s  \n\n", formatNumber(int(catStats.AvgScore))))

		// List repos in this category
		if repos, exists := reposByCategory[catStats.Name]; exists {
			for _, repo := range repos {
				sb.WriteString(fmt.Sprintf("- [%s](%s) - %s\n",
					SanitizeRepoName(repo.RepoName),
					SanitizeURL(repo.URL),
					SanitizeDescription(repo.Description)))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// formatDarkHorses generates dark horse section
func (g *Generator) formatDarkHorses(darkHorses []models.DarkHorseInfo, notes string) string {
	var sb strings.Builder

	sb.WriteString("## Dark Horse Projects\n\n")
	sb.WriteString(SanitizeMarkdown(notes))
	sb.WriteString("\n\n")

	sb.WriteString("| Repository | Score | Heat_30 | Heat_7 | Category |\n")
	sb.WriteString("|-----------|-------|---------|---------|----------|\n")

	for _, dh := range darkHorses {
		sb.WriteString(fmt.Sprintf("| [%s](%s) | %s | %s | %s | %s |\n",
			SanitizeRepoName(dh.RepoName),
			SanitizeURL(dh.URL),
			formatNumber(dh.Score),
			formatNumber(dh.Heat30),
			formatNumber(dh.Heat7),
			dh.Category,
		))
	}

	return sb.String()
}

// formatRepeaters generates repeater section
func (g *Generator) formatRepeaters(repeaters []models.RepeaterInfo, notes string) string {
	var sb strings.Builder

	sb.WriteString("## Consecutive Appearances\n\n")
	sb.WriteString(SanitizeMarkdown(notes))
	sb.WriteString("\n\n")

	sb.WriteString("| Repository | Weeks in Top | Category | Current Heat_7 |\n")
	sb.WriteString("|-----------|--------------|----------|----------------|\n")

	for _, rep := range repeaters {
		sb.WriteString(fmt.Sprintf("| [%s](%s) | %d | %s | %s |\n",
			SanitizeRepoName(rep.RepoName),
			SanitizeURL(rep.URL),
			rep.WeeksInTop,
			rep.Category,
			formatNumber(rep.CurrentHeat7),
		))
	}

	return sb.String()
}

// formatHighlights generates highlighted repositories section
func (g *Generator) formatHighlights(highlights []models.HighlightComment) string {
	var sb strings.Builder

	sb.WriteString("## Highlighted Repositories\n\n")

	for _, highlight := range highlights {
		sb.WriteString(fmt.Sprintf("### %s\n\n", highlight.Repo))
		sb.WriteString(SanitizeMarkdown(highlight.Comment))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// formatMethodology generates methodology explanation section
func (g *Generator) formatMethodology(meta models.MetaInfo) string {
	includeKeywords := strings.Join(g.keywords.Include, ", ")
	excludeKeywords := strings.Join(g.keywords.Exclude, ", ")

	categoryList := make([]string, 0, len(g.keywords.Categories))
	for cat := range g.keywords.Categories {
		categoryList = append(categoryList, cat)
	}
	categories := strings.Join(categoryList, ", ")

	return fmt.Sprintf(`## Methodology

**Data Sources**:
- GitHub Trending pages
- GitHub API stargazer timestamps

**Metrics**:
- Heat_7: Stars gained in last 7 days
- Heat_30: Stars gained in last 30 days
- Score: Weighted scoring combining short-term heat and sustained growth
  - Formula: 0.6 × stars_1d + 0.3 × (stars_7d / 7) + 0.1 × (stars_30d / 30)
  - Emphasizes recent activity (60%%) while considering sustained trends (40%%)

**Filtering**:
- Include keywords: %s
- Exclude keywords: %s
- Categories: %s

**Ranking**:
1. Sort by Heat_30 (descending)
2. Tie-break by Score (descending)
3. Select top %d

**Limitations**:
- Trending data limited to GitHub's trending algorithm
- Star history may be incomplete for repos with >40k stars (API pagination limits)
- LLM-generated commentary is interpretive, not prescriptive
- Weekly snapshots may miss short-lived trends`,
		meta.WindowDays,
		meta.ShortWindowDays,
		meta.ShortWindowDays,
		includeKeywords,
		excludeKeywords,
		categories,
		meta.TopN,
	)
}

// SaveReport saves report to reports/{report_id}.md
func (g *Generator) SaveReport(content string, reportID string) error {
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return apperrors.NewFilesystemError("failed to create reports directory", reportsDir, err)
	}

	filename := filepath.Join(reportsDir, reportID+".md")

	// Add generation timestamp
	timestamp := time.Now().UTC().Format(time.RFC3339)
	contentWithTimestamp := fmt.Sprintf("%s\n\n---\n*Generated at: %s*\n", content, timestamp)

	if err := os.WriteFile(filename, []byte(contentWithTimestamp), 0644); err != nil {
		return apperrors.NewFilesystemError("failed to write report file", filename, err)
	}

	return nil
}

// formatNumber formats numbers with thousands separators
func formatNumber(n int) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}

	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	return formatNumber(n/1000) + "," + fmt.Sprintf("%03d", n%1000)
}

// formatAcceleration formats acceleration as percentage
// e.g., 45 → "+45%", -12 → "-12%"
func formatAcceleration(accel int) string {
	sign := ""
	if accel > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%d%%", sign, accel)
}
