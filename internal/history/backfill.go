package history

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"ai-repo-insights/internal/models"
)

var (
	reportDateLine = regexp.MustCompile(`\*\*Report Date\*\*:\s*(\d{4}-\d{2}-\d{2})`)
	// Only ranking-table rows count; category lists link to repositories too.
	tableRowRepo = regexp.MustCompile(`(?m)^\|\s*\d+\s*\|\s*\[[^\]]*\]\(https://github\.com/([^/\s)]+/[^/\s)]+)\)`)
)

type pastReport struct {
	id   string
	date string
	keys []string
}

// Backfill records the first-seen week of every repository in the Markdown
// reports under reportsDir. The earliest report wins for first seen and the
// latest for last seen, so it is safe to run on top of existing history.
// It returns how many repositories were not in history before.
func Backfill(hist *models.History, reportsDir string) (int, error) {
	hist.History = normalizeHistory(hist.History)
	paths, err := filepath.Glob(filepath.Join(reportsDir, "*.md"))
	if err != nil {
		return 0, err
	}
	reports := make([]pastReport, 0, len(paths))
	for _, path := range paths {
		report, err := readPastReport(path)
		if err != nil {
			return 0, err
		}
		reports = append(reports, report)
	}
	// File names sort wrongly across months ("week10" < "week6"), so order by date.
	sort.SliceStable(reports, func(i, j int) bool { return reports[i].date < reports[j].date })

	added := 0
	for _, report := range reports {
		for _, rawKey := range report.keys {
			key := models.NormalizeRepoKey(rawKey)
			existing, ok := hist.History[key]
			if !ok {
				added++
				hist.History[key] = models.RepoHistory{WeeksInTop: 1, FirstSeenReport: report.id, FirstSeenDate: report.date, LastSeenReport: report.id, LastSeenDate: report.date}
				continue
			}
			if existing.FirstSeenDate == "" || report.date < existing.FirstSeenDate {
				existing.FirstSeenReport, existing.FirstSeenDate = report.id, report.date
			}
			if report.date > existing.LastSeenDate {
				existing.LastSeenReport, existing.LastSeenDate = report.id, report.date
				existing.WeeksInTop++
			}
			hist.History[key] = existing
		}
	}
	return added, nil
}

func readPastReport(path string) (pastReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return pastReport{}, fmt.Errorf("read report %s: %w", path, err)
	}
	text := string(data)
	date := reportDateLine.FindStringSubmatch(text)
	if date == nil {
		return pastReport{}, fmt.Errorf("report %s has no **Report Date** line; cannot place it in time", path)
	}
	seen := map[string]bool{}
	var keys []string
	for _, match := range tableRowRepo.FindAllStringSubmatch(text, -1) {
		if key := models.NormalizeRepoKey(match[1]); !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	return pastReport{id: strings.TrimSuffix(filepath.Base(path), ".md"), date: date[1], keys: keys}, nil
}
