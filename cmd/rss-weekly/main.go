package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/llm"
	"ai-repo-insights/internal/logging"
	"ai-repo-insights/internal/rsspipeline"
	"ai-repo-insights/internal/rssweekly"
)

func main() {
	configDir := flag.String("config", "config", "Path to configuration directory")
	outDir := flag.String("out", "data/rss-weekly", "Directory the RSS weekly JSON is written to")
	opmlURL := flag.String("opml", rsspipeline.DefaultOPMLURL, "URL of the OPML subscription list")
	weekFlag := flag.String("week", "previous", "Week selection: previous (default) or current")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	logger := logging.NewLogger(*logLevel)

	weekMode, err := parseWeekMode(*weekFlag)
	if err != nil {
		fail("%v", err)
	}

	cfg, err := config.Load(*configDir)
	if err != nil {
		fail("load configuration: %v", err)
	}
	if errs := cfg.Validate(); len(errs) > 0 {
		fail("invalid configuration: %v", errs)
	}

	var llmClient *llm.Client
	summarize := func(articles []rssweekly.Article) (map[string]string, error) {
		if len(articles) == 0 {
			return map[string]string{}, nil
		}
		apiKey := os.Getenv("LLM_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("LLM_API_KEY must be set when articles need summaries")
		}
		if llmClient == nil {
			llmClient = llm.NewClient(cfg.LLM, apiKey, logger)
		}
		return llmClient.WriteRSSSummaries(articles)
	}

	deps := rsspipeline.Deps{
		OPML: func(ctx context.Context) ([]rssweekly.Feed, []string, error) {
			return rsspipeline.ParseOPMLFromURL(ctx, *opmlURL)
		},
		Collect:   rsspipeline.CollectFeeds,
		Summarize: summarize,
		OutDir:    *outDir,
		Now:       time.Now(),
		WeekMode:  weekMode,
		OPMLURL:   *opmlURL,
	}

	path, err := rsspipeline.Run(context.Background(), deps, logger)
	if err != nil {
		fail("RSS weekly run failed, nothing was published: %v", err)
	}
	fmt.Printf("Digest written: %s\n", path)
}

func parseWeekMode(raw string) (rssweekly.WeekMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "previous", "":
		return rssweekly.WeekPrevious, nil
	case "current":
		return rssweekly.WeekCurrent, nil
	default:
		return "", fmt.Errorf("unsupported -week %q: use previous or current", raw)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
