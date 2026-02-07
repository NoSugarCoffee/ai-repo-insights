package classifier

import (
	"strings"
	"testing"
	"time"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/models"
)

func TestClassifier_Classify(t *testing.T) {
	keywords := config.KeywordConfig{
		Include: []string{"ai", "llm", "machine-learning"},
		Exclude: []string{"tutorial", "awesome-list"},
		Categories: map[string][]string{
			"agent": {"agent", "autonomous"},
			"llm":   {"llm", "language-model", "gpt"},
			"rag":   {"rag", "retrieval", "vector"},
		},
	}

	classifier := New(keywords)

	tests := []struct {
		name     string
		repos    []models.RepoMetadata
		expected int // Expected number of classified repos
	}{
		{
			name: "filters repos that match include and not exclude",
			repos: []models.RepoMetadata{
				{
					Owner:       "test",
					Name:        "ai-project",
					Description: "An AI project",
					Topics:      []string{"machine-learning"},
				},
				{
					Owner:       "test",
					Name:        "tutorial-repo",
					Description: "AI tutorial",
					Topics:      []string{"ai"},
				},
				{
					Owner:       "test",
					Name:        "web-app",
					Description: "A web application",
					Topics:      []string{"web"},
				},
			},
			expected: 1, // Only first repo passes (second excluded, third doesn't match include)
		},
		{
			name:     "returns empty list for empty input",
			repos:    []models.RepoMetadata{},
			expected: 0,
		},
		{
			name: "filters all repos when none match include",
			repos: []models.RepoMetadata{
				{
					Owner:       "test",
					Name:        "web-app",
					Description: "A web application",
					Topics:      []string{"web"},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.repos)
			if len(result) != tt.expected {
				t.Errorf("expected %d classified repos, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestClassifier_MatchesInclude(t *testing.T) {
	keywords := config.KeywordConfig{
		Include: []string{"ai", "llm", "machine-learning"},
	}

	classifier := New(keywords)

	tests := []struct {
		name     string
		repo     models.RepoMetadata
		expected bool
	}{
		{
			name: "matches keyword in name",
			repo: models.RepoMetadata{
				Name:        "ai-project",
				Description: "A project",
			},
			expected: true,
		},
		{
			name: "matches keyword in description",
			repo: models.RepoMetadata{
				Name:        "project",
				Description: "An LLM framework",
			},
			expected: true,
		},
		{
			name: "matches keyword in topics",
			repo: models.RepoMetadata{
				Name:        "project",
				Description: "A project",
				Topics:      []string{"machine-learning"},
			},
			expected: true,
		},
		{
			name: "case insensitive matching",
			repo: models.RepoMetadata{
				Name:        "AI-Project",
				Description: "An AI project",
			},
			expected: true,
		},
		{
			name: "no match",
			repo: models.RepoMetadata{
				Name:        "web-app",
				Description: "A web application",
				Topics:      []string{"web"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.matchesInclude(tt.repo)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestClassifier_MatchesExclude(t *testing.T) {
	keywords := config.KeywordConfig{
		Exclude: []string{"tutorial", "awesome-list", "learning"},
	}

	classifier := New(keywords)

	tests := []struct {
		name     string
		repo     models.RepoMetadata
		expected bool
	}{
		{
			name: "matches exclude keyword in name",
			repo: models.RepoMetadata{
				Name:        "ai-tutorial",
				Description: "A project",
			},
			expected: true,
		},
		{
			name: "matches exclude keyword in description",
			repo: models.RepoMetadata{
				Name:        "project",
				Description: "An awesome-list of resources",
			},
			expected: true,
		},
		{
			name: "case insensitive matching",
			repo: models.RepoMetadata{
				Name:        "Learning-AI",
				Description: "A project",
			},
			expected: true,
		},
		{
			name: "no match",
			repo: models.RepoMetadata{
				Name:        "ai-project",
				Description: "An AI framework",
				Topics:      []string{"ai"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.matchesExclude(tt.repo)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestClassifier_AssignCategories(t *testing.T) {
	keywords := config.KeywordConfig{
		Categories: map[string][]string{
			"agent": {"agent", "autonomous"},
			"llm":   {"llm", "language-model", "gpt"},
			"rag":   {"rag", "retrieval", "vector"},
		},
	}

	classifier := New(keywords)

	tests := []struct {
		name     string
		repo     models.RepoMetadata
		expected []string
	}{
		{
			name: "assigns single category",
			repo: models.RepoMetadata{
				Name:        "llm-project",
				Description: "A language model",
			},
			expected: []string{"llm"},
		},
		{
			name: "assigns multiple categories",
			repo: models.RepoMetadata{
				Name:        "rag-agent",
				Description: "An autonomous agent with retrieval",
			},
			expected: []string{"agent", "rag"},
		},
		{
			name: "assigns no categories when no match",
			repo: models.RepoMetadata{
				Name:        "web-app",
				Description: "A web application",
			},
			expected: []string{},
		},
		{
			name: "case insensitive category matching",
			repo: models.RepoMetadata{
				Name:        "GPT-Agent",
				Description: "An Autonomous system",
			},
			expected: []string{"agent", "llm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.assignCategories(tt.repo)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d categories, got %d", len(tt.expected), len(result))
				return
			}

			// Check that all expected categories are present (order may vary)
			categoryMap := make(map[string]bool)
			for _, cat := range result {
				categoryMap[cat] = true
			}

			for _, expectedCat := range tt.expected {
				if !categoryMap[expectedCat] {
					t.Errorf("expected category %s not found in result", expectedCat)
				}
			}
		})
	}
}

func TestClassifier_SelectPrimaryCategory(t *testing.T) {
	keywords := config.KeywordConfig{
		Categories: map[string][]string{
			"agent": {"agent", "autonomous"},
			"llm":   {"llm", "language-model", "gpt"},
			"rag":   {"rag", "retrieval", "vector"},
		},
	}

	classifier := New(keywords)

	tests := []struct {
		name       string
		repo       models.RepoMetadata
		categories []string
		expected   string
	}{
		{
			name: "selects category with most keyword matches",
			repo: models.RepoMetadata{
				Name:        "llm-gpt-language-model",
				Description: "An agent system",
			},
			categories: []string{"agent", "llm"},
			expected:   "llm", // llm has 3 matches, agent has 1
		},
		{
			name: "returns first category when match counts are equal",
			repo: models.RepoMetadata{
				Name:        "agent-llm",
				Description: "A system",
			},
			categories: []string{"agent", "llm"},
			expected:   "agent", // Both have 1 match, return first
		},
		{
			name: "returns empty string for empty categories",
			repo: models.RepoMetadata{
				Name:        "project",
				Description: "A project",
			},
			categories: []string{},
			expected:   "",
		},
		{
			name: "returns single category",
			repo: models.RepoMetadata{
				Name:        "rag-system",
				Description: "A retrieval system",
			},
			categories: []string{"rag"},
			expected:   "rag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.selectPrimaryCategory(tt.repo, tt.categories)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClassifier_CalculateMatchScore(t *testing.T) {
	keywords := config.KeywordConfig{
		Include: []string{"ai", "llm"},
		Categories: map[string][]string{
			"agent": {"agent", "autonomous"},
			"llm":   {"llm", "gpt"},
		},
	}

	classifier := New(keywords)

	tests := []struct {
		name     string
		repo     models.RepoMetadata
		expected int
	}{
		{
			name: "counts all keyword matches",
			repo: models.RepoMetadata{
				Name:        "ai-llm-agent",
				Description: "An autonomous GPT system",
			},
			expected: 6, // ai, llm (include) + agent, autonomous, llm, gpt (categories)
		},
		{
			name: "counts zero for no matches",
			repo: models.RepoMetadata{
				Name:        "web-app",
				Description: "A web application",
			},
			expected: 0,
		},
		{
			name: "counts duplicate keyword only once per occurrence",
			repo: models.RepoMetadata{
				Name:        "llm-project",
				Description: "An LLM framework",
			},
			expected: 2, // llm appears in both include and categories, counted twice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.calculateMatchScore(tt.repo)
			if result != tt.expected {
				t.Errorf("expected score %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestClassifier_BuildSearchText(t *testing.T) {
	classifier := New(config.KeywordConfig{})

	tests := []struct {
		name     string
		repo     models.RepoMetadata
		contains []string
	}{
		{
			name: "includes name, description, and topics",
			repo: models.RepoMetadata{
				Name:        "AI-Project",
				Description: "An LLM Framework",
				Topics:      []string{"Machine-Learning", "GPT"},
			},
			contains: []string{"ai-project", "llm framework", "machine-learning", "gpt"},
		},
		{
			name: "converts to lowercase",
			repo: models.RepoMetadata{
				Name:        "MyProject",
				Description: "DESCRIPTION",
				Topics:      []string{"TOPIC"},
			},
			contains: []string{"myproject", "description", "topic"},
		},
		{
			name: "handles empty fields",
			repo: models.RepoMetadata{
				Name:        "project",
				Description: "",
				Topics:      []string{},
			},
			contains: []string{"project"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.buildSearchText(tt.repo)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected search text to contain %q, got %q", expected, result)
				}
			}
		})
	}
}

func TestClassifier_IntegrationTest(t *testing.T) {
	keywords := config.KeywordConfig{
		Include: []string{"ai", "llm", "machine-learning"},
		Exclude: []string{"tutorial", "awesome"},
		Categories: map[string][]string{
			"agent": {"agent", "autonomous"},
			"llm":   {"llm", "language-model", "gpt"},
			"rag":   {"rag", "retrieval", "vector"},
		},
	}

	classifier := New(keywords)

	repos := []models.RepoMetadata{
		{
			Owner:       "openai",
			Name:        "gpt-agent",
			Description: "An autonomous AI agent using GPT",
			Topics:      []string{"ai", "agent", "llm"},
			CreatedAt:   time.Now(),
		},
		{
			Owner:       "user",
			Name:        "ai-tutorial",
			Description: "Learn AI basics",
			Topics:      []string{"tutorial", "ai"},
			CreatedAt:   time.Now(),
		},
		{
			Owner:       "company",
			Name:        "rag-system",
			Description: "Retrieval augmented generation with vector database",
			Topics:      []string{"rag", "ai", "retrieval"},
			CreatedAt:   time.Now(),
		},
		{
			Owner:       "dev",
			Name:        "web-framework",
			Description: "A modern web framework",
			Topics:      []string{"web", "framework"},
			CreatedAt:   time.Now(),
		},
	}

	result := classifier.Classify(repos)

	// Should classify 2 repos: gpt-agent and rag-system
	// ai-tutorial is excluded, web-framework doesn't match include
	if len(result) != 2 {
		t.Errorf("expected 2 classified repos, got %d", len(result))
	}

	// Check first repo (gpt-agent)
	if len(result) > 0 {
		repo := result[0]
		if repo.Metadata.Name != "gpt-agent" {
			t.Errorf("expected first repo to be gpt-agent, got %s", repo.Metadata.Name)
		}
		if len(repo.Categories) == 0 {
			t.Error("expected categories to be assigned")
		}
		if repo.PrimaryCategory == "" {
			t.Error("expected primary category to be set")
		}
		if repo.MatchScore == 0 {
			t.Error("expected match score to be greater than 0")
		}

		// Should have both agent and llm categories
		categoryMap := make(map[string]bool)
		for _, cat := range repo.Categories {
			categoryMap[cat] = true
		}
		if !categoryMap["agent"] || !categoryMap["llm"] {
			t.Errorf("expected both agent and llm categories, got %v", repo.Categories)
		}
	}

	// Check second repo (rag-system)
	if len(result) > 1 {
		repo := result[1]
		if repo.Metadata.Name != "rag-system" {
			t.Errorf("expected second repo to be rag-system, got %s", repo.Metadata.Name)
		}
		if len(repo.Categories) == 0 {
			t.Error("expected categories to be assigned")
		}

		// Should have rag category
		categoryMap := make(map[string]bool)
		for _, cat := range repo.Categories {
			categoryMap[cat] = true
		}
		if !categoryMap["rag"] {
			t.Errorf("expected rag category, got %v", repo.Categories)
		}
	}
}
