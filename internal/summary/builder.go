package summary

import (
	"time"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/models"
)

// Builder builds summary JSON from scored repositories and history
type Builder struct {
	settings config.Settings
}

// NewBuilder creates a new summary builder
func NewBuilder(settings config.Settings) *Builder {
	return &Builder{
		settings: settings,
	}
}

// BuildSummary builds complete summary JSON for LLM
func (b *Builder) BuildSummary(
	topRepos []models.ScoredRepo,
	history *models.History,
	runDate string,
) models.SummaryJSON {
	return models.SummaryJSON{
		Meta:       b.buildMeta(runDate),
		Categories: b.aggregateCategories(topRepos),
		Languages:  b.aggregateLanguages(topRepos),
		NewRepos:   b.identifyNewRepos(topRepos),
		DarkHorses: b.identifyDarkHorses(topRepos),
		Repeaters:  b.identifyRepeaters(topRepos, history),
		TopRepos:   b.buildTopRepos(topRepos),
	}
}

// buildMeta creates metadata for the summary
func (b *Builder) buildMeta(runDate string) models.MetaInfo {
	return models.MetaInfo{
		RunDate:         runDate,
		WindowDays:      b.settings.WindowDays,
		ShortWindowDays: b.settings.ShortWindowDays,
		TopN:            b.settings.TopN,
		FilterDomain:    b.settings.FilterDomain,
	}
}

// aggregateCategories calculates per-category statistics
func (b *Builder) aggregateCategories(repos []models.ScoredRepo) []models.CategoryStats {
	categoryMap := make(map[string]*categoryAggregator)

	for _, repo := range repos {
		category := repo.Repo.PrimaryCategory
		if _, exists := categoryMap[category]; !exists {
			categoryMap[category] = &categoryAggregator{
				name:     category,
				count:    0,
				heat7Sum: 0,
				scoreSum: 0,
			}
		}
		agg := categoryMap[category]
		agg.count++
		agg.heat7Sum += repo.Heat7
		agg.scoreSum += repo.Score
	}

	stats := make([]models.CategoryStats, 0, len(categoryMap))
	for _, agg := range categoryMap {
		stats = append(stats, models.CategoryStats{
			Name:     agg.name,
			Count:    agg.count,
			AvgHeat7: float64(agg.heat7Sum) / float64(agg.count),
			AvgScore: float64(agg.scoreSum) / float64(agg.count),
		})
	}

	return stats
}

// categoryAggregator accumulates category statistics
type categoryAggregator struct {
	name     string
	count    int
	heat7Sum int
	scoreSum int
}

// aggregateLanguages calculates per-language statistics
func (b *Builder) aggregateLanguages(repos []models.ScoredRepo) []models.LanguageStats {
	languageMap := make(map[string]int)

	for _, repo := range repos {
		language := repo.Repo.Metadata.Language
		languageMap[language]++
	}

	stats := make([]models.LanguageStats, 0, len(languageMap))
	for lang, count := range languageMap {
		stats = append(stats, models.LanguageStats{
			Name:  lang,
			Count: count,
		})
	}

	return stats
}

// identifyNewRepos finds repos created within threshold
func (b *Builder) identifyNewRepos(repos []models.ScoredRepo) models.NewReposInfo {
	now := time.Now()
	thresholdDate := now.AddDate(0, 0, -b.settings.NewRepoThresholdDays)

	newRepos := make([]string, 0)
	for _, repo := range repos {
		if repo.Repo.Metadata.CreatedAt.After(thresholdDate) {
			newRepos = append(newRepos, repo.Key())
		}
	}

	return models.NewReposInfo{
		Count:         len(newRepos),
		ThresholdDays: b.settings.NewRepoThresholdDays,
		Repos:         newRepos,
	}
}

// identifyDarkHorses finds repos with high acceleration
func (b *Builder) identifyDarkHorses(repos []models.ScoredRepo) []models.DarkHorseInfo {
	darkHorses := make([]models.DarkHorseInfo, 0)

	for _, repo := range repos {
		if repo.Score >= b.settings.DarkHorseAccelThreshold {
			darkHorses = append(darkHorses, models.DarkHorseInfo{
				RepoKey:  repo.Key(),
				RepoName: repo.Repo.Metadata.Name,
				URL:      repo.Repo.Metadata.URL,
				Score:    repo.Score,
				Heat30:   repo.Heat30,
				Heat7:    repo.Heat7,
				Category: repo.Repo.PrimaryCategory,
			})
		}
	}

	return darkHorses
}

// identifyRepeaters finds repos with consecutive appearances (weeks_in_top >= 2)
func (b *Builder) identifyRepeaters(repos []models.ScoredRepo, history *models.History) []models.RepeaterInfo {
	repeaters := make([]models.RepeaterInfo, 0)

	for _, repo := range repos {
		key := repo.Key()
		if histEntry, exists := history.History[key]; exists && histEntry.WeeksInTop >= 2 {
			repeaters = append(repeaters, models.RepeaterInfo{
				RepoKey:      key,
				RepoName:     repo.Repo.Metadata.Name,
				URL:          repo.Repo.Metadata.URL,
				WeeksInTop:   histEntry.WeeksInTop,
				CurrentHeat7: repo.Heat7,
				Category:     repo.Repo.PrimaryCategory,
			})
		}
	}

	return repeaters
}

// buildTopRepos creates top repository info list
func (b *Builder) buildTopRepos(repos []models.ScoredRepo) []models.TopRepoInfo {
	topRepos := make([]models.TopRepoInfo, len(repos))

	for i, repo := range repos {
		topRepos[i] = models.TopRepoInfo{
			Rank:        i + 1,
			RepoKey:     repo.Key(),
			RepoName:    repo.Repo.Metadata.Name,
			URL:         repo.Repo.Metadata.URL,
			Category:    repo.Repo.PrimaryCategory,
			Language:    repo.Repo.Metadata.Language,
			Heat7:       repo.Heat7,
			Heat30:      repo.Heat30,
			Score:       repo.Score,
			Description: repo.Repo.Metadata.Description,
		}
	}

	return topRepos
}
