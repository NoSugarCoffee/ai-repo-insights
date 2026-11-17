package radar_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-repo-insights/internal/githubapi"
	"ai-repo-insights/internal/radar"
)

var issueNow = time.Date(2026, 11, 16, 4, 46, 0, 0, time.UTC)

func entry(name string, week, month int) radar.Candidate {
	r := repo("o", name)
	r.Metadata.StarsThisWeek = week
	r.Metadata.StarsThisMonth = month
	return radar.Candidate{Repo: r, Details: githubapi.RepoDetails{CreatedAt: issueNow.AddDate(-1, 0, 0)}, Column: radar.ColumnNewEntries}
}

func TestBuildIssueSortsAndCaps(t *testing.T) {
	var cands []radar.Candidate
	for i := 0; i < 10; i++ {
		cands = append(cands, entry(fmt.Sprintf("w%02d", i), 100+i, 0)) // on weekly page
	}
	for i := 0; i < 10; i++ {
		cands = append(cands, entry(fmt.Sprintf("m%02d", i), 0, 1000+i)) // monthly only
	}
	issue := radar.BuildIssue(issueNow, cands, radar.Copy{})
	if len(issue.NewEntries) != radar.MaxNewEntries {
		t.Fatalf("got %d entries", len(issue.NewEntries))
	}
	// Weekly stars first (desc), then repos without weekly stars by monthly stars (desc).
	if issue.NewEntries[0].Name != "w09" || issue.NewEntries[9].Name != "w00" || issue.NewEntries[10].Name != "m09" || issue.NewEntries[14].Name != "m05" {
		var names []string
		for _, it := range issue.NewEntries {
			names = append(names, it.Name)
		}
		t.Fatalf("order %v", names)
	}
	if issue.NewEntries[10].StarsThisWeek != nil {
		t.Fatalf("starsThisWeek should be null for monthly-only repo")
	}
	if issue.SchemaVersion != 1 || issue.ID != "2026-W47" || issue.Year != 2026 || issue.Week != 47 {
		t.Fatalf("header %#v", issue)
	}
}

func TestBuildIssueUsesIDAsFinalTieBreakerBeforeCap(t *testing.T) {
	var cands []radar.Candidate
	for i := radar.MaxNewEntries; i >= 0; i-- {
		cands = append(cands, entry(fmt.Sprintf("daily-%02d", i), 0, 0))
	}
	issue := radar.BuildIssue(issueNow, cands, radar.Copy{})
	if len(issue.NewEntries) != radar.MaxNewEntries {
		t.Fatalf("got %d entries", len(issue.NewEntries))
	}
	for i, item := range issue.NewEntries {
		want := fmt.Sprintf("o/daily-%02d", i)
		if item.ID != want {
			t.Fatalf("entry %d: got %s want %s", i, item.ID, want)
		}
	}
}

func TestBuildIssueEmptyWeek(t *testing.T) {
	data, err := json.Marshal(radar.BuildIssue(issueNow, nil, radar.Copy{Headline: "h", Summary: "s"}))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"newProjects":[]`, `"newEntries":[]`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("%s missing in %s", want, data)
		}
	}
}

func TestBuildIssueCopiesComments(t *testing.T) {
	r := repo("a", "x")
	r.Metadata.URL = "https://github.com/a/x"
	r.Metadata.Description = "desc"
	r.Metadata.Language = "go"
	c := radar.Candidate{Repo: r, Details: githubapi.RepoDetails{CreatedAt: issueNow.AddDate(0, 0, -3), Stars: 42}, Column: radar.ColumnNewProjects}
	issue := radar.BuildIssue(issueNow, []radar.Candidate{c}, radar.Copy{Headline: "h", Summary: "s", Comments: map[string]string{"a/x": "评"}})
	got := issue.NewProjects[0]
	want := radar.Item{Source: "github", ID: "a/x", URL: "https://github.com/a/x", Name: "x", Description: "desc", Comment: "评", Language: "go", CreatedAt: issueNow.AddDate(0, 0, -3), Stars: 42}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %#v", got)
	}
	if issue.Headline != "h" || issue.Summary != "s" {
		t.Fatalf("copy not applied")
	}
}

func TestWriteIssueFilename(t *testing.T) {
	dir := t.TempDir()
	path, err := radar.WriteIssue(dir, radar.BuildIssue(issueNow, nil, radar.Copy{Headline: "h", Summary: "s"}))
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, "2026-W47.json") {
		t.Fatalf("path %s", path)
	}
	data, _ := os.ReadFile(path)
	var back radar.Issue
	if err := json.Unmarshal(data, &back); err != nil || back.ID != "2026-W47" {
		t.Fatalf("round trip: %v %#v", err, back)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".2026-W47-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func TestWriteIssueRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	issue := radar.BuildIssue(issueNow, nil, radar.Copy{Headline: "h", Summary: "s"})
	if _, err := radar.WriteIssue(dir, issue); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(filepath.Join(dir, "2026-W47.json"))
	issue.Headline = "replacement"
	_, err := radar.WriteIssue(dir, issue)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected an already-exists error, got %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "2026-W47.json"))
	if string(after) != string(original) {
		t.Fatal("existing issue was overwritten")
	}
}
