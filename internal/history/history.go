package history

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/errors"
	"ai-repo-insights/internal/models"
)

// Manager handles tracking consecutive appearances in rankings
type Manager struct {
	historyPath string
	logger      zerolog.Logger
}

// NewManager creates a new history manager instance
func NewManager(historyPath string, logger zerolog.Logger) *Manager {
	return &Manager{
		historyPath: historyPath,
		logger:      logger,
	}
}

// LoadHistory loads existing history from JSON file
func (m *Manager) LoadHistory() (*models.History, error) {
	// Check if history file exists
	if _, err := os.Stat(m.historyPath); os.IsNotExist(err) {
		m.logger.Info().
			Str("path", m.historyPath).
			Msg("history file does not exist, creating new history")
		return models.NewHistory(), nil
	}

	data, err := os.ReadFile(m.historyPath)
	if err != nil {
		return nil, errors.NewFilesystemError("failed to read history file", m.historyPath, err)
	}

	var history models.History
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, errors.NewFilesystemError("failed to parse history file", m.historyPath, err)
	}

	// Ensure history map is initialized
	if history.History == nil {
		history.History = make(map[string]models.RepoHistory)
	}

	m.logger.Info().
		Str("path", m.historyPath).
		Int("tracked_repos", len(history.History)).
		Msg("loaded history")

	return &history, nil
}

// UpdateHistory updates history with current rankings
func (m *Manager) UpdateHistory(currentTop []models.ScoredRepo, reportID string, reportDate string) (*models.History, error) {
	// Load existing history
	history, err := m.LoadHistory()
	if err != nil {
		return nil, err
	}

	// Build set of current top repo keys
	currentRepoKeys := make(map[string]bool)
	for _, repo := range currentTop {
		currentRepoKeys[repo.Key()] = true
	}

	// Update or initialize repos in current top
	for _, repo := range currentTop {
		repoKey := repo.Key()
		
		if existingHistory, exists := history.History[repoKey]; exists {
			// Repo was in history, increment counter
			existingHistory.WeeksInTop++
			existingHistory.LastSeenReport = reportID
			existingHistory.LastSeenDate = reportDate
			history.History[repoKey] = existingHistory
			
			m.logger.Debug().
				Str("repo", repoKey).
				Int("weeks_in_top", existingHistory.WeeksInTop).
				Msg("updated existing repo in history")
		} else {
			// New repo, initialize
			history.History[repoKey] = models.RepoHistory{
				WeeksInTop:      1,
				LastSeenReport:  reportID,
				LastSeenDate:    reportDate,
				FirstSeenReport: reportID,
				FirstSeenDate:   reportDate,
			}
			
			m.logger.Debug().
				Str("repo", repoKey).
				Msg("added new repo to history")
		}
	}

	// Remove repos not in current top (streak broken)
	reposToRemove := []string{}
	for repoKey := range history.History {
		if !currentRepoKeys[repoKey] {
			reposToRemove = append(reposToRemove, repoKey)
		}
	}

	for _, repoKey := range reposToRemove {
		weeks := history.History[repoKey].WeeksInTop
		delete(history.History, repoKey)
		
		m.logger.Debug().
			Str("repo", repoKey).
			Int("final_weeks", weeks).
			Msg("removed repo from history (streak broken)")
	}

	// Update latest report
	history.LatestReport = reportID

	m.logger.Info().
		Str("report_id", reportID).
		Int("current_top", len(currentTop)).
		Int("tracked_repos", len(history.History)).
		Int("removed_repos", len(reposToRemove)).
		Msg("updated history")

	return history, nil
}

// SaveHistory saves updated history to JSON file
func (m *Manager) SaveHistory(history *models.History) error {
	// Ensure directory exists
	dir := filepath.Dir(m.historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.NewFilesystemError("failed to create history directory", dir, err)
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return errors.NewFilesystemError("failed to marshal history", m.historyPath, err)
	}

	if err := os.WriteFile(m.historyPath, data, 0644); err != nil {
		return errors.NewFilesystemError("failed to write history file", m.historyPath, err)
	}

	m.logger.Info().
		Str("path", m.historyPath).
		Int("tracked_repos", len(history.History)).
		Msg("saved history")

	return nil
}
