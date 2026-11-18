// Package rsspipeline orchestrates RSS weekly digest generation with injectable dependencies.
package rsspipeline

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/rssweekly"
)

const (
	// DefaultOPMLURL is the live subscription list fetched on every run.
	DefaultOPMLURL = "https://raw.githubusercontent.com/NoSugarCoffee/rss-feeds/main/feeds.opml"
	rssUserAgent   = "ai-repo-insights/rss-weekly"
	requestTimeout = 30 * time.Second
)

// Deps are injected boundaries for OPML, feed collection, summarization, and output.
type Deps struct {
	OPML        func(ctx context.Context) ([]rssweekly.Feed, []string, error)
	Collect     func(ctx context.Context, feeds []rssweekly.Feed, week rssweekly.Week, now time.Time) ([]rssweekly.Article, []rssweekly.FailedFeed)
	Summarize   func(articles []rssweekly.Article) (map[string]string, error)
	OutDir      string
	Now         time.Time
	WeekMode    rssweekly.WeekMode
	OPMLURL     string
	SummaryMode string
}

// Run fetches OPML, collects articles for the selected week, optionally summarizes them, and writes the digest.
func Run(ctx context.Context, deps Deps, logger zerolog.Logger) (string, error) {
	week, err := rssweekly.WeekFor(deps.Now, deps.WeekMode)
	if err != nil {
		return "", fmt.Errorf("select week: %w", err)
	}
	logger.Info().Str("week_id", week.ID).Str("week_mode", string(deps.WeekMode)).Msg("starting RSS weekly run")

	feeds, categories, err := deps.OPML(ctx)
	if err != nil {
		return "", fmt.Errorf("load OPML: %w", err)
	}

	articles, failures := deps.Collect(ctx, feeds, week, deps.Now)
	logger.Info().
		Int("feeds", len(feeds)).
		Int("articles", len(articles)).
		Int("failed_feeds", len(failures)).
		Msg("collected feed items")

	if len(articles) > 0 {
		summaries, err := deps.Summarize(articles)
		if err != nil {
			return "", fmt.Errorf("summarize articles: %w", err)
		}
		for i := range articles {
			if summary, ok := summaries[articles[i].URL]; ok {
				articles[i].Summary = summary
			}
		}
	}

	opmlURL := deps.OPMLURL
	if opmlURL == "" {
		opmlURL = DefaultOPMLURL
	}
	digest := rssweekly.BuildDigest(week, categories, articles, failures, deps.Now, opmlURL)
	digest.SummaryMode = deps.SummaryMode
	path, err := rssweekly.WriteDigest(deps.OutDir, digest)
	if err != nil {
		return "", fmt.Errorf("write digest: %w", err)
	}

	logger.Info().Str("digest", path).Int("groups", len(digest.Groups)).Msg("RSS weekly run completed")
	return path, nil
}

// FetchOPML downloads the OPML document using the production HTTP settings.
func FetchOPML(ctx context.Context, url string) (io.ReadCloser, error) {
	client := &http.Client{Timeout: requestTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create OPML request: %w", err)
	}
	req.Header.Set("User-Agent", rssUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch OPML %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("fetch OPML %s: HTTP %s", url, resp.Status)
	}
	return resp.Body, nil
}

// ParseOPMLFromURL fetches and parses feeds from url.
func ParseOPMLFromURL(ctx context.Context, url string) ([]rssweekly.Feed, []string, error) {
	body, err := FetchOPML(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	defer body.Close()
	return rssweekly.ParseOPML(body)
}

// CollectFeeds uses production HTTP settings via rssweekly.Collect.
func CollectFeeds(ctx context.Context, feeds []rssweekly.Feed, week rssweekly.Week, now time.Time) ([]rssweekly.Article, []rssweekly.FailedFeed) {
	return rssweekly.Collect(ctx, feeds, week, now, nil)
}
