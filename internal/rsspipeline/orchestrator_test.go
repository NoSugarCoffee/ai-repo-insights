package rsspipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/rssweekly"
)

var runNow = time.Date(2026, time.November, 16, 8, 0, 0, 0, time.UTC)

func TestRunEmptyWeekSkipsSummariesAndWritesFile(t *testing.T) {
	dir := t.TempDir()
	summarizer := &recordingSummarizer{t: t}
	deps := Deps{
		OPML: func(context.Context) ([]rssweekly.Feed, []string, error) {
			return []rssweekly.Feed{
				{Category: "Engineering", Source: "Go Blog", XMLURL: "https://example.com/go.xml"},
			}, []string{"Engineering", "Empty"}, nil
		},
		Collect: func(context.Context, []rssweekly.Feed, rssweekly.Week, time.Time) ([]rssweekly.Article, []rssweekly.FailedFeed) {
			return nil, nil
		},
		Summarize: summarizer.Summarize,
		OutDir:    dir,
		Now:       runNow,
		WeekMode:  rssweekly.WeekPrevious,
	}

	path, err := Run(context.Background(), deps, zerolog.Nop())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if summarizer.calls != 0 {
		t.Fatalf("Run() summarizer calls = %d, want 0 for empty week", summarizer.calls)
	}
	if path != filepath.Join(dir, "2026-W46.json") {
		t.Errorf("Run() path = %q, want weekly JSON path", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var digest rssweekly.Digest
	if err := json.Unmarshal(data, &digest); err != nil {
		t.Fatalf("digest JSON: %v", err)
	}
	if digest.ID != "2026-W46" || digest.Schema != rssweekly.SchemaVersion {
		t.Errorf("digest identity = %+v", digest)
	}
	if len(digest.Groups) != 2 || digest.Groups[0].Category != "Engineering" || digest.Groups[1].Category != "Empty" {
		t.Fatalf("digest groups = %#v", digest.Groups)
	}
	for _, g := range digest.Groups {
		if g.Items == nil || len(g.Items) != 0 {
			t.Errorf("group %q items = %#v, want empty array", g.Category, g.Items)
		}
	}
	if digest.FailedFeeds == nil || len(digest.FailedFeeds) != 0 {
		t.Errorf("failed_feeds = %#v, want empty array", digest.FailedFeeds)
	}
}

func TestRunRetainsFailedFeedsAndContinues(t *testing.T) {
	dir := t.TempDir()
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	deps := Deps{
		OPML: func(context.Context) ([]rssweekly.Feed, []string, error) {
			return []rssweekly.Feed{
				{Category: "Engineering", Source: "Good", XMLURL: "https://example.com/good.xml"},
				{Category: "Ops", Source: "Broken", XMLURL: "https://example.com/broken.xml"},
			}, []string{"Engineering", "Ops"}, nil
		},
		Collect: func(context.Context, []rssweekly.Feed, rssweekly.Week, time.Time) ([]rssweekly.Article, []rssweekly.FailedFeed) {
			return []rssweekly.Article{
					{
						Title:       "Fresh",
						URL:         "https://example.com/fresh",
						Source:      "Good",
						Category:    "Engineering",
						PublishedAt: time.Date(2026, 11, 15, 12, 0, 0, 0, shanghai),
						Description: "body",
					},
				}, []rssweekly.FailedFeed{{
					Source: "Broken",
					XMLURL: "https://example.com/broken.xml",
					Error:  "503 unavailable",
				}}
		},
		Summarize: func(articles []rssweekly.Article) (map[string]string, error) {
			out := make(map[string]string, len(articles))
			for _, a := range articles {
				out[a.URL] = "中文摘要"
			}
			return out, nil
		},
		OutDir:   dir,
		Now:      runNow,
		WeekMode: rssweekly.WeekPrevious,
	}

	path, err := Run(context.Background(), deps, zerolog.Nop())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"failed_feeds": [`) {
		t.Fatalf("digest missing failed_feeds: %s", data)
	}
	var digest rssweekly.Digest
	if err := json.Unmarshal(data, &digest); err != nil {
		t.Fatal(err)
	}
	if len(digest.FailedFeeds) != 1 || digest.FailedFeeds[0].Source != "Broken" {
		t.Errorf("failed_feeds = %#v", digest.FailedFeeds)
	}
	if len(digest.Groups[0].Items) != 1 || digest.Groups[0].Items[0].Summary != "中文摘要" {
		t.Errorf("group items = %#v", digest.Groups[0].Items)
	}
	_ = path
}

func TestRunAppliesSummariesBeforeWrite(t *testing.T) {
	dir := t.TempDir()
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	var summarizedURLs []string
	deps := Deps{
		OPML: func(context.Context) ([]rssweekly.Feed, []string, error) {
			return []rssweekly.Feed{
				{Category: "Engineering", Source: "Feed", XMLURL: "https://example.com/feed.xml"},
			}, []string{"Engineering"}, nil
		},
		Collect: func(context.Context, []rssweekly.Feed, rssweekly.Week, time.Time) ([]rssweekly.Article, []rssweekly.FailedFeed) {
			return []rssweekly.Article{
				{
					Title:       "One",
					URL:         "https://example.com/one",
					Source:      "Feed",
					Category:    "Engineering",
					PublishedAt: time.Date(2026, 11, 10, 9, 0, 0, 0, shanghai),
				},
				{
					Title:       "Two",
					URL:         "https://example.com/two",
					Source:      "Feed",
					Category:    "Engineering",
					PublishedAt: time.Date(2026, 11, 11, 10, 0, 0, 0, shanghai),
				},
			}, nil
		},
		Summarize: func(articles []rssweekly.Article) (map[string]string, error) {
			summarizedURLs = make([]string, len(articles))
			out := make(map[string]string, len(articles))
			for i, a := range articles {
				summarizedURLs[i] = a.URL
				out[a.URL] = "摘要-" + a.Title
			}
			return out, nil
		},
		OutDir:   dir,
		Now:      runNow,
		WeekMode: rssweekly.WeekPrevious,
	}

	_, err := Run(context.Background(), deps, zerolog.Nop())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(summarizedURLs) != 2 {
		t.Fatalf("summarized URLs = %v, want two articles", summarizedURLs)
	}

	path := filepath.Join(dir, "2026-W46.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var digest rssweekly.Digest
	if err := json.Unmarshal(data, &digest); err != nil {
		t.Fatal(err)
	}
	if len(digest.Groups[0].Items) != 2 {
		t.Fatalf("items = %#v", digest.Groups[0].Items)
	}
	want := map[string]string{
		"https://example.com/one": "摘要-One",
		"https://example.com/two": "摘要-Two",
	}
	for _, item := range digest.Groups[0].Items {
		if want[item.URL] != item.Summary {
			t.Errorf("item %q summary = %q, want %q", item.URL, item.Summary, want[item.URL])
		}
	}
}

type recordingSummarizer struct {
	t     *testing.T
	calls int
}

func (r *recordingSummarizer) Summarize(articles []rssweekly.Article) (map[string]string, error) {
	r.calls++
	r.t.Fatal("Summarize must not run for an empty week")
	return nil, nil
}
