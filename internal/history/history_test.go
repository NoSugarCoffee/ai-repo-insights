package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/models"
)

func newManager(t *testing.T, seed *models.History) *Manager {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.json")
	if seed != nil {
		data, err := json.Marshal(seed)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return NewManager(path, zerolog.Nop())
}

func seedWith(key string, h models.RepoHistory) *models.History {
	return &models.History{LatestReport: h.LastSeenReport, History: map[string]models.RepoHistory{key: h}}
}

func TestLoadHistory_NewFile(t *testing.T) {
	hist, err := newManager(t, nil).LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if hist.History == nil || len(hist.History) != 0 {
		t.Fatalf("expected empty initialized history, got %#v", hist.History)
	}
}

func TestUpdateHistory_NewRepo(t *testing.T) {
	hist, err := newManager(t, nil).UpdateHistory([]string{"a/x"}, "2026-W47", "2026-11-16")
	if err != nil {
		t.Fatal(err)
	}
	got := hist.History["a/x"]
	want := models.RepoHistory{WeeksInTop: 1, LastSeenReport: "2026-W47", LastSeenDate: "2026-11-16", FirstSeenReport: "2026-W47", FirstSeenDate: "2026-11-16"}
	if got != want {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if hist.LatestReport != "2026-W47" {
		t.Fatalf("latest report %q", hist.LatestReport)
	}
}

func TestUpdateHistory_ExistingRepoKeepsFirstSeen(t *testing.T) {
	seed := seedWith("a/x", models.RepoHistory{WeeksInTop: 2, LastSeenReport: "2026-W46", LastSeenDate: "2026-11-09", FirstSeenReport: "2026-W45", FirstSeenDate: "2026-11-02"})
	hist, err := newManager(t, seed).UpdateHistory([]string{"a/x"}, "2026-W47", "2026-11-16")
	if err != nil {
		t.Fatal(err)
	}
	got := hist.History["a/x"]
	want := models.RepoHistory{WeeksInTop: 3, LastSeenReport: "2026-W47", LastSeenDate: "2026-11-16", FirstSeenReport: "2026-W45", FirstSeenDate: "2026-11-02"}
	if got != want {
		t.Fatalf("got %#v", got)
	}
}

func TestUpdateHistoryKeepsReposThatLeftTheList(t *testing.T) {
	seed := seedWith("a/x", models.RepoHistory{WeeksInTop: 1, LastSeenReport: "2026-W46", LastSeenDate: "2026-11-09", FirstSeenReport: "2026-W46", FirstSeenDate: "2026-11-09"})
	hist, err := newManager(t, seed).UpdateHistory([]string{"b/y"}, "2026-W47", "2026-11-16")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hist.History["b/y"]; !ok {
		t.Fatal("b/y missing")
	}
	ax, ok := hist.History["a/x"]
	if !ok {
		t.Fatal("a/x was removed")
	}
	if ax.LastSeenReport != "2026-W46" {
		t.Fatalf("a/x LastSeenReport changed to %q", ax.LastSeenReport)
	}
}

func TestHistorySeen(t *testing.T) {
	h := seedWith("a/x", models.RepoHistory{WeeksInTop: 1})
	if !h.Seen("a/x") {
		t.Fatal("a/x should be seen")
	}
	if !h.Seen("A/X") {
		t.Fatal("repository lookup should be case-insensitive")
	}
	if h.Seen("c/z") {
		t.Fatal("c/z should not be seen")
	}
}

func TestSaveHistoryRoundTrip(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "nested", "dir", "history.json"), zerolog.Nop())
	hist, err := m.UpdateHistory([]string{"a/x", "b/y"}, "2026-W47", "2026-11-16")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SaveHistory(hist); err != nil {
		t.Fatal(err)
	}
	loaded, err := m.LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.History) != 2 || loaded.LatestReport != "2026-W47" {
		t.Fatalf("got %#v", loaded)
	}
	for _, key := range []string{"a/x", "b/y"} {
		want := models.RepoHistory{WeeksInTop: 1, LastSeenReport: "2026-W47", LastSeenDate: "2026-11-16", FirstSeenReport: "2026-W47", FirstSeenDate: "2026-11-16"}
		if got := loaded.History[key]; got != want {
			t.Fatalf("%s: got %#v want %#v", key, got, want)
		}
	}
}

func TestLoadHistoryNormalizesAndMergesRepositoryKeys(t *testing.T) {
	seed := &models.History{History: map[string]models.RepoHistory{
		"ruvnet/ruvector": {WeeksInTop: 3, FirstSeenReport: "2026-W08", FirstSeenDate: "2026-02-16", LastSeenReport: "2026-W11", LastSeenDate: "2026-03-09"},
		"ruvnet/RuVector": {WeeksInTop: 3, FirstSeenReport: "2026-W12", FirstSeenDate: "2026-03-16", LastSeenReport: "2026-W14", LastSeenDate: "2026-03-30"},
	}}
	hist, err := newManager(t, seed).LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(hist.History) != 1 {
		t.Fatalf("got keys %#v", hist.History)
	}
	want := models.RepoHistory{WeeksInTop: 6, FirstSeenReport: "2026-W08", FirstSeenDate: "2026-02-16", LastSeenReport: "2026-W14", LastSeenDate: "2026-03-30"}
	if got := hist.History["ruvnet/ruvector"]; got != want {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestUpdateHistoryNormalizesAndDeduplicatesKeys(t *testing.T) {
	hist, err := newManager(t, nil).UpdateHistory([]string{"Owner/Repo", "owner/repo"}, "2026-W47", "2026-11-16")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist.History) != 1 || hist.History["owner/repo"].WeeksInTop != 1 {
		t.Fatalf("got %#v", hist.History)
	}
}
