package calculator

import (
	"sort"

	"ai-repo-insights/internal/models"
)

// ScoreCalculator calculates ranking metrics from star history
type ScoreCalculator struct {
	windowDays      int
	shortWindowDays int
}

// New creates a new ScoreCalculator with the specified time windows
func New(windowDays int, shortWindowDays int) *ScoreCalculator {
	return &ScoreCalculator{
		windowDays:      windowDays,
		shortWindowDays: shortWindowDays,
	}
}

// CalculateScores calculates Heat_7, Heat_30, Acceleration for all repos
// Uses stars gained from trending data (today, week, month)
func (sc *ScoreCalculator) CalculateScores(repos []models.ClassifiedRepo) []models.ScoredRepo {
	scoredRepos := make([]models.ScoredRepo, 0, len(repos))

	for _, repo := range repos {
		starsToday := repo.Metadata.StarsToday
		starsThisWeek := repo.Metadata.StarsThisWeek
		starsThisMonth := repo.Metadata.StarsThisMonth
		
		// Heat_30: Use StarsThisMonth directly (GitHub's "month" is ~30 days)
		heat30 := starsThisMonth
		
		// Heat_7: Use StarsThisWeek directly (GitHub's "week" is 7 days)
		heat7 := starsThisWeek
		
		// Score: Weighted scoring combining short-term heat and sustained growth
		// Formula: 0.6 × today + 0.3 × (week/7) + 0.1 × (month/30)
		// Emphasizes recent activity (60%) while considering sustained trends (40%)
		
		dailyRate := float64(starsToday)
		weeklyAvgRate := float64(starsThisWeek) / 7.0
		monthlyAvgRate := float64(starsThisMonth) / 30.0
		
		scoreValue := dailyRate*0.6 + weeklyAvgRate*0.3 + monthlyAvgRate*0.1
		score := int(scoreValue)

		scoredRepo := models.ScoredRepo{
			Repo:       repo,
			TotalStars: repo.Metadata.Stars,
			Heat7:      heat7,
			Heat30:     heat30,
			Prev30:     0,
			Score:      score,
		}

		scoredRepos = append(scoredRepos, scoredRepo)
	}

	return scoredRepos
}



// RankAndSelectTop ranks repositories by (heat_30 desc, acceleration desc) and selects top N
func (sc *ScoreCalculator) RankAndSelectTop(scoredRepos []models.ScoredRepo, topN int) []models.ScoredRepo {
	// Sort by heat_30 descending, then by acceleration descending
	ranked := sc.RankRepositories(scoredRepos)
	
	// Select top N
	if len(ranked) > topN {
		return ranked[:topN]
	}
	return ranked
}

// RankRepositories sorts repositories by (heat_30 desc, acceleration desc)
func (sc *ScoreCalculator) RankRepositories(scoredRepos []models.ScoredRepo) []models.ScoredRepo {
	// Create a copy to avoid modifying the input
	ranked := make([]models.ScoredRepo, len(scoredRepos))
	copy(ranked, scoredRepos)
	
	// Sort by heat_30 descending (primary), then score descending (secondary)
	sort.SliceStable(ranked, func(i int, j int) bool {
		// Primary sort: heat_30 descending
		if ranked[i].Heat30 != ranked[j].Heat30 {
			return ranked[i].Heat30 > ranked[j].Heat30
		}
		// Secondary sort: score descending
		return ranked[i].Score > ranked[j].Score
	})
	
	return ranked
}
