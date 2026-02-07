package pipeline

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/calculator"
	"ai-repo-insights/internal/classifier"
	"ai-repo-insights/internal/config"
	apperrors "ai-repo-insights/internal/errors"
	"ai-repo-insights/internal/fetcher"
	"ai-repo-insights/internal/history"
	"ai-repo-insights/internal/llm"
	"ai-repo-insights/internal/models"
	"ai-repo-insights/internal/report"
	"ai-repo-insights/internal/summary"
)

// Orchestrator executes complete workflow with error handling and logging
type Orchestrator struct {
	config config.Config
	logger zerolog.Logger
}

// NewOrchestrator creates a new pipeline orchestrator
func NewOrchestrator(cfg config.Config, logger zerolog.Logger) *Orchestrator {
	return &Orchestrator{
		config: cfg,
		logger: logger,
	}
}

// RunPipeline executes complete pipeline
func (o *Orchestrator) RunPipeline(reportID string) (models.PipelineResult, error) {
	o.logger.Info().Str("report_id", reportID).Msg("starting pipeline execution")
	pipelineStart := time.Now()

	// Generate report ID if not provided
	if reportID == "" {
		reportID = o.generateReportID()
	}

	// 1. Fetch trending
	stepStart := time.Now()
	o.logger.Info().Msg("step 1: fetching trending repositories")
	
	trendingFetcher := fetcher.New(o.config.Languages, o.logger)
	now := time.Now()
	trendingRepos, err := trendingFetcher.FetchTrending()
	if err != nil {
		return models.PipelineResult{Success: false, Error: err.Error()}, 
			apperrors.NewDataFetchError("failed to fetch trending data", err)
	}
	
	if err := trendingFetcher.SaveRaw(trendingRepos, now); err != nil {
		o.logger.Warn().Err(err).Msg("failed to save raw trending data")
	}
	
	o.logger.Info().
		Int("repo_count", len(trendingRepos)).
		Dur("duration", time.Since(stepStart)).
		Msg("step 1 completed")

	// 2. Classify repos
	stepStart = time.Now()
	o.logger.Info().Msg("step 2: classifying repositories")
	
	repoClassifier := classifier.New(o.config.Keywords)
	classifiedRepos := repoClassifier.Classify(trendingRepos)
	
	o.logger.Info().
		Int("classified_count", len(classifiedRepos)).
		Dur("duration", time.Since(stepStart)).
		Msg("step 2 completed")

	if len(classifiedRepos) == 0 {
		return models.PipelineResult{Success: false, Error: "no repositories matched filter criteria"},
			fmt.Errorf("no repositories matched filter criteria")
	}

	// 3. Calculate scores
	stepStart = time.Now()
	o.logger.Info().Msg("step 3: calculating scores")
	
	calc := calculator.New(o.config.Settings.WindowDays, o.config.Settings.ShortWindowDays)
	scoredRepos := calc.CalculateScores(classifiedRepos)
	topRepos := calc.RankAndSelectTop(scoredRepos, o.config.Settings.TopN)
	
	o.logger.Info().
		Int("top_count", len(topRepos)).
		Dur("duration", time.Since(stepStart)).
		Msg("step 3 completed")

	// 4. Update history
	stepStart = time.Now()
	o.logger.Info().Msg("step 4: updating history")
	
	now = time.Now()
	runDate := now.Format("2006-01-02")
	historyManager := history.NewManager("data/history.json", o.logger)
	hist, err := historyManager.LoadHistory()
	if err != nil {
		o.logger.Warn().Err(err).Msg("failed to load history, starting fresh")
		hist = models.NewHistory()
	}
	
	hist, err = historyManager.UpdateHistory(topRepos, reportID, runDate)
	if err != nil {
		o.logger.Warn().Err(err).Msg("failed to update history")
	}
	
	if err := historyManager.SaveHistory(hist); err != nil {
		o.logger.Warn().Err(err).Msg("failed to save history")
	}
	
	o.logger.Info().
		Dur("duration", time.Since(stepStart)).
		Msg("step 4 completed")

	// 5. Build summary
	stepStart = time.Now()
	o.logger.Info().Msg("step 5: building summary")
	
	summaryBuilder := summary.NewBuilder(o.config.Settings)
	summaryJSON := summaryBuilder.BuildSummary(topRepos, hist, runDate)
	
	o.logger.Info().
		Dur("duration", time.Since(stepStart)).
		Msg("step 5 completed")

	// 6. Call LLM
	stepStart = time.Now()
	o.logger.Info().Msg("step 6: calling LLM for analysis")
	
	llmAPIKey := os.Getenv("LLM_API_KEY")
	if llmAPIKey == "" {
		o.logger.Warn().Msg("LLM_API_KEY not set, using template fallback")
	}
	
	var llmOutput models.LLMOutput
	if llmAPIKey != "" {
		llmClient := llm.NewClient(o.config.LLM, llmAPIKey, o.logger)
		llmOutput, err = llmClient.GenerateAnalysis(summaryJSON, o.config.Settings.ReportLanguage)
		if err != nil {
			o.logger.Warn().Err(err).Msg("LLM call failed, using template fallback")
			llmOutput = llm.GenerateTemplateFallback(summaryJSON)
		}
	} else {
		llmOutput = llm.GenerateTemplateFallback(summaryJSON)
	}
	
	o.logger.Info().
		Dur("duration", time.Since(stepStart)).
		Msg("step 6 completed")

	// 7. Generate report
	stepStart = time.Now()
	o.logger.Info().Msg("step 7: generating report")
	
	reportGenerator := report.NewGenerator(o.config.Settings, o.config.Keywords)
	reportContent := reportGenerator.GenerateReport(summaryJSON, llmOutput, reportID, o.config.Languages)
	
	if err := reportGenerator.SaveReport(reportContent, reportID); err != nil {
		return models.PipelineResult{Success: false, Error: err.Error()},
			apperrors.NewFilesystemError("failed to save report", "reports/"+reportID+".md", err)
	}
	
	o.logger.Info().
		Dur("duration", time.Since(stepStart)).
		Msg("step 7 completed")

	// 8. Save summary backup
	if err := o.saveSummaryBackup(summaryJSON, reportID); err != nil {
		o.logger.Warn().Err(err).Msg("failed to save summary backup")
	}

	// Pipeline complete
	o.logger.Info().
		Str("report_id", reportID).
		Dur("total_duration", time.Since(pipelineStart)).
		Msg("pipeline execution completed successfully")

	return models.PipelineResult{
		Success:  true,
		ReportID: reportID,
	}, nil
}

// generateReportID generates report ID based on configured format
func (o *Orchestrator) generateReportID() string {
	now := time.Now()
	
	// For now, use simple YYYY-MM-DD format
	// TODO: Implement format parsing for YYYY-MM-weekN
	return now.Format("2006-01-02")
}

// saveSummaryBackup saves summary JSON to data/summaries/
func (o *Orchestrator) saveSummaryBackup(summaryJSON models.SummaryJSON, reportID string) error {
	summariesDir := "data/summaries"
	if err := os.MkdirAll(summariesDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/%s.json", summariesDir, reportID)
	
	data, err := models.MarshalJSON(summaryJSON)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
