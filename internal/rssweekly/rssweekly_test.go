package rssweekly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseOPML(t *testing.T) {
	tests := []struct {
		name           string
		opml           string
		wantFeeds      []Feed
		wantCategories []string
	}{
		{
			name: "nested outlines preserve feed and category order",
			opml: `<?xml version="1.0"?>
<opml version="2.0">
  <body>
    <outline text="Engineering">
      <outline text="Go Blog" xmlUrl="https://go.dev/blog/feed.atom"/>
      <outline text="Nested folder">
        <outline title="Rust Blog" xmlUrl="https://blog.rust-lang.org/feed.xml"/>
      </outline>
    </outline>
    <outline text="Security">
      <outline title="Krebs" xmlUrl="https://krebsonsecurity.com/feed/"/>
    </outline>
  </body>
</opml>`,
			wantFeeds: []Feed{
				{Category: "Engineering", Source: "Go Blog", XMLURL: "https://go.dev/blog/feed.atom"},
				{Category: "Nested folder", Source: "Rust Blog", XMLURL: "https://blog.rust-lang.org/feed.xml"},
				{Category: "Security", Source: "Krebs", XMLURL: "https://krebsonsecurity.com/feed/"},
			},
			wantCategories: []string{"Engineering", "Nested folder", "Security"},
		},
		{
			name: "category falls back to title",
			opml: `<opml version="2.0"><body>
  <outline title="Data">
    <outline text="Database Weekly" xmlUrl="https://example.com/database.xml"/>
  </outline>
</body></opml>`,
			wantFeeds: []Feed{
				{Category: "Data", Source: "Database Weekly", XMLURL: "https://example.com/database.xml"},
			},
			wantCategories: []string{"Data"},
		},
		{
			name: "repeated category names form one group in first occurrence order",
			opml: `<opml version="2.0"><body>
  <outline text="Engineering">
    <outline text="Go Blog" xmlUrl="https://go.dev/blog/feed.atom"/>
  </outline>
  <outline text="Security">
    <outline text="Krebs" xmlUrl="https://krebsonsecurity.com/feed/"/>
  </outline>
  <outline text="Engineering">
    <outline text="Rust Blog" xmlUrl="https://blog.rust-lang.org/feed.xml"/>
  </outline>
</body></opml>`,
			wantFeeds: []Feed{
				{Category: "Engineering", Source: "Go Blog", XMLURL: "https://go.dev/blog/feed.atom"},
				{Category: "Security", Source: "Krebs", XMLURL: "https://krebsonsecurity.com/feed/"},
				{Category: "Engineering", Source: "Rust Blog", XMLURL: "https://blog.rust-lang.org/feed.xml"},
			},
			wantCategories: []string{"Engineering", "Security"},
		},
		{
			name: "unnamed wrappers retain nearest named category",
			opml: `<opml version="2.0"><body>
  <outline text="Engineering">
    <outline>
      <outline text="Go Blog" xmlUrl="https://go.dev/blog/feed.atom"/>
    </outline>
  </outline>
</body></opml>`,
			wantFeeds: []Feed{
				{Category: "Engineering", Source: "Go Blog", XMLURL: "https://go.dev/blog/feed.atom"},
			},
			wantCategories: []string{"Engineering"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFeeds, gotCategories, err := ParseOPML(strings.NewReader(tt.opml))
			if err != nil {
				t.Fatalf("ParseOPML() error = %v", err)
			}
			if !equalFeeds(gotFeeds, tt.wantFeeds) {
				t.Errorf("ParseOPML() feeds = %#v, want %#v", gotFeeds, tt.wantFeeds)
			}
			if !equalStrings(gotCategories, tt.wantCategories) {
				t.Errorf("ParseOPML() categories = %#v, want %#v", gotCategories, tt.wantCategories)
			}
		})
	}
}

func TestWeekFor(t *testing.T) {
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	tests := []struct {
		name string
		now  time.Time
		mode WeekMode
		want Week
	}{
		{
			name: "ISO year boundary",
			now:  time.Date(2027, time.January, 1, 12, 0, 0, 0, shanghai),
			mode: WeekCurrent,
			want: Week{
				ID:    "2026-W53",
				Start: time.Date(2026, time.December, 28, 0, 0, 0, 0, shanghai),
				End:   time.Date(2027, time.January, 4, 0, 0, 0, 0, shanghai),
			},
		},
		{
			name: "previous mode from Shanghai Monday",
			now:  time.Date(2026, time.November, 15, 16, 0, 0, 0, time.UTC),
			mode: WeekPrevious,
			want: Week{
				ID:    "2026-W46",
				Start: time.Date(2026, time.November, 9, 0, 0, 0, 0, shanghai),
				End:   time.Date(2026, time.November, 16, 0, 0, 0, 0, shanghai),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WeekFor(tt.now, tt.mode)
			if err != nil {
				t.Fatalf("WeekFor() error = %v", err)
			}
			if got.ID != tt.want.ID || !got.Start.Equal(tt.want.Start) || !got.End.Equal(tt.want.End) {
				t.Errorf("WeekFor() = %#v, want %#v", got, tt.want)
			}
			if got.Start.Location().String() != "Asia/Shanghai" || got.End.Location().String() != "Asia/Shanghai" {
				t.Errorf("WeekFor() locations = %q, %q; want Asia/Shanghai", got.Start.Location(), got.End.Location())
			}
		})
	}
}

func TestCollectFiltersDeduplicatesAndContinuesAfterFeedFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/engineering.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Engineering</title>
  <item><title>Older</title><link>https://example.com/older</link><pubDate>Tue, 10 Nov 2026 01:00:00 GMT</pubDate><description>older description</description></item>
  <item><title>Newest</title><link>https://example.com/newest</link><pubDate>Sun, 15 Nov 2026 10:00:00 GMT</pubDate><description>newest description</description></item>
  <item><title>Before start</title><link>https://example.com/before</link><pubDate>Sun, 08 Nov 2026 15:59:59 GMT</pubDate></item>
  <item><title>At end</title><link>https://example.com/end</link><pubDate>Sun, 15 Nov 2026 16:00:00 GMT</pubDate></item>
  <item><title>No date</title><link>https://example.com/no-date</link></item>
  <item><title>No URL</title><pubDate>Tue, 10 Nov 2026 02:00:00 GMT</pubDate></item>
  <item><title>First duplicate</title><link> https://example.com/shared </link><pubDate>Wed, 11 Nov 2026 03:00:00 GMT</pubDate></item>
</channel></rss>`))
	})
	mux.HandleFunc("/security.atom", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Security</title>
  <entry><title>Updated fallback</title><link href="https://example.com/updated"/><updated>2026-11-12T10:00:00Z</updated><summary>updated summary</summary></entry>
  <entry><title>Later duplicate</title><link href="https://example.com/shared"/><published>2026-11-13T10:00:00Z</published></entry>
</feed>`))
	})
	mux.HandleFunc("/broken.xml", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	week := Week{
		ID:    "2026-W46",
		Start: time.Date(2026, time.November, 9, 0, 0, 0, 0, shanghai),
		End:   time.Date(2026, time.November, 16, 0, 0, 0, 0, shanghai),
	}
	feeds := []Feed{
		{Category: "Engineering", Source: "Engineering Feed", XMLURL: server.URL + "/engineering.xml"},
		{Category: "Security", Source: "Security Feed", XMLURL: server.URL + "/security.atom"},
		{Category: "Operations", Source: "Broken Feed", XMLURL: server.URL + "/broken.xml"},
	}

	articles, failures := Collect(context.Background(), feeds, week, nil)

	if len(failures) != 1 {
		t.Fatalf("Collect() failures = %#v, want one", failures)
	}
	if failures[0].Source != "Broken Feed" || failures[0].XMLURL != server.URL+"/broken.xml" || failures[0].Error == "" {
		t.Errorf("Collect() failure = %#v, want populated Broken Feed failure", failures[0])
	}
	wantTitles := []string{"Older", "Newest", "First duplicate", "Updated fallback"}
	if len(articles) != len(wantTitles) {
		t.Fatalf("Collect() articles = %#v, want titles %v", articles, wantTitles)
	}
	for i, wantTitle := range wantTitles {
		if articles[i].Title != wantTitle {
			t.Errorf("Collect() article %d title = %q, want %q", i, articles[i].Title, wantTitle)
		}
		if articles[i].PublishedAt.Location().String() != "Asia/Shanghai" {
			t.Errorf("Collect() article %q location = %q, want Asia/Shanghai", articles[i].Title, articles[i].PublishedAt.Location())
		}
	}
	if articles[2].URL != "https://example.com/shared" || articles[2].Source != "Engineering Feed" {
		t.Errorf("Collect() duplicate owner = %#v, want first feed with trimmed URL", articles[2])
	}
	if articles[3].PublishedAt.Hour() != 18 || articles[3].Description != "updated summary" {
		t.Errorf("Collect() updated fallback article = %#v, want Shanghai updated time and summary", articles[3])
	}
}

func TestBuildDigestPreservesCategoriesAndSortsItemsDescending(t *testing.T) {
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	week := Week{
		ID:    "2026-W46",
		Start: time.Date(2026, time.November, 9, 0, 0, 0, 0, shanghai),
		End:   time.Date(2026, time.November, 16, 0, 0, 0, 0, shanghai),
	}
	articles := []Article{
		{Category: "Engineering", Title: "Older", URL: "https://example.com/older", PublishedAt: time.Date(2026, 11, 10, 9, 0, 0, 0, shanghai)},
		{Category: "Engineering", Title: "Newer", URL: "https://example.com/newer", PublishedAt: time.Date(2026, 11, 15, 18, 0, 0, 0, shanghai)},
	}
	generatedAt := time.Date(2026, 11, 16, 8, 30, 0, 0, time.UTC)

	digest := BuildDigest(week, []string{"Engineering", "Empty", "Operations"}, articles, []FailedFeed{{
		Source: "Broken Feed", XMLURL: "https://example.com/broken.xml", Error: "unavailable",
	}}, generatedAt, "https://raw.githubusercontent.com/NoSugarCoffee/rss-feeds/main/feeds.opml")

	if digest.Schema != SchemaVersion || digest.ID != week.ID {
		t.Errorf("BuildDigest() identity = %q, %q, want %q, %q", digest.Schema, digest.ID, SchemaVersion, week.ID)
	}
	if digest.WeekStart != "2026-11-09" || digest.WeekEnd != "2026-11-15" {
		t.Errorf("BuildDigest() week bounds = %q..%q, want 2026-11-09..2026-11-15", digest.WeekStart, digest.WeekEnd)
	}
	if digest.OPML != "https://raw.githubusercontent.com/NoSugarCoffee/rss-feeds/main/feeds.opml" {
		t.Errorf("BuildDigest() opml = %q", digest.OPML)
	}
	if digest.GeneratedAt.Location().String() != "Asia/Shanghai" || digest.GeneratedAt.Hour() != 16 {
		t.Errorf("BuildDigest() generated_at = %v, want Shanghai-normalized time", digest.GeneratedAt)
	}
	if len(digest.Groups) != 3 {
		t.Fatalf("BuildDigest() groups = %#v, want three ordered groups", digest.Groups)
	}
	for i, category := range []string{"Engineering", "Empty", "Operations"} {
		if digest.Groups[i].Category != category {
			t.Errorf("BuildDigest() group %d = %q, want %q", i, digest.Groups[i].Category, category)
		}
		if digest.Groups[i].Items == nil {
			t.Errorf("BuildDigest() group %q items is nil, want empty JSON array", category)
		}
	}
	if got := []string{digest.Groups[0].Items[0].Title, digest.Groups[0].Items[1].Title}; !equalStrings(got, []string{"Newer", "Older"}) {
		t.Errorf("BuildDigest() sorted titles = %v, want [Newer Older]", got)
	}
	if digest.FailedFeeds == nil || len(digest.FailedFeeds) != 1 {
		t.Errorf("BuildDigest() failed_feeds = %#v, want non-nil one-item slice", digest.FailedFeeds)
	}
}

func TestWriteDigestWritesContractAndRefusesReplacement(t *testing.T) {
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	digest := Digest{
		Schema:      SchemaVersion,
		ID:          "2026-W46",
		WeekStart:   "2026-11-09",
		WeekEnd:     "2026-11-15",
		GeneratedAt: time.Date(2026, 11, 16, 8, 0, 0, 0, shanghai),
		OPML:        "https://raw.githubusercontent.com/NoSugarCoffee/rss-feeds/main/feeds.opml",
		Groups:      []Group{{Category: "Empty", Items: []Article{}}},
		FailedFeeds: []FailedFeed{},
	}
	dir := filepath.Join(t.TempDir(), "nested")

	path, err := WriteDigest(dir, digest)
	if err != nil {
		t.Fatalf("WriteDigest() error = %v", err)
	}
	if path != filepath.Join(dir, "2026-W46.json") {
		t.Errorf("WriteDigest() path = %q, want weekly JSON path", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("written digest is invalid JSON: %v", err)
	}
	wantKeys := []string{"schema", "id", "week_start", "week_end", "generated_at", "opml", "groups", "failed_feeds"}
	if len(raw) != len(wantKeys) {
		t.Errorf("written keys = %v, want exactly %v", mapKeys(raw), wantKeys)
	}
	for _, key := range wantKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("written digest missing key %q", key)
		}
	}
	if !strings.Contains(string(data), `"week_start": "2026-11-09"`) || !strings.Contains(string(data), `"week_end": "2026-11-15"`) {
		t.Errorf("written digest week bounds must be YYYY-MM-DD dates:\n%s", data)
	}
	if !strings.Contains(string(data), `"items": []`) || !strings.Contains(string(data), `"failed_feeds": []`) {
		t.Errorf("written digest must preserve empty arrays:\n%s", data)
	}

	if _, err := WriteDigest(dir, digest); err == nil {
		t.Fatal("WriteDigest() replacement error = nil, want refusal")
	}
}

func mapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func equalFeeds(got, want []Feed) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestDigestJSONUsesContractFieldNames(t *testing.T) {
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	digest := Digest{
		Schema:      SchemaVersion,
		ID:          "2026-W47",
		WeekStart:   "2026-11-16",
		WeekEnd:     "2026-11-22",
		GeneratedAt: time.Date(2026, 11, 18, 12, 0, 0, 0, shanghai),
		OPML:        "https://raw.githubusercontent.com/NoSugarCoffee/rss-feeds/main/feeds.opml",
		Groups: []Group{{
			Category: "Blogs",
			Items: []Article{{
				Title:       "Example",
				URL:         "https://example.com/post",
				Source:      "Example Blog",
				PublishedAt: time.Date(2026, 11, 17, 9, 0, 0, 0, shanghai),
				Summary:     "一句中文要点",
			}},
		}},
		FailedFeeds: []FailedFeed{{
			Source: "Broken",
			XMLURL: "https://example.com/broken.xml",
			Error:  "timeout",
		}},
	}
	data, err := json.Marshal(digest)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(data)
	for _, want := range []string{`"summary_zh":"一句中文要点"`, `"xmlUrl":"https://example.com/broken.xml"`, `"published_at":`, `"opml":`} {
		if !strings.Contains(raw, want) {
			t.Errorf("digest JSON missing %s in %s", want, raw)
		}
	}
	if strings.Contains(raw, `"summary":`) || strings.Contains(raw, `"xml_url":`) {
		t.Errorf("digest JSON used unexpected field names: %s", raw)
	}
}
