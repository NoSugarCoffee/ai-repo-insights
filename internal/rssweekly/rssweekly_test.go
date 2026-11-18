package rssweekly

import (
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
