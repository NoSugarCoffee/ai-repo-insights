package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadValidConfig tests loading a valid configuration
func TestLoadValidConfig(t *testing.T) {
	// Use the actual config directory
	config, err := Load("../../config")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify languages loaded
	if len(config.Languages) == 0 {
		t.Error("Languages should not be empty")
	}

	// Verify keywords loaded
	if len(config.Keywords.Include) == 0 {
		t.Error("Include keywords should not be empty")
	}
	if len(config.Keywords.Categories) == 0 {
		t.Error("Categories should not be empty")
	}

	// Verify settings loaded
	if config.Settings.WindowDays == 0 {
		t.Error("WindowDays should be set")
	}
	if config.Settings.TopN == 0 {
		t.Error("TopN should be set")
	}

	// Verify LLM config loaded
	if config.LLM.BaseURL == "" {
		t.Error("LLM BaseURL should be set")
	}
	if config.LLM.Model == "" {
		t.Error("LLM Model should be set")
	}

	// Verify validation passes
	errors := config.Validate()
	if len(errors) > 0 {
		t.Errorf("Validation should pass, but got errors: %v", errors)
	}
}

// TestLoadMissingFile tests error handling for missing config files
func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Error("Should return error for missing config directory")
	}
}

// TestDefaultValues tests that default values are applied for optional fields
func TestDefaultValues(t *testing.T) {
	// Create a temporary config directory
	tmpDir := t.TempDir()

	// Create minimal config files
	languagesJSON := `["python", "go"]`
	keywordsJSON := `{
		"include": ["test"],
		"exclude": [],
		"categories": {"test": ["test"]}
	}`
	settingsJSON := `{
		"window_days": 90,
		"short_window_days": 30,
		"top_n": 10,
		"report_language": "en",
		"filter_domain": "Test"
	}`
	llmJSON := `{
		"base_url": "https://api.test.com",
		"model": "test-model",
		"role_description": "test role"
	}`

	// Write config files
	if err := os.WriteFile(filepath.Join(tmpDir, "languages.json"), []byte(languagesJSON), 0644); err != nil {
		t.Fatalf("Failed to write languages.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "keywords.json"), []byte(keywordsJSON), 0644); err != nil {
		t.Fatalf("Failed to write keywords.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "llm.json"), []byte(llmJSON), 0644); err != nil {
		t.Fatalf("Failed to write llm.json: %v", err)
	}

	// Load config
	config, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults were applied
	if config.Settings.CacheTTLHours != 24 {
		t.Errorf("Expected CacheTTLHours default of 24, got %d", config.Settings.CacheTTLHours)
	}
	if config.Settings.NewRepoThresholdDays != 90 {
		t.Errorf("Expected NewRepoThresholdDays default of 90, got %d", config.Settings.NewRepoThresholdDays)
	}
	if config.Settings.DarkHorseAccelThreshold != 100 {
		t.Errorf("Expected DarkHorseAccelThreshold default of 100, got %d", config.Settings.DarkHorseAccelThreshold)
	}
	if config.Settings.ReportIDFormat != "YYYY-MM-weekN" {
		t.Errorf("Expected ReportIDFormat default of 'YYYY-MM-weekN', got %s", config.Settings.ReportIDFormat)
	}

	if config.LLM.TimeoutSeconds != 60 {
		t.Errorf("Expected TimeoutSeconds default of 60, got %d", config.LLM.TimeoutSeconds)
	}
	if config.LLM.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries default of 3, got %d", config.LLM.MaxRetries)
	}
	if config.LLM.Temperature != 0.7 {
		t.Errorf("Expected Temperature default of 0.7, got %f", config.LLM.Temperature)
	}
	if config.LLM.OutputTone != "concise, analytical, non-promotional" {
		t.Errorf("Expected OutputTone default, got %s", config.LLM.OutputTone)
	}
}

// TestValidation tests configuration validation
func TestValidation(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectErrors  bool
		errorContains string
	}{
		{
			name: "valid config",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					MaxRetries:      3,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors: false,
		},
		{
			name: "empty languages",
			config: Config{
				Languages: []string{},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "languages",
		},
		{
			name: "invalid window days",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      30,
					ShortWindowDays: 90, // Greater than WindowDays
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "short_window_days",
		},
		{
			name: "zero window days",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      0, // Invalid: must be > 0
					ShortWindowDays: 30,
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "window_days",
		},
		{
			name: "negative top_n",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            -5, // Invalid: must be > 0
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "top_n",
		},
		{
			name: "empty include keywords",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{}, // Invalid: cannot be empty
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "include keywords",
		},
		{
			name: "empty categories",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{}, // Invalid: cannot be empty
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "categories",
		},
		{
			name: "negative cache ttl",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            10,
					CacheTTLHours:   -1, // Invalid: cannot be negative
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "cache_ttl_hours",
		},
		{
			name: "invalid llm temperature",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     3.0, // Invalid: must be between 0 and 2
				},
			},
			expectErrors:  true,
			errorContains: "temperature",
		},
		{
			name: "empty llm base url",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "", // Invalid: cannot be empty
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "base_url",
		},
		{
			name: "negative max retries",
			config: Config{
				Languages: []string{"python"},
				Keywords: KeywordConfig{
					Include:    []string{"test"},
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      90,
					ShortWindowDays: 30,
					TopN:            10,
					ReportLanguage:  "en",
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "https://api.test.com",
					Model:           "test",
					TimeoutSeconds:  60,
					MaxRetries:      -1, // Invalid: cannot be negative
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "max_retries",
		},
		{
			name: "multiple validation errors",
			config: Config{
				Languages: []string{}, // Invalid: empty
				Keywords: KeywordConfig{
					Include:    []string{}, // Invalid: empty
					Categories: map[string][]string{"test": {"test"}},
				},
				Settings: Settings{
					WindowDays:      0,  // Invalid: must be > 0
					ShortWindowDays: 30,
					TopN:            -1, // Invalid: must be > 0
					ReportLanguage:  "",  // Invalid: empty
					FilterDomain:    "Test",
				},
				LLM: LLMConfig{
					BaseURL:         "", // Invalid: empty
					Model:           "test",
					TimeoutSeconds:  60,
					RoleDescription: "test",
					OutputTone:      "test",
					Temperature:     0.7,
				},
			},
			expectErrors:  true,
			errorContains: "", // Multiple errors expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.config.Validate()
			hasErrors := len(errors) > 0

			if hasErrors != tt.expectErrors {
				t.Errorf("Expected errors: %v, got errors: %v (%v)", tt.expectErrors, hasErrors, errors)
			}

			if tt.expectErrors && tt.errorContains != "" {
				found := false
				for _, err := range errors {
					if contains(err, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, errors)
				}
			}

			// For multiple validation errors test, verify we get multiple errors
			if tt.name == "multiple validation errors" && len(errors) < 2 {
				t.Errorf("Expected multiple validation errors, got %d: %v", len(errors), errors)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s string, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s string, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
