package summary

import (
	"testing"
	"time"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/models"
)

func TestBuildSummary(t *testing.T) {
	settings := config.Settings{
		WindowDays:              90,
		ShortWindowDays:         30,
		TopN:                    10,
		NewRepoThresholdDays:    90,
		DarkHorseAccelThreshold: 100,
		FilterDomain:            "AI",
	}

	builder := NewBuilder(settings)

	// Create test data
	now := time.Now()
	oldDate := now.AddDate(0, 0, -200)
	newDate := now.AddDate(0, 0, -30)

	topRepos := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner:       "owner1",
					Name:        "repo1",
					URL:         "https://github.com/owner1/repo1",
					Description: "Test repo 1",
					Language:    "Python",
					CreatedAt:   oldDate,
				},
				PrimaryCategory: "llm",
			},
			Heat90:       500,
			Heat30:       200,
			Acceleration: 50,
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner:       "owner2",
					Name:        "repo2",
					URL:         "https://github.com/owner2/repo2",
					Description: "Test repo 2",
					Language:    "Go",
					CreatedAt:   newDate,
				},
				PrimaryCategory: "agent",
			},
			Heat90:       300,
			Heat30:       150,
			Acceleration: 120,
		},
	}

	history := &models.History{
		LatestReport: "2024-01-week1",
		History: map[string]models.RepoHistory{
			"owner1/repo1": {
				WeeksInTop: 3,
			},
		},
	}

	summary := builder.BuildSummary(topRepos, history, "2024-01-15")

	// Verify meta
	if summary.Meta.RunDate != "2024-01-15" {
		t.Errorf("Expected run_date 2024-01-15, got %s", summary.Meta.RunDate)
	}
	if summary.Meta.WindowDays != 90 {
		t.Errorf("Expected window_days 90, got %d", summary.Meta.WindowDays)
	}
	if summary.Meta.FilterDomain != "AI" {
		t.Errorf("Expected filter_domain AI, got %s", summary.Meta.FilterDomain)
	}

	// Verify categories
	if len(summary.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(summary.Categories))
	}

	// Verify languages
	if len(summary.Languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(summary.Languages))
	}

	// Verify new repos
	if summary.NewRepos.Count != 1 {
		t.Errorf("Expected 1 new repo, got %d", summary.NewRepos.Count)
	}
	if len(summary.NewRepos.Repos) != 1 {
		t.Errorf("Expected 1 repo in new repos list, got %d", len(summary.NewRepos.Repos))
	}

	// Verify dark horses
	if len(summary.DarkHorses) != 1 {
		t.Errorf("Expected 1 dark horse, got %d", len(summary.DarkHorses))
	}
	if summary.DarkHorses[0].Acceleration != 120 {
		t.Errorf("Expected dark horse acceleration 120, got %d", summary.DarkHorses[0].Acceleration)
	}

	// Verify repeaters
	if len(summary.Repeaters) != 1 {
		t.Errorf("Expected 1 repeater, got %d", len(summary.Repeaters))
	}
	if summary.Repeaters[0].WeeksInTop != 3 {
		t.Errorf("Expected repeater weeks_in_top 3, got %d", summary.Repeaters[0].WeeksInTop)
	}

	// Verify top repos
	if len(summary.TopRepos) != 2 {
		t.Errorf("Expected 2 top repos, got %d", len(summary.TopRepos))
	}
	if summary.TopRepos[0].Rank != 1 {
		t.Errorf("Expected first repo rank 1, got %d", summary.TopRepos[0].Rank)
	}
}

func TestAggregateCategories(t *testing.T) {
	settings := config.Settings{
		WindowDays:      90,
		ShortWindowDays: 30,
		TopN:            10,
		FilterDomain:    "AI",
	}

	builder := NewBuilder(settings)

	repos := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				PrimaryCategory: "llm",
			},
			Heat90:       500,
			Acceleration: 50,
		},
		{
			Repo: models.ClassifiedRepo{
				PrimaryCategory: "llm",
			},
			Heat90:       300,
			Acceleration: 30,
		},
		{
			Repo: models.ClassifiedRepo{
				PrimaryCategory: "agent",
			},
			Heat90:       200,
			Acceleration: 20,
		},
	}

	stats := builder.aggregateCategories(repos)

	if len(stats) != 2 {
		t.Fatalf("Expected 2 categories, got %d", len(stats))
	}

	// Find llm category
	var llmStats *models.CategoryStats
	for i := range stats {
		if stats[i].Name == "llm" {
			llmStats = &stats[i]
			break
		}
	}

	if llmStats == nil {
		t.Fatal("Expected to find llm category")
	}

	if llmStats.Count != 2 {
		t.Errorf("Expected llm count 2, got %d", llmStats.Count)
	}

	expectedAvgHeat90 := (500.0 + 300.0) / 2.0
	if llmStats.AvgHeat90 != expectedAvgHeat90 {
		t.Errorf("Expected llm avg_heat_90 %.2f, got %.2f", expectedAvgHeat90, llmStats.AvgHeat90)
	}

	expectedAvgAccel := (50.0 + 30.0) / 2.0
	if llmStats.AvgAcceleration != expectedAvgAccel {
		t.Errorf("Expected llm avg_acceleration %.2f, got %.2f", expectedAvgAccel, llmStats.AvgAcceleration)
	}
}

func TestAggregateLanguages(t *testing.T) {
	settings := config.Settings{
		WindowDays:      90,
		ShortWindowDays: 30,
		TopN:            10,
		FilterDomain:    "AI",
	}

	builder := NewBuilder(settings)

	repos := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Language: "Python",
				},
			},
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Language: "Python",
				},
			},
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Language: "Go",
				},
			},
		},
	}

	stats := builder.aggregateLanguages(repos)

	if len(stats) != 2 {
		t.Fatalf("Expected 2 languages, got %d", len(stats))
	}

	// Find Python language
	var pythonStats *models.LanguageStats
	for i := range stats {
		if stats[i].Name == "Python" {
			pythonStats = &stats[i]
			break
		}
	}

	if pythonStats == nil {
		t.Fatal("Expected to find Python language")
	}

	if pythonStats.Count != 2 {
		t.Errorf("Expected Python count 2, got %d", pythonStats.Count)
	}
}

func TestIdentifyNewRepos(t *testing.T) {
	settings := config.Settings{
		NewRepoThresholdDays: 90,
		FilterDomain:         "AI",
	}

	builder := NewBuilder(settings)

	now := time.Now()
	oldDate := now.AddDate(0, 0, -200)
	newDate := now.AddDate(0, 0, -30)

	repos := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner:     "owner1",
					Name:      "old-repo",
					CreatedAt: oldDate,
				},
			},
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner:     "owner2",
					Name:      "new-repo",
					CreatedAt: newDate,
				},
			},
		},
	}

	info := builder.identifyNewRepos(repos)

	if info.Count != 1 {
		t.Errorf("Expected 1 new repo, got %d", info.Count)
	}

	if info.ThresholdDays != 90 {
		t.Errorf("Expected threshold_days 90, got %d", info.ThresholdDays)
	}

	if len(info.Repos) != 1 {
		t.Fatalf("Expected 1 repo in list, got %d", len(info.Repos))
	}

	if info.Repos[0] != "owner2/new-repo" {
		t.Errorf("Expected owner2/new-repo, got %s", info.Repos[0])
	}
}

func TestIdentifyDarkHorses(t *testing.T) {
	settings := config.Settings{
		DarkHorseAccelThreshold: 100,
		FilterDomain:            "AI",
	}

	builder := NewBuilder(settings)

	repos := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner1",
					Name:  "slow-repo",
					URL:   "https://github.com/owner1/slow-repo",
				},
				PrimaryCategory: "llm",
			},
			Heat90:       500,
			Heat30:       200,
			Acceleration: 50,
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner2",
					Name:  "fast-repo",
					URL:   "https://github.com/owner2/fast-repo",
				},
				PrimaryCategory: "agent",
			},
			Heat90:       300,
			Heat30:       150,
			Acceleration: 120,
		},
	}

	darkHorses := builder.identifyDarkHorses(repos)

	if len(darkHorses) != 1 {
		t.Fatalf("Expected 1 dark horse, got %d", len(darkHorses))
	}

	dh := darkHorses[0]
	if dh.RepoKey != "owner2/fast-repo" {
		t.Errorf("Expected owner2/fast-repo, got %s", dh.RepoKey)
	}
	if dh.Acceleration != 120 {
		t.Errorf("Expected acceleration 120, got %d", dh.Acceleration)
	}
	if dh.Category != "agent" {
		t.Errorf("Expected category agent, got %s", dh.Category)
	}
}

func TestIdentifyRepeaters(t *testing.T) {
	settings := config.Settings{
		FilterDomain: "AI",
	}

	builder := NewBuilder(settings)

	repos := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner1",
					Name:  "repeater-repo",
					URL:   "https://github.com/owner1/repeater-repo",
				},
				PrimaryCategory: "llm",
			},
			Heat90: 500,
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner2",
					Name:  "new-repo",
					URL:   "https://github.com/owner2/new-repo",
				},
				PrimaryCategory: "agent",
			},
			Heat90: 300,
		},
	}

	history := &models.History{
		History: map[string]models.RepoHistory{
			"owner1/repeater-repo": {
				WeeksInTop: 3,
			},
			"owner2/new-repo": {
				WeeksInTop: 1,
			},
		},
	}

	repeaters := builder.identifyRepeaters(repos, history)

	if len(repeaters) != 1 {
		t.Fatalf("Expected 1 repeater, got %d", len(repeaters))
	}

	rep := repeaters[0]
	if rep.RepoKey != "owner1/repeater-repo" {
		t.Errorf("Expected owner1/repeater-repo, got %s", rep.RepoKey)
	}
	if rep.WeeksInTop != 3 {
		t.Errorf("Expected weeks_in_top 3, got %d", rep.WeeksInTop)
	}
	if rep.CurrentHeat90 != 500 {
		t.Errorf("Expected current_heat_90 500, got %d", rep.CurrentHeat90)
	}
}

func TestBuildTopRepos(t *testing.T) {
	settings := config.Settings{
		FilterDomain: "AI",
	}

	builder := NewBuilder(settings)

	repos := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner:       "owner1",
					Name:        "repo1",
					URL:         "https://github.com/owner1/repo1",
					Description: "First repo",
					Language:    "Python",
				},
				PrimaryCategory: "llm",
			},
			Heat90:       500,
			Heat30:       200,
			Acceleration: 50,
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner:       "owner2",
					Name:        "repo2",
					URL:         "https://github.com/owner2/repo2",
					Description: "Second repo",
					Language:    "Go",
				},
				PrimaryCategory: "agent",
			},
			Heat90:       300,
			Heat30:       150,
			Acceleration: 30,
		},
	}

	topRepos := builder.buildTopRepos(repos)

	if len(topRepos) != 2 {
		t.Fatalf("Expected 2 top repos, got %d", len(topRepos))
	}

	// Check first repo
	if topRepos[0].Rank != 1 {
		t.Errorf("Expected rank 1, got %d", topRepos[0].Rank)
	}
	if topRepos[0].RepoKey != "owner1/repo1" {
		t.Errorf("Expected owner1/repo1, got %s", topRepos[0].RepoKey)
	}
	if topRepos[0].Heat90 != 500 {
		t.Errorf("Expected heat_90 500, got %d", topRepos[0].Heat90)
	}

	// Check second repo
	if topRepos[1].Rank != 2 {
		t.Errorf("Expected rank 2, got %d", topRepos[1].Rank)
	}
	if topRepos[1].RepoKey != "owner2/repo2" {
		t.Errorf("Expected owner2/repo2, got %s", topRepos[1].RepoKey)
	}
}
