// Command backfill-history records the first-seen week of every repository
// in the old Markdown reports, so the radar does not re-announce repositories
// that were on the list before history kept every repository.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/history"
)

func main() {
	reportsDir := flag.String("reports", "reports", "Directory of old Markdown reports")
	historyPath := flag.String("history", "data/history.json", "History file to update")
	flag.Parse()

	manager := history.NewManager(*historyPath, zerolog.Nop())
	hist, err := manager.LoadHistory()
	if err != nil {
		fail(err)
	}
	if err := validateReportsDir(*reportsDir); err != nil {
		fail(err)
	}
	added, err := history.Backfill(hist, *reportsDir)
	if err != nil {
		fail(err)
	}
	if err := manager.SaveHistory(hist); err != nil {
		fail(err)
	}
	fmt.Printf("Backfilled %d repositories; history now tracks %d.\n", added, len(hist.History))
}

func validateReportsDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("reports directory %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("reports path is not a directory: %s", path)
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "backfill failed:", err)
	os.Exit(1)
}
