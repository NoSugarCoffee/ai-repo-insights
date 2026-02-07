package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/models"
)

func TestLoadHistory_NewFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	// Execute
	history, err := manager.LoadHistory()

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if history == nil {
		t.Fatal("expected history to be non-nil")
	}
	if history.History == nil {
		t.Fatal("expected history.History map to be initialized")
	}
	if len(history.History) != 0 {
		t.Errorf("expected empty history, got %d entries", len(history.History))
	}
	if history.LatestReport != "" {
		t.Errorf("expected empty latest report, got %s", history.LatestReport)
	}
}

func TestLoadHistory_ExistingFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.json")
	logger := zerolog.Nop()

	// Create existing history file
	existingHistory := models.History{
		LatestReport: "2024-01-week1",
		History: map[string]models.RepoHistory{
			"owner1/repo1": {
				WeeksInTop:      2,
				LastSeenReport:  "2024-01-week1",
				LastSeenDate:    "2024-01-07",
				FirstSeenReport: "2023-12-week4",
				FirstSeenDate:   "2023-12-31",
			},
		},
	}
	data, _ := json.MarshalIndent(existingHistory, "", "  ")
	os.WriteFile(historyPath, data, 0644)

	manager := NewManager(historyPath, logger)

	// Execute
	history, err := manager.LoadHistory()

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if history.LatestReport != "2024-01-week1" {
		t.Errorf("expected latest report '2024-01-week1', got '%s'", history.LatestReport)
	}
	if len(history.History) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history.History))
	}
	
	repo, exists := history.History["owner1/repo1"]
	if !exists {
		t.Fatal("expected owner1/repo1 in history")
	}
	if repo.WeeksInTop != 2 {
		t.Errorf("expected weeks_in_top=2, got %d", repo.WeeksInTop)
	}
}

func TestUpdateHistory_NewRepo(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	currentTop := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner1",
					Name:  "repo1",
				},
			},
			Heat90: 1000,
		},
	}

	// Execute
	history, err := manager.UpdateHistory(currentTop, "2024-01-week1", "2024-01-07")

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if history.LatestReport != "2024-01-week1" {
		t.Errorf("expected latest report '2024-01-week1', got '%s'", history.LatestReport)
	}
	
	repo, exists := history.History["owner1/repo1"]
	if !exists {
		t.Fatal("expected owner1/repo1 in history")
	}
	if repo.WeeksInTop != 1 {
		t.Errorf("expected weeks_in_top=1, got %d", repo.WeeksInTop)
	}
	if repo.LastSeenReport != "2024-01-week1" {
		t.Errorf("expected last_seen_report='2024-01-week1', got '%s'", repo.LastSeenReport)
	}
	if repo.LastSeenDate != "2024-01-07" {
		t.Errorf("expected last_seen_date='2024-01-07', got '%s'", repo.LastSeenDate)
	}
	if repo.FirstSeenReport != "2024-01-week1" {
		t.Errorf("expected first_seen_report='2024-01-week1', got '%s'", repo.FirstSeenReport)
	}
	if repo.FirstSeenDate != "2024-01-07" {
		t.Errorf("expected first_seen_date='2024-01-07', got '%s'", repo.FirstSeenDate)
	}
}

func TestUpdateHistory_ExistingRepo(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	// Create existing history
	existingHistory := models.History{
		LatestReport: "2024-01-week1",
		History: map[string]models.RepoHistory{
			"owner1/repo1": {
				WeeksInTop:      1,
				LastSeenReport:  "2024-01-week1",
				LastSeenDate:    "2024-01-07",
				FirstSeenReport: "2024-01-week1",
				FirstSeenDate:   "2024-01-07",
			},
		},
	}
	data, _ := json.MarshalIndent(existingHistory, "", "  ")
	os.WriteFile(historyPath, data, 0644)

	currentTop := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner1",
					Name:  "repo1",
				},
			},
			Heat90: 1000,
		},
	}

	// Execute
	history, err := manager.UpdateHistory(currentTop, "2024-01-week2", "2024-01-14")

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	
	repo, exists := history.History["owner1/repo1"]
	if !exists {
		t.Fatal("expected owner1/repo1 in history")
	}
	if repo.WeeksInTop != 2 {
		t.Errorf("expected weeks_in_top=2, got %d", repo.WeeksInTop)
	}
	if repo.LastSeenReport != "2024-01-week2" {
		t.Errorf("expected last_seen_report='2024-01-week2', got '%s'", repo.LastSeenReport)
	}
	if repo.LastSeenDate != "2024-01-14" {
		t.Errorf("expected last_seen_date='2024-01-14', got '%s'", repo.LastSeenDate)
	}
	// First seen should remain unchanged
	if repo.FirstSeenReport != "2024-01-week1" {
		t.Errorf("expected first_seen_report='2024-01-week1', got '%s'", repo.FirstSeenReport)
	}
	if repo.FirstSeenDate != "2024-01-07" {
		t.Errorf("expected first_seen_date='2024-01-07', got '%s'", repo.FirstSeenDate)
	}
}

func TestUpdateHistory_RemoveRepoNotInTop(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	// Create existing history with two repos
	existingHistory := models.History{
		LatestReport: "2024-01-week1",
		History: map[string]models.RepoHistory{
			"owner1/repo1": {
				WeeksInTop:      2,
				LastSeenReport:  "2024-01-week1",
				LastSeenDate:    "2024-01-07",
				FirstSeenReport: "2023-12-week4",
				FirstSeenDate:   "2023-12-31",
			},
			"owner2/repo2": {
				WeeksInTop:      1,
				LastSeenReport:  "2024-01-week1",
				LastSeenDate:    "2024-01-07",
				FirstSeenReport: "2024-01-week1",
				FirstSeenDate:   "2024-01-07",
			},
		},
	}
	data, _ := json.MarshalIndent(existingHistory, "", "  ")
	os.WriteFile(historyPath, data, 0644)

	// Current top only has repo1
	currentTop := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner1",
					Name:  "repo1",
				},
			},
			Heat90: 1000,
		},
	}

	// Execute
	history, err := manager.UpdateHistory(currentTop, "2024-01-week2", "2024-01-14")

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	
	// repo1 should still be in history with incremented counter
	repo1, exists := history.History["owner1/repo1"]
	if !exists {
		t.Fatal("expected owner1/repo1 in history")
	}
	if repo1.WeeksInTop != 3 {
		t.Errorf("expected weeks_in_top=3, got %d", repo1.WeeksInTop)
	}
	
	// repo2 should be removed (streak broken)
	_, exists = history.History["owner2/repo2"]
	if exists {
		t.Error("expected owner2/repo2 to be removed from history")
	}
	
	if len(history.History) != 1 {
		t.Errorf("expected 1 repo in history, got %d", len(history.History))
	}
}

func TestUpdateHistory_MultipleRepos(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	currentTop := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner1",
					Name:  "repo1",
				},
			},
			Heat90: 1000,
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner2",
					Name:  "repo2",
				},
			},
			Heat90: 900,
		},
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner3",
					Name:  "repo3",
				},
			},
			Heat90: 800,
		},
	}

	// Execute
	history, err := manager.UpdateHistory(currentTop, "2024-01-week1", "2024-01-07")

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(history.History) != 3 {
		t.Errorf("expected 3 repos in history, got %d", len(history.History))
	}
	
	for _, repo := range currentTop {
		key := repo.Key()
		histEntry, exists := history.History[key]
		if !exists {
			t.Errorf("expected %s in history", key)
			continue
		}
		if histEntry.WeeksInTop != 1 {
			t.Errorf("expected weeks_in_top=1 for %s, got %d", key, histEntry.WeeksInTop)
		}
	}
}

func TestSaveHistory(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "data", "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	history := &models.History{
		LatestReport: "2024-01-week1",
		History: map[string]models.RepoHistory{
			"owner1/repo1": {
				WeeksInTop:      2,
				LastSeenReport:  "2024-01-week1",
				LastSeenDate:    "2024-01-07",
				FirstSeenReport: "2023-12-week4",
				FirstSeenDate:   "2023-12-31",
			},
		},
	}

	// Execute
	err := manager.SaveHistory(history)

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Fatal("expected history file to exist")
	}

	// Verify content
	data, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("failed to read history file: %v", err)
	}

	var loaded models.History
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse history file: %v", err)
	}

	if loaded.LatestReport != "2024-01-week1" {
		t.Errorf("expected latest report '2024-01-week1', got '%s'", loaded.LatestReport)
	}
	if len(loaded.History) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(loaded.History))
	}
}

func TestSaveHistory_CreatesDirectory(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "nested", "dir", "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	history := models.NewHistory()
	history.LatestReport = "2024-01-week1"

	// Execute
	err := manager.SaveHistory(history)

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(historyPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("expected directory to be created")
	}

	// Verify file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Fatal("expected history file to exist")
	}
}

func TestUpdateHistory_EmptyCurrentTop(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	// Create existing history
	existingHistory := models.History{
		LatestReport: "2024-01-week1",
		History: map[string]models.RepoHistory{
			"owner1/repo1": {
				WeeksInTop:      1,
				LastSeenReport:  "2024-01-week1",
				LastSeenDate:    "2024-01-07",
				FirstSeenReport: "2024-01-week1",
				FirstSeenDate:   "2024-01-07",
			},
		},
	}
	data, _ := json.MarshalIndent(existingHistory, "", "  ")
	os.WriteFile(historyPath, data, 0644)

	// Empty current top
	currentTop := []models.ScoredRepo{}

	// Execute
	history, err := manager.UpdateHistory(currentTop, "2024-01-week2", "2024-01-14")

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	
	// All repos should be removed
	if len(history.History) != 0 {
		t.Errorf("expected empty history, got %d entries", len(history.History))
	}
	
	if history.LatestReport != "2024-01-week2" {
		t.Errorf("expected latest report '2024-01-week2', got '%s'", history.LatestReport)
	}
}

func TestUpdateHistory_PreservesFirstSeen(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	// Create existing history
	existingHistory := models.History{
		LatestReport: "2024-01-week1",
		History: map[string]models.RepoHistory{
			"owner1/repo1": {
				WeeksInTop:      1,
				LastSeenReport:  "2024-01-week1",
				LastSeenDate:    "2024-01-07",
				FirstSeenReport: "2024-01-week1",
				FirstSeenDate:   "2024-01-07",
			},
		},
	}
	data, _ := json.MarshalIndent(existingHistory, "", "  ")
	os.WriteFile(historyPath, data, 0644)

	currentTop := []models.ScoredRepo{
		{
			Repo: models.ClassifiedRepo{
				Metadata: models.RepoMetadata{
					Owner: "owner1",
					Name:  "repo1",
				},
			},
			Heat90: 1000,
		},
	}

	// Execute multiple updates
	history, err := manager.UpdateHistory(currentTop, "2024-01-week2", "2024-01-14")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	
	// Save and reload
	manager.SaveHistory(history)
	
	history, err = manager.UpdateHistory(currentTop, "2024-01-week3", "2024-01-21")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify
	repo, exists := history.History["owner1/repo1"]
	if !exists {
		t.Fatal("expected owner1/repo1 in history")
	}
	
	// First seen should remain unchanged
	if repo.FirstSeenReport != "2024-01-week1" {
		t.Errorf("expected first_seen_report='2024-01-week1', got '%s'", repo.FirstSeenReport)
	}
	if repo.FirstSeenDate != "2024-01-07" {
		t.Errorf("expected first_seen_date='2024-01-07', got '%s'", repo.FirstSeenDate)
	}
	
	// Last seen should be updated
	if repo.LastSeenReport != "2024-01-week3" {
		t.Errorf("expected last_seen_report='2024-01-week3', got '%s'", repo.LastSeenReport)
	}
	if repo.LastSeenDate != "2024-01-21" {
		t.Errorf("expected last_seen_date='2024-01-21', got '%s'", repo.LastSeenDate)
	}
	
	// Weeks should be incremented
	if repo.WeeksInTop != 3 {
		t.Errorf("expected weeks_in_top=3, got %d", repo.WeeksInTop)
	}
}

// TestSaveHistory_PreservesAllMetadata verifies that SaveHistory preserves
// all historical metadata fields as required by Requirements 5.6 and 14.4
func TestSaveHistory_PreservesAllMetadata(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "data", "history.json")
	logger := zerolog.Nop()
	manager := NewManager(historyPath, logger)

	// Create history with complete metadata
	history := &models.History{
		LatestReport: "2024-01-week3",
		History: map[string]models.RepoHistory{
			"owner1/repo1": {
				WeeksInTop:      3,
				LastSeenReport:  "2024-01-week3",
				LastSeenDate:    "2024-01-21",
				FirstSeenReport: "2024-01-week1",
				FirstSeenDate:   "2024-01-07",
			},
			"owner2/repo2": {
				WeeksInTop:      2,
				LastSeenReport:  "2024-01-week3",
				LastSeenDate:    "2024-01-21",
				FirstSeenReport: "2024-01-week2",
				FirstSeenDate:   "2024-01-14",
			},
		},
	}

	// Execute - Save history
	err := manager.SaveHistory(history)
	if err != nil {
		t.Fatalf("SaveHistory failed: %v", err)
	}

	// Verify - Load back and check all metadata is preserved
	data, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("failed to read saved history: %v", err)
	}

	var loaded models.History
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse saved history: %v", err)
	}

	// Verify LatestReport is preserved (Requirement 5.6)
	if loaded.LatestReport != history.LatestReport {
		t.Errorf("LatestReport not preserved: expected %s, got %s",
			history.LatestReport, loaded.LatestReport)
	}

	// Verify all repos are preserved (Requirement 14.4)
	if len(loaded.History) != len(history.History) {
		t.Errorf("History map size not preserved: expected %d, got %d",
			len(history.History), len(loaded.History))
	}

	// Verify all metadata fields for each repo (Requirement 14.4)
	for repoKey, originalRepo := range history.History {
		loadedRepo, exists := loaded.History[repoKey]
		if !exists {
			t.Errorf("Repo %s not preserved in history", repoKey)
			continue
		}

		if loadedRepo.WeeksInTop != originalRepo.WeeksInTop {
			t.Errorf("%s: WeeksInTop not preserved: expected %d, got %d",
				repoKey, originalRepo.WeeksInTop, loadedRepo.WeeksInTop)
		}
		if loadedRepo.LastSeenReport != originalRepo.LastSeenReport {
			t.Errorf("%s: LastSeenReport not preserved: expected %s, got %s",
				repoKey, originalRepo.LastSeenReport, loadedRepo.LastSeenReport)
		}
		if loadedRepo.LastSeenDate != originalRepo.LastSeenDate {
			t.Errorf("%s: LastSeenDate not preserved: expected %s, got %s",
				repoKey, originalRepo.LastSeenDate, loadedRepo.LastSeenDate)
		}
		if loadedRepo.FirstSeenReport != originalRepo.FirstSeenReport {
			t.Errorf("%s: FirstSeenReport not preserved: expected %s, got %s",
				repoKey, originalRepo.FirstSeenReport, loadedRepo.FirstSeenReport)
		}
		if loadedRepo.FirstSeenDate != originalRepo.FirstSeenDate {
			t.Errorf("%s: FirstSeenDate not preserved: expected %s, got %s",
				repoKey, originalRepo.FirstSeenDate, loadedRepo.FirstSeenDate)
		}
	}
}
