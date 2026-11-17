package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"ai-repo-insights/internal/classifier"
	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/fetcher"
	"ai-repo-insights/internal/githubapi"
	"ai-repo-insights/internal/history"
	"ai-repo-insights/internal/llm"
	"ai-repo-insights/internal/logging"
	"ai-repo-insights/internal/pipeline"
)

const version = "2.0.0"

func main() {
	configDir := flag.String("config", "config", "Path to configuration directory")
	outDir := flag.String("out", "data/weekly", "Directory the weekly issue JSON is written to")
	historyPath := flag.String("history", "data/history.json", "Path to the repository history file")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ai-repo-insights v%s\n", version)
		return
	}

	logger := logging.NewLogger(*logLevel)

	cfg, err := config.Load(*configDir)
	if err != nil {
		fail("load configuration: %v", err)
	}
	if errs := cfg.Validate(); len(errs) > 0 {
		fail("invalid configuration: %v", errs)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	llmAPIKey := os.Getenv("LLM_API_KEY")
	if githubToken == "" || llmAPIKey == "" {
		fail("GITHUB_TOKEN and LLM_API_KEY must both be set: the radar reads repository details from GitHub and has no text without the LLM")
	}

	repoClassifier := classifier.New(cfg.Keywords)
	deps := pipeline.Deps{
		Trending: fetcher.New(cfg.Languages, logger),
		Classify: repoClassifier.Classify,
		GitHub:   githubapi.NewClient("https://api.github.com", githubToken, &http.Client{Timeout: 30 * time.Second}),
		Copy:     llm.NewClient(cfg.LLM, llmAPIKey, logger),
		History:  history.NewManager(*historyPath, logger),
		OutDir:   *outDir,
		Now:      time.Now(),
	}

	path, err := pipeline.Run(context.Background(), *cfg, deps, logger)
	if err != nil {
		fail("radar run failed, nothing was published: %v", err)
	}
	fmt.Printf("Issue written: %s\n", path)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
