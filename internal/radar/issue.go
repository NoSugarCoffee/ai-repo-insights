package radar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	// SchemaVersion is bumped on any incompatible change; the site refuses
	// versions it does not know.
	SchemaVersion = 1
	// MaxNewEntries caps the 新上榜 column; every new project is shown.
	MaxNewEntries = 15
)

// Copy is the LLM-written text of an issue. Comments are keyed by "owner/repo".
type Copy struct {
	Headline string            `json:"headline"`
	Summary  string            `json:"summary"`
	Comments map[string]string `json:"comments"`
}

// Item is one repository in an issue. Star deltas are nil when the repository
// was not on that trending page, so readers never see an unknown shown as 0.
type Item struct {
	Source         string    `json:"source"`
	ID             string    `json:"id"`
	URL            string    `json:"url"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Comment        string    `json:"comment"`
	Language       string    `json:"language"`
	CreatedAt      time.Time `json:"createdAt"`
	Stars          int       `json:"stars"`
	StarsThisWeek  *int      `json:"starsThisWeek"`
	StarsThisMonth *int      `json:"starsThisMonth"`
}

// Issue is one week of the radar, written to data/weekly/<ID>.json.
type Issue struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	Year          int       `json:"year"`
	Week          int       `json:"week"`
	GeneratedAt   time.Time `json:"generatedAt"`
	Headline      string    `json:"headline"`
	Summary       string    `json:"summary"`
	NewProjects   []Item    `json:"newProjects"`
	NewEntries    []Item    `json:"newEntries"`
}

// BuildIssue assembles the issue for the week containing now.
func BuildIssue(now time.Time, cands []Candidate, copy Copy) Issue {
	year, week := now.UTC().ISOWeek()
	issue := Issue{
		SchemaVersion: SchemaVersion,
		ID:            ReportID(now),
		Year:          year,
		Week:          week,
		GeneratedAt:   now.UTC(),
		Headline:      copy.Headline,
		Summary:       copy.Summary,
		NewProjects:   []Item{},
		NewEntries:    []Item{},
	}
	for _, cand := range cands {
		item := toItem(cand, copy.Comments[cand.Repo.Key()])
		if cand.Column == ColumnNewProjects {
			issue.NewProjects = append(issue.NewProjects, item)
		} else {
			issue.NewEntries = append(issue.NewEntries, item)
		}
	}
	sortItems(issue.NewProjects)
	sortItems(issue.NewEntries)
	if len(issue.NewEntries) > MaxNewEntries {
		issue.NewEntries = issue.NewEntries[:MaxNewEntries]
	}
	return issue
}

// WriteIssue writes the issue as <dir>/<ID>.json and returns the path. It
// refuses to replace an existing issue: a second run in the same week finds
// every repository already in history and would publish an empty issue over
// the real one.
func WriteIssue(dir string, issue Issue) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create issue dir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(issue, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal issue %s: %w", issue.ID, err)
	}
	path := filepath.Join(dir, issue.ID+".json")
	file, err := os.CreateTemp(dir, "."+issue.ID+"-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary issue for %s: %w", path, err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0o644); err != nil {
		return "", closeAfterFileError(file, path, fmt.Errorf("set permissions on temporary issue for %s: %w", path, err))
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return "", closeAfterFileError(file, path, fmt.Errorf("write issue %s: %w", path, err))
	}
	if err := file.Sync(); err != nil {
		return "", closeAfterFileError(file, path, fmt.Errorf("sync issue %s: %w", path, err))
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close issue %s: %w", path, err)
	}
	// Link publishes the fully written file atomically without replacing an
	// existing issue. os.Rename would overwrite the destination on Unix.
	if err := os.Link(tempPath, path); errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("issue %s already exists; to regenerate this week, delete it and revert data/history.json to the commit before it, then re-run", path)
	} else if err != nil {
		return "", fmt.Errorf("publish issue %s: %w", path, err)
	}
	return path, nil
}

func closeAfterFileError(file *os.File, path string, cause error) error {
	if err := file.Close(); err != nil {
		return errors.Join(cause, fmt.Errorf("close temporary issue for %s: %w", path, err))
	}
	return cause
}

func toItem(cand Candidate, comment string) Item {
	meta := cand.Repo.Metadata
	return Item{
		Source:         "github",
		ID:             cand.Repo.Key(),
		URL:            meta.URL,
		Name:           meta.Name,
		Description:    meta.Description,
		Comment:        comment,
		Language:       meta.Language,
		CreatedAt:      cand.Details.CreatedAt,
		Stars:          cand.Details.Stars,
		StarsThisWeek:  positive(meta.StarsThisWeek),
		StarsThisMonth: positive(meta.StarsThisMonth),
	}
}

// positive maps the fetcher's 0 ("not on that trending page") to nil.
func positive(n int) *int {
	if n <= 0 {
		return nil
	}
	return &n
}

// sortItems orders by weekly stars, then monthly stars, both descending,
// with unknown (nil) values after every known one.
func sortItems(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if c := compareDesc(items[i].StarsThisWeek, items[j].StarsThisWeek); c != 0 {
			return c < 0
		}
		if c := compareDesc(items[i].StarsThisMonth, items[j].StarsThisMonth); c != 0 {
			return c < 0
		}
		return items[i].ID < items[j].ID
	})
}

func compareDesc(a, b *int) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	case *a > *b:
		return -1
	case *a < *b:
		return 1
	}
	return 0
}
