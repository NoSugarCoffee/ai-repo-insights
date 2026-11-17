package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/githubapi"
	"ai-repo-insights/internal/history"
	"ai-repo-insights/internal/models"
	"ai-repo-insights/internal/radar"
)

var runNow = time.Date(2026, 11, 16, 4, 46, 0, 0, time.UTC)

type fakeTrending []models.RepoMetadata

func (f fakeTrending) FetchTrending() ([]models.RepoMetadata, error) { return f, nil }

type fakeGitHub map[string]time.Time

func (f fakeGitHub) Repo(_ context.Context, name string) (githubapi.RepoDetails, error) {
	return githubapi.RepoDetails{CreatedAt: f[name], Stars: 10}, nil
}
func (f fakeGitHub) Readme(context.Context, string) (string, error) { return "readme", nil }

type fakeCopy struct {
	err   error
	calls int
	seen  int
	hook  func()
}

func (f *fakeCopy) WriteRadarCopy(cands []radar.Candidate, _ string) (radar.Copy, error) {
	f.calls++
	f.seen = len(cands)
	if f.hook != nil {
		f.hook()
	}
	if f.err != nil {
		return radar.Copy{}, f.err
	}
	comments := map[string]string{}
	for _, c := range cands {
		comments[c.Repo.Key()] = "点评"
	}
	return radar.Copy{Headline: "主线", Summary: "摘要", Comments: comments}, nil
}

func classifyAll(repos []models.RepoMetadata) []models.ClassifiedRepo {
	out := make([]models.ClassifiedRepo, 0, len(repos))
	for _, r := range repos {
		out = append(out, models.ClassifiedRepo{Metadata: r})
	}
	return out
}

type fixture struct {
	deps        Deps
	historyPath string
	outDir      string
	copy        *fakeCopy
}

func newFixture(t *testing.T, seenKeys ...string) fixture {
	t.Helper()
	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history.json")
	seed := models.NewHistory()
	for _, k := range seenKeys {
		seed.History[k] = models.RepoHistory{WeeksInTop: 1, LastSeenReport: "2026-W46", LastSeenDate: "2026-11-09", FirstSeenReport: "2026-W46", FirstSeenDate: "2026-11-09"}
	}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(historyPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cp := &fakeCopy{}
	return fixture{
		deps: Deps{
			Trending: fakeTrending{{Owner: "old", Name: "seen"}, {Owner: "new", Name: "fresh"}, {Owner: "old", Name: "returning"}},
			Classify: classifyAll,
			GitHub:   fakeGitHub{"new/fresh": runNow.AddDate(0, 0, -5), "old/returning": runNow.AddDate(0, 0, -400), "old/seen": runNow.AddDate(-2, 0, 0)},
			Copy:     cp,
			History:  history.NewManager(historyPath, zerolog.Nop()),
			OutDir:   filepath.Join(dir, "weekly"),
			Now:      runNow,
		},
		historyPath: historyPath,
		outDir:      filepath.Join(dir, "weekly"),
		copy:        cp,
	}
}

func testConfig() config.Config {
	return config.Config{Settings: config.Settings{NewRepoThresholdDays: 30}, WritingGuide: "guide"}
}

func TestRunWritesIssueAndUpdatesHistory(t *testing.T) {
	f := newFixture(t, "old/seen")
	path, err := Run(context.Background(), testConfig(), f.deps, zerolog.Nop())
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "2026-W47.json" {
		t.Fatalf("path %s", path)
	}
	data, _ := os.ReadFile(path)
	var issue radar.Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		t.Fatal(err)
	}
	if len(issue.NewProjects) != 1 || issue.NewProjects[0].ID != "new/fresh" || len(issue.NewEntries) != 1 || issue.NewEntries[0].ID != "old/returning" {
		t.Fatalf("issue %#v", issue)
	}
	hist, _ := f.deps.History.LoadHistory()
	for _, k := range []string{"new/fresh", "old/returning"} {
		if hist.History[k].FirstSeenReport != "2026-W47" {
			t.Errorf("%s first seen %q", k, hist.History[k].FirstSeenReport)
		}
	}
	if hist.History["old/seen"].LastSeenReport != "2026-W47" || hist.History["old/seen"].FirstSeenReport != "2026-W46" {
		t.Errorf("seen repo %#v", hist.History["old/seen"])
	}
}

func TestRunLLMFailureWritesNothing(t *testing.T) {
	f := newFixture(t, "old/seen")
	f.copy.err = errors.New("402 credits depleted")
	before, _ := os.ReadFile(f.historyPath)
	if _, err := Run(context.Background(), testConfig(), f.deps, zerolog.Nop()); err == nil {
		t.Fatal("expected error")
	}
	if entries, _ := os.ReadDir(f.outDir); len(entries) != 0 {
		t.Fatalf("issue written despite LLM failure: %v", entries)
	}
	after, _ := os.ReadFile(f.historyPath)
	if !bytes.Equal(before, after) {
		t.Fatal("history changed despite LLM failure")
	}
}

func TestRunEmptyWeek(t *testing.T) {
	f := newFixture(t, "old/seen", "new/fresh", "old/returning")
	path, err := Run(context.Background(), testConfig(), f.deps, zerolog.Nop())
	if err != nil {
		t.Fatal(err)
	}
	if f.copy.calls != 1 || f.copy.seen != 0 {
		t.Fatalf("copy calls=%d candidates=%d", f.copy.calls, f.copy.seen)
	}
	data, _ := os.ReadFile(path)
	if !bytes.Contains(data, []byte(`"newProjects": []`)) || !bytes.Contains(data, []byte(`"newEntries": []`)) {
		t.Fatalf("issue %s", data)
	}
}

func TestRunRemovesIssueWhenHistoryUpdateFails(t *testing.T) {
	f := newFixture(t, "old/seen")
	f.copy.hook = func() {
		if err := os.Remove(f.historyPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(f.historyPath, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Run(context.Background(), testConfig(), f.deps, zerolog.Nop()); err == nil {
		t.Fatal("expected history update error")
	}
	if _, err := os.Stat(filepath.Join(f.outDir, "2026-W47.json")); !os.IsNotExist(err) {
		t.Fatalf("issue should be removed after history failure, stat error: %v", err)
	}
}
