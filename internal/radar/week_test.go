package radar_test

import (
	"testing"
	"time"

	"ai-repo-insights/internal/radar"
)

func TestReportID(t *testing.T) {
	cases := map[string]string{
		"2026-11-16T04:46:00Z": "2026-W47",
		"2027-01-01T00:00:00Z": "2026-W53",
		"2026-01-05T00:00:00Z": "2026-W02",
	}
	for in, want := range cases {
		ts, _ := time.Parse(time.RFC3339, in)
		if got := radar.ReportID(ts); got != want {
			t.Errorf("%s: got %s want %s", in, got, want)
		}
	}
}
