package radar

import (
	"time"

	"ai-repo-insights/internal/githubapi"
	"ai-repo-insights/internal/models"
)

// Column is the section of an issue a repository is shown in.
type Column string

const (
	ColumnNewProjects Column = "newProjects"
	ColumnNewEntries  Column = "newEntries"
)

// Candidate is a repository shown for the first time this week, with the
// details the LLM needs to write about it.
type Candidate struct {
	Repo    models.ClassifiedRepo
	Details githubapi.RepoDetails
	Readme  string
	Column  Column
}

// Unseen keeps the repositories that have never appeared in any issue: each
// repository is shown once, in the week it first surfaces.
func Unseen(repos []models.ClassifiedRepo, hist *models.History) []models.ClassifiedRepo {
	unseen := make([]models.ClassifiedRepo, 0, len(repos))
	for _, repo := range repos {
		if !hist.Seen(repo.Key()) {
			unseen = append(unseen, repo)
		}
	}
	return unseen
}

// Assign places an unseen repository: created within newRepoDays of now makes
// it a new project, anything older a new entry.
func Assign(repo models.ClassifiedRepo, details githubapi.RepoDetails, readme string, now time.Time, newRepoDays int) Candidate {
	column := ColumnNewEntries
	if !details.CreatedAt.Before(now.AddDate(0, 0, -newRepoDays)) {
		column = ColumnNewProjects
	}
	return Candidate{Repo: repo, Details: details, Readme: readme, Column: column}
}
