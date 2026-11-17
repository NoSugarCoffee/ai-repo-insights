// Package radar turns trending data into weekly "AI 开源雷达" issues.
package radar

import (
	"fmt"
	"time"
)

// ReportID returns the ISO-week issue id (YYYY-Www) for t. ISO years differ
// from calendar years around New Year, e.g. 2027-01-01 is 2026-W53.
func ReportID(t time.Time) string {
	year, week := t.UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}
