package rssweekly

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	SchemaVersion = "rss-weekly/1"
	shanghaiName  = "Asia/Shanghai"
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

	var walk func([]opmlOutline, string)
	walk = func(outlines []opmlOutline, inheritedCategory string) {
		for _, outline := range outlines {
			xmlURL := strings.TrimSpace(outline.XMLURL)
			name := outlineName(outline)
			category := inheritedCategory
			if xmlURL == "" {
				if name != "" {
					category = name
					categories = append(categories, category)
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
