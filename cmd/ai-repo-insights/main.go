package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/logging"
	"ai-repo-insights/internal/pipeline"
)

const (
	version = "1.0.0"
)

func main() {
	// Define CLI flags
	configDir := flag.String("config", "config", "Path to configuration directory")
	reportID := flag.String("report-id", "", "Custom report ID (default: auto-generated)")
	weekly := flag.Bool("weekly", false, "Use week-based report ID format")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("GitHub Insights v%s\n", version)
		os.Exit(0)
	}

	// Show help
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Initialize logger
	logger := logging.NewLogger(*logLevel)
	logger.Info().Str("version", version).Msg("starting GitHub Insights")

	// Load configuration
	logger.Info().Str("config_dir", *configDir).Msg("loading configuration")
	cfg, err := config.Load(*configDir)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Validate configuration
	errors := cfg.Validate()
	if len(errors) > 0 {
		logger.Error().Strs("validation_errors", errors).Msg("configuration validation failed")
		for _, errMsg := range errors {
			fmt.Fprintf(os.Stderr, "Configuration error: %s\n", errMsg)
		}
		os.Exit(1)
	}

	logger.Info().Msg("configuration loaded and validated successfully")

	// Check required environment variables
	if os.Getenv("GITHUB_TOKEN") == "" {
		logger.Warn().Msg("GITHUB_TOKEN not set - star history fetching will fail")
	}
	if os.Getenv("LLM_API_KEY") == "" {
		logger.Warn().Msg("LLM_API_KEY not set - will use template-based report")
	}

	// Generate report ID if needed
	finalReportID := *reportID
	if finalReportID == "" {
		if *weekly {
			finalReportID = generateWeeklyReportID()
		} else {
			finalReportID = generateDailyReportID()
		}
	}

	logger.Info().Str("report_id", finalReportID).Msg("using report ID")

	// Create and run pipeline
	orchestrator := pipeline.NewOrchestrator(*cfg, logger)
	result, err := orchestrator.RunPipeline(finalReportID)

	if err != nil {
		logger.Error().Err(err).Msg("pipeline execution failed")
		fmt.Fprintf(os.Stderr, "Pipeline failed: %s\n", err.Error())
		os.Exit(1)
	}

	if !result.Success {
		logger.Error().Str("error", result.Error).Msg("pipeline completed with errors")
		fmt.Fprintf(os.Stderr, "Pipeline failed: %s\n", result.Error)
		os.Exit(1)
	}

	// Success
	logger.Info().
		Str("report_id", result.ReportID).
		Msg("pipeline completed successfully")

	fmt.Printf("\n✓ Report generated successfully\n")
	fmt.Printf("  Report ID: %s\n", result.ReportID)
	fmt.Printf("  Report file: reports/%s.md\n", result.ReportID)
	fmt.Printf("  Summary backup: data/summaries/%s.json\n", result.ReportID)
}

// printHelp prints usage information
func printHelp() {
	fmt.Printf("GitHub Insights v%s\n\n", version)
	fmt.Println("Usage: github-insights [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  -config string")
	fmt.Println("        Path to configuration directory (default: config)")
	fmt.Println("  -report-id string")
	fmt.Println("        Custom report ID (default: auto-generated)")
	fmt.Println("  -weekly")
	fmt.Println("        Use week-based report ID format")
	fmt.Println("  -log-level string")
	fmt.Println("        Log level: debug, info, warn, error (default: info)")
	fmt.Println("  -version")
	fmt.Println("        Show version information")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println("\nEnvironment Variables:")
	fmt.Println("  GITHUB_TOKEN    GitHub API token (required)")
	fmt.Println("  LLM_API_KEY     LLM API key (optional, uses template if not set)")
	fmt.Println("\nExamples:")
	fmt.Println("  github-insights")
	fmt.Println("  github-insights -weekly")
	fmt.Println("  github-insights -config ./custom-config")
	fmt.Println("  github-insights -report-id 2024-02-week6")
	fmt.Println("  github-insights -log-level debug")
}

// generateDailyReportID generates a daily report ID (YYYY-MM-DD)
func generateDailyReportID() string {
	now := time.Now()
	return now.Format("2006-01-02")
}

// generateWeeklyReportID generates a weekly report ID (YYYY-MM-weekN)
func generateWeeklyReportID() string {
	now := time.Now()
	year, week := now.ISOWeek()
	return fmt.Sprintf("%d-%02d-week%d", year, int(now.Month()), week)
}
