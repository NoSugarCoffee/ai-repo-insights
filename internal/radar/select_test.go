package radar_test

import (
	"testing"
	"time"

	"ai-repo-insights/internal/githubapi"
	"ai-repo-insights/internal/models"
	"ai-repo-insights/internal/radar"
)

func repo(owner, name string) models.ClassifiedRepo {
	return models.ClassifiedRepo{Metadata: models.RepoMetadata{Owner: owner, Name: name}}
}

func TestUnseenDropsHistory(t *testing.T) {
	hist := &models.History{History: map[string]models.RepoHistory{"a/x": {WeeksInTop: 1}}}
	got := radar.Unseen([]models.ClassifiedRepo{repo("a", "x"), repo("b", "y")}, hist)
	if len(got) != 1 || got[0].Key() != "b/y" {
		t.Fatalf("got %#v", got)
	}
}

func TestAssignBoundary(t *testing.T) {
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	cutoff := now.AddDate(0, 0, -30)
	onCutoff := radar.Assign(repo("a", "x"), githubapi.RepoDetails{CreatedAt: cutoff}, "", now, 30)
	if onCutoff.Column != radar.ColumnNewProjects {
		t.Fatalf("created exactly 30 days ago: got %s", onCutoff.Column)
	}
	older := radar.Assign(repo("a", "x"), githubapi.RepoDetails{CreatedAt: cutoff.Add(-time.Second)}, "", now, 30)
	if older.Column != radar.ColumnNewEntries {
		t.Fatalf("created 30 days + 1s ago: got %s", older.Column)
	}
}
