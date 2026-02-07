package llm

import (
	"fmt"

	"ai-repo-insights/internal/models"
)

// GenerateTemplateFallback creates a template-based report when LLM fails
func GenerateTemplateFallback(summary models.SummaryJSON) models.LLMOutput {
	return models.LLMOutput{
		Intro:          generateIntroFallback(summary),
		CategoryNotes:  generateCategoryNotesFallback(summary),
		DarkHorseNotes: generateDarkHorseNotesFallback(summary),
		RepeatersNotes: generateRepeatersNotesFallback(summary),
		Highlights:     generateHighlightsFallback(summary),
	}
}

// generateIntroFallback creates a template introduction
func generateIntroFallback(summary models.SummaryJSON) string {
	return fmt.Sprintf(
		"This report analyzes the top %d %s repositories based on %d-day star growth metrics. "+
			"The analysis covers %d categories across multiple programming languages.",
		summary.Meta.TopN,
		summary.Meta.FilterDomain,
		summary.Meta.WindowDays,
		len(summary.Categories),
	)
}

// generateCategoryNotesFallback creates template category notes
func generateCategoryNotesFallback(summary models.SummaryJSON) map[string]string {
	notes := make(map[string]string)

	for _, cat := range summary.Categories {
		notes[cat.Name] = fmt.Sprintf(
			"This category contains %d repositories with an average Heat_7 of %.0f stars and average score of %.0f.",
			cat.Count,
			cat.AvgHeat7,
			cat.AvgScore,
		)
	}

	return notes
}

// generateDarkHorseNotesFallback creates template dark horse notes
func generateDarkHorseNotesFallback(summary models.SummaryJSON) string {
	if len(summary.DarkHorses) == 0 {
		return "No dark horse projects identified in this period."
	}

	return fmt.Sprintf(
		"Identified %d dark horse projects showing exceptional scores in star growth, "+
			"indicating rapidly emerging interest from the developer community.",
		len(summary.DarkHorses),
	)
}

// generateRepeatersNotesFallback creates template repeater notes
func generateRepeatersNotesFallback(summary models.SummaryJSON) string {
	if len(summary.Repeaters) == 0 {
		return "No repeater projects identified in this period."
	}

	return fmt.Sprintf(
		"Found %d projects with consecutive appearances in top rankings, "+
			"demonstrating sustained community interest and development momentum.",
		len(summary.Repeaters),
	)
}

// generateHighlightsFallback creates template highlights
func generateHighlightsFallback(summary models.SummaryJSON) []models.HighlightComment {
	highlights := make([]models.HighlightComment, 0)

	// Select top 3 repos by Heat_30
	count := 3
	if len(summary.TopRepos) < count {
		count = len(summary.TopRepos)
	}

	for i := 0; i < count; i++ {
		repo := summary.TopRepos[i]
		comment := fmt.Sprintf(
			"Ranked #%d with %d stars gained in the last %d days. "+
				"Category: %s. Language: %s. Score: %d.",
			repo.Rank,
			repo.Heat7,
			summary.Meta.WindowDays,
			repo.Category,
			repo.Language,
			repo.Score,
		)

		highlights = append(highlights, models.HighlightComment{
			Repo:    repo.RepoKey,
			Comment: comment,
			Tone:    "neutral-analytical",
		})
	}

	return highlights
}
