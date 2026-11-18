package rssweekly

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

const (
	SchemaVersion      = "rss-weekly/1"
	shanghaiName       = "Asia/Shanghai"
	feedRequestTimeout = 30 * time.Second
	maxFeedErrorRunes  = 500
)

// Feed identifies one subscription and its nearest enclosing OPML category.
type Feed struct {
	Category string
	Source   string
	XMLURL   string
}

// Article is an item included in a weekly digest.
type Article struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Summary     string    `json:"summary"`
	Description string    `json:"-"`
	Category    string    `json:"-"`
}

// Group contains the articles for one OPML category.
type Group struct {
	Category string    `json:"category"`
	Items    []Article `json:"items"`
}

// FailedFeed records a subscription that could not be collected.
type FailedFeed struct {
	Source string `json:"source"`
	XMLURL string `json:"xml_url"`
	Error  string `json:"error"`
}

// Digest is the serialized weekly RSS publication.
type Digest struct {
	Schema      string       `json:"schema"`
	ID          string       `json:"id"`
	GeneratedAt time.Time    `json:"generated_at"`
	WeekStart   time.Time    `json:"week_start"`
	WeekEnd     time.Time    `json:"week_end"`
	Groups      []Group      `json:"groups"`
	FailedFeeds []FailedFeed `json:"failed_feeds"`
	SummaryMode string       `json:"summary_mode,omitempty"`
}

// WeekMode selects either the current Shanghai week or the preceding week.
type WeekMode string

const (
	WeekCurrent  WeekMode = "current"
	WeekPrevious WeekMode = "previous"
)

// Week describes a half-open Monday-to-Monday interval in Asia/Shanghai.
type Week struct {
	ID    string
	Start time.Time
	End   time.Time
}

// FeedParser is the part of gofeed.Parser needed by Collect.
type FeedParser interface {
	ParseURLWithContext(string, context.Context) (*gofeed.Feed, error)
}

type opmlDocument struct {
	Body opmlBody `xml:"body"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr"`
	XMLURL   string        `xml:"xmlUrl,attr"`
	Outlines []opmlOutline `xml:"outline"`
}

// ParseOPML extracts feeds and categories in document order.
func ParseOPML(r io.Reader) ([]Feed, []string, error) {
	var document opmlDocument
	if err := xml.NewDecoder(r).Decode(&document); err != nil {
		return nil, nil, fmt.Errorf("parse OPML: %w", err)
	}

	var feeds []Feed
	var categories []string
	seenCategories := make(map[string]struct{})

	var walk func([]opmlOutline, string)
	walk = func(outlines []opmlOutline, inheritedCategory string) {
		for _, outline := range outlines {
			xmlURL := strings.TrimSpace(outline.XMLURL)
			name := outlineName(outline)
			category := inheritedCategory
			if xmlURL == "" {
				if name != "" {
					category = name
					if _, seen := seenCategories[category]; !seen {
						categories = append(categories, category)
						seenCategories[category] = struct{}{}
					}
				}
			} else {
				feeds = append(feeds, Feed{
					Category: category,
					Source:   name,
					XMLURL:   xmlURL,
				})
			}
			walk(outline.Outlines, category)
		}
	}
	walk(document.Body.Outlines, "")

	return feeds, categories, nil
}

func outlineName(outline opmlOutline) string {
	if text := strings.TrimSpace(outline.Text); text != "" {
		return text
	}
	return strings.TrimSpace(outline.Title)
}

// WeekFor returns the selected Monday-to-Monday Shanghai week.
func WeekFor(now time.Time, mode WeekMode) (Week, error) {
	shanghai, err := time.LoadLocation(shanghaiName)
	if err != nil {
		return Week{}, fmt.Errorf("load %s timezone: %w", shanghaiName, err)
	}

	local := now.In(shanghai)
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, shanghai).
		AddDate(0, 0, -daysSinceMonday)

	switch mode {
	case WeekCurrent:
	case WeekPrevious:
		start = start.AddDate(0, 0, -7)
	default:
		return Week{}, fmt.Errorf("unsupported week mode %q", mode)
	}

	end := start.AddDate(0, 0, 7)
	isoYear, isoWeek := start.ISOWeek()
	return Week{
		ID:    fmt.Sprintf("%04d-W%02d", isoYear, isoWeek),
		Start: start,
		End:   end,
	}, nil
}

// Collect fetches feeds in order and retains items in the selected week.
// A nil parser uses a gofeed parser with the production HTTP settings.
func Collect(ctx context.Context, feeds []Feed, week Week, parser FeedParser) ([]Article, []FailedFeed) {
	if parser == nil {
		defaultParser := gofeed.NewParser()
		defaultParser.Client = &http.Client{Timeout: feedRequestTimeout}
		defaultParser.UserAgent = "ai-repo-insights/rss-weekly"
		parser = defaultParser
	}

	shanghai, err := time.LoadLocation(shanghaiName)
	if err != nil {
		return nil, []FailedFeed{{Error: boundedError(err)}}
	}

	articles := make([]Article, 0)
	failures := make([]FailedFeed, 0)
	seenURLs := make(map[string]struct{})
	for _, subscription := range feeds {
		parsed, err := parser.ParseURLWithContext(subscription.XMLURL, ctx)
		if err != nil {
			failures = append(failures, FailedFeed{
				Source: subscription.Source,
				XMLURL: subscription.XMLURL,
				Error:  boundedError(err),
			})
			continue
		}
		for _, item := range parsed.Items {
			url := strings.TrimSpace(item.Link)
			publishedAt := item.PublishedParsed
			if publishedAt == nil {
				publishedAt = item.UpdatedParsed
			}
			if url == "" || publishedAt == nil || publishedAt.Before(week.Start) || !publishedAt.Before(week.End) {
				continue
			}
			if _, exists := seenURLs[url]; exists {
				continue
			}
			seenURLs[url] = struct{}{}

			description := item.Description
			if strings.TrimSpace(description) == "" {
				description = item.Content
			}
			articles = append(articles, Article{
				Title:       strings.TrimSpace(item.Title),
				URL:         url,
				Source:      subscription.Source,
				PublishedAt: publishedAt.In(shanghai),
				Description: description,
				Category:    subscription.Category,
			})
		}
	}
	return articles, failures
}

// BuildDigest groups collected articles in logical category order.
func BuildDigest(week Week, categories []string, articles []Article, failures []FailedFeed, generatedAt time.Time) Digest {
	shanghai, err := time.LoadLocation(shanghaiName)
	if err != nil {
		shanghai = time.FixedZone(shanghaiName, 8*60*60)
	}

	groups := make([]Group, 0, len(categories))
	groupIndexes := make(map[string]int, len(categories))
	addGroup := func(category string) int {
		if index, exists := groupIndexes[category]; exists {
			return index
		}
		groupIndexes[category] = len(groups)
		groups = append(groups, Group{Category: category, Items: make([]Article, 0)})
		return len(groups) - 1
	}
	for _, category := range categories {
		addGroup(category)
	}
	for _, article := range articles {
		index := addGroup(article.Category)
		groups[index].Items = append(groups[index].Items, article)
	}
	for i := range groups {
		sort.SliceStable(groups[i].Items, func(a, b int) bool {
			return groups[i].Items[a].PublishedAt.After(groups[i].Items[b].PublishedAt)
		})
	}
	if failures == nil {
		failures = make([]FailedFeed, 0)
	}

	return Digest{
		Schema:      SchemaVersion,
		ID:          week.ID,
		GeneratedAt: generatedAt.In(shanghai),
		WeekStart:   week.Start.In(shanghai),
		WeekEnd:     week.End.In(shanghai),
		Groups:      groups,
		FailedFeeds: failures,
	}
}

// WriteDigest writes digest as <dir>/<ID>.json without replacing an existing file.
func WriteDigest(dir string, digest Digest) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create digest dir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(digest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal digest %s: %w", digest.ID, err)
	}

	path := filepath.Join(dir, digest.ID+".json")
	file, err := os.CreateTemp(dir, "."+digest.ID+"-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary digest for %s: %w", path, err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	closeWith := func(cause error) error {
		if closeErr := file.Close(); closeErr != nil {
			return errors.Join(cause, fmt.Errorf("close temporary digest for %s: %w", path, closeErr))
		}
		return cause
	}
	if err := file.Chmod(0o644); err != nil {
		return "", closeWith(fmt.Errorf("set permissions on temporary digest for %s: %w", path, err))
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return "", closeWith(fmt.Errorf("write digest %s: %w", path, err))
	}
	if err := file.Sync(); err != nil {
		return "", closeWith(fmt.Errorf("sync digest %s: %w", path, err))
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close digest %s: %w", path, err)
	}
	if err := os.Link(tempPath, path); errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("digest %s already exists", path)
	} else if err != nil {
		return "", fmt.Errorf("publish digest %s: %w", path, err)
	}
	return path, nil
}

func boundedError(err error) string {
	runes := []rune(err.Error())
	if len(runes) <= maxFeedErrorRunes {
		return string(runes)
	}
	return string(runes[:maxFeedErrorRunes])
}
