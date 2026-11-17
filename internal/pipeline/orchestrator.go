// Package pipeline runs one weekly radar issue end to end.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/githubapi"
	"ai-repo-insights/internal/history"
	"ai-repo-insights/internal/models"
	"ai-repo-insights/internal/radar"
)

// Deps are the pipeline's external systems, injected so tests can fake them.
type Deps struct {
	Trending interface {
		FetchTrending() ([]models.RepoMetadata, error)
	}
	Classify func([]models.RepoMetadata) []models.ClassifiedRepo
	GitHub   interface {
		Repo(context.Context, string) (githubapi.RepoDetails, error)
		Readme(context.Context, string) (string, error)
	}
	Copy interface {
		WriteRadarCopy([]radar.Candidate, string) (radar.Copy, error)
	}
	History *history.Manager
	OutDir  string
	Now     time.Time
}

// Run fetches this week's trending AI repositories, writes the issue file and
// returns its path. History is saved only after the issue is written, so a
// failed run leaves no trace and can simply be re-run.
func Run(ctx context.Context, cfg config.Config, deps Deps, logger zerolog.Logger) (string, error) {
	reportID := radar.ReportID(deps.Now)
	logger.Info().Str("report_id", reportID).Msg("starting radar run")

	trending, err := deps.Trending.FetchTrending()
	if err != nil {
		return "", fmt.Errorf("fetch trending: %w", err)
	}
	classified := deps.Classify(trending)
	if len(classified) == 0 {
		return "", fmt.Errorf("no trending repository matched the AI keyword filter (fetched %d); check config/keywords.json and the trending scraper", len(trending))
	}

	hist, err := deps.History.LoadHistory()
	if err != nil {
		return "", fmt.Errorf("load history: %w", err)
	}
	unseen := radar.Unseen(classified, hist)
	logger.Info().Int("trending_ai", len(classified)).Int("unseen", len(unseen)).Msg("selected unseen repositories")

	cands := make([]radar.Candidate, 0, len(unseen))
	for _, repo := range unseen {
		details, err := deps.GitHub.Repo(ctx, repo.Key())
		if err != nil {
			return "", err
		}
		readme, err := deps.GitHub.Readme(ctx, repo.Key())
		if err != nil {
			return "", err
		}
		cands = append(cands, radar.Assign(repo, details, readme, deps.Now, cfg.Settings.NewRepoThresholdDays))
	}

	copy, err := deps.Copy.WriteRadarCopy(cands, cfg.WritingGuide)
	if err != nil {
		return "", err
	}

	path, err := radar.WriteIssue(deps.OutDir, radar.BuildIssue(deps.Now, cands, copy))
	if err != nil {
		return "", err
	}

	keys := make([]string, 0, len(classified))
	for _, repo := range classified {
		keys = append(keys, repo.Key())
	}
	updated, err := deps.History.UpdateHistory(keys, reportID, deps.Now.UTC().Format("2006-01-02"))
	if err != nil {
		return "", rollbackIssue(path, fmt.Errorf("update history after writing %s: %w", path, err))
	}
	if err := deps.History.SaveHistory(updated); err != nil {
		return "", rollbackIssue(path, fmt.Errorf("save history after writing %s: %w", path, err))
	}

	logger.Info().Str("issue", path).Int("candidates", len(cands)).Msg("radar run completed")
	return path, nil
}

func rollbackIssue(path string, cause error) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Join(cause, fmt.Errorf("remove issue %s after history failure: %w", path, err))
	}
	return cause
}
