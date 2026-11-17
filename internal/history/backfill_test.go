package history

import (
	"os"
	"path/filepath"
	"testing"

	"ai-repo-insights/internal/models"
)

func TestBackfillEarliestWins(t *testing.T) {
	hist := models.NewHistory()
	hist.History["openclaw/openclaw"] = models.RepoHistory{WeeksInTop: 1, LastSeenReport: "2026-11-week47", LastSeenDate: "2026-11-16", FirstSeenReport: "2026-11-week47", FirstSeenDate: "2026-11-16"}

	added, err := Backfill(hist, "testdata")
	if err != nil {
		t.Fatal(err)
	}
	if added != 4 {
		t.Fatalf("added %d, want 4", added)
	}
	if len(hist.History) != 5 {
		t.Fatalf("tracked %d, want 5: %v", len(hist.History), hist.History)
	}
	got := hist.History["openclaw/openclaw"]
	if got.FirstSeenReport != "2026-02-week6" || got.FirstSeenDate != "2026-02-07" || got.LastSeenReport != "2026-11-week47" {
		t.Fatalf("openclaw %#v", got)
	}
	shannon := hist.History["KeygraphHQ/shannon"]
	if shannon.FirstSeenReport != "2026-03-week10" || shannon.LastSeenReport != "2026-03-week10" || shannon.LastSeenDate != "2026-03-02" {
		t.Fatalf("shannon %#v", shannon)
	}
	if _, ok := hist.History["category/list-link"]; ok {
		t.Fatal("parsed a non-table link")
	}
}

func TestBackfillRequiresReportDate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "2026-04-week15.md"), []byte("| 1 | [x](https://github.com/a/x) |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Backfill(models.NewHistory(), dir)
	if err == nil {
		t.Fatal("expected an error for a report without Report Date")
	}
}
