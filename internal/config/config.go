package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apperrors "ai-repo-insights/internal/errors"
)

// KeywordConfig represents keyword filtering configuration
type KeywordConfig struct {
	Include    []string            `json:"include"`
	Exclude    []string            `json:"exclude"`
	Categories map[string][]string `json:"categories"`
}

// Settings represents operational settings
type Settings struct {
	WindowDays              int    `json:"window_days"`
	ShortWindowDays         int    `json:"short_window_days"`
	TopN                    int    `json:"top_n"`
	NewRepoThresholdDays    int    `json:"new_repo_threshold_days"`
	DarkHorseAccelThreshold int    `json:"dark_horse_accel_threshold"`
	CacheTTLHours           int    `json:"cache_ttl_hours"`
	ReportLanguage          string `json:"report_language"`
	ReportIDFormat          string `json:"report_id_format"`
	FilterDomain            string `json:"filter_domain"`
}

// LLMConfig represents LLM integration settings
type LLMConfig struct {
	BaseURL         string  `json:"base_url"`
	Model           string  `json:"model"`
	Provider        string  `json:"provider"`
	TimeoutSeconds  int     `json:"timeout_seconds"`
	MaxRetries      int     `json:"max_retries"`
	RoleDescription string  `json:"role_description"`
	OutputTone      string  `json:"output_tone"`
	Temperature     float64 `json:"temperature"`
}

// Config represents the complete system configuration
type Config struct {
	Languages []string      `json:"languages"`
	Keywords  KeywordConfig `json:"keywords"`
	Settings  Settings      `json:"settings"`
	LLM       LLMConfig     `json:"llm"`
}

// Load loads all configuration files from the specified directory
// Applies default values for optional fields after loading
func Load(configDir string) (*Config, error) {
	config := &Config{}

	// Load languages
	languagesPath := filepath.Join(configDir, "languages.json")
	if err := loadJSONFile(languagesPath, &config.Languages); err != nil {
		return nil, apperrors.NewConfigError("failed to load languages.json", err)
	}

	// Load keywords
	keywordsPath := filepath.Join(configDir, "keywords.json")
	if err := loadJSONFile(keywordsPath, &config.Keywords); err != nil {
		return nil, apperrors.NewConfigError("failed to load keywords.json", err)
	}

	// Load settings
	settingsPath := filepath.Join(configDir, "settings.json")
	if err := loadJSONFile(settingsPath, &config.Settings); err != nil {
		return nil, apperrors.NewConfigError("failed to load settings.json", err)
	}
	applySettingsDefaults(&config.Settings)

	// Load LLM config
	llmPath := filepath.Join(configDir, "llm.json")
	if err := loadJSONFile(llmPath, &config.LLM); err != nil {
		return nil, apperrors.NewConfigError("failed to load llm.json", err)
	}
	applyLLMDefaults(&config.LLM)

	return config, nil
}

// loadJSONFile loads a JSON file into the provided interface
func loadJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %w", path, err)
	}

	return nil
}

// Validate validates the configuration and returns a list of errors
func (c *Config) Validate() []string {
	var errors []string

	// Validate languages
	if len(c.Languages) == 0 {
		errors = append(errors, "languages list cannot be empty")
	}

	// Validate keywords
	if len(c.Keywords.Include) == 0 {
		errors = append(errors, "include keywords cannot be empty")
	}
	if len(c.Keywords.Categories) == 0 {
		errors = append(errors, "categories cannot be empty")
	}

	// Validate settings - required fields
	if c.Settings.WindowDays <= 0 {
		errors = append(errors, "window_days must be greater than 0")
	}
	if c.Settings.ShortWindowDays <= 0 {
		errors = append(errors, "short_window_days must be greater than 0")
	}
	if c.Settings.ShortWindowDays > c.Settings.WindowDays {
		errors = append(errors, "short_window_days must be less than or equal to window_days")
	}
	if c.Settings.TopN <= 0 {
		errors = append(errors, "top_n must be greater than 0")
	}
	if c.Settings.ReportLanguage == "" {
		errors = append(errors, "report_language cannot be empty")
	}
	if c.Settings.FilterDomain == "" {
		errors = append(errors, "filter_domain cannot be empty")
	}
	// Optional fields with defaults are validated for range only
	if c.Settings.NewRepoThresholdDays < 0 {
		errors = append(errors, "new_repo_threshold_days cannot be negative")
	}
	if c.Settings.DarkHorseAccelThreshold < 0 {
		errors = append(errors, "dark_horse_accel_threshold cannot be negative")
	}
	if c.Settings.CacheTTLHours < 0 {
		errors = append(errors, "cache_ttl_hours cannot be negative")
	}

	// Validate LLM config - all fields are required
	if c.LLM.BaseURL == "" {
		errors = append(errors, "llm base_url cannot be empty")
	}
	if c.LLM.Model == "" {
		errors = append(errors, "llm model cannot be empty")
	}
	if c.LLM.TimeoutSeconds <= 0 {
		errors = append(errors, "llm timeout_seconds must be greater than 0")
	}
	if c.LLM.MaxRetries < 0 {
		errors = append(errors, "llm max_retries cannot be negative")
	}
	if c.LLM.RoleDescription == "" {
		errors = append(errors, "llm role_description cannot be empty")
	}
	if c.LLM.OutputTone == "" {
		errors = append(errors, "llm output_tone cannot be empty")
	}
	if c.LLM.Temperature < 0 || c.LLM.Temperature > 2 {
		errors = append(errors, "llm temperature must be between 0 and 2")
	}

	return errors
}

// applySettingsDefaults applies default values for optional settings fields
func applySettingsDefaults(s *Settings) {
	// Optional fields with defaults
	if s.CacheTTLHours == 0 {
		s.CacheTTLHours = 24 // Default: 24 hours
	}
	if s.NewRepoThresholdDays == 0 {
		s.NewRepoThresholdDays = 90 // Default: 90 days
	}
	if s.DarkHorseAccelThreshold == 0 {
		s.DarkHorseAccelThreshold = 100 // Default: 100 stars
	}
	if s.ReportIDFormat == "" {
		s.ReportIDFormat = "YYYY-MM-weekN" // Default format
	}
}

// applyLLMDefaults applies default values for optional LLM config fields
func applyLLMDefaults(l *LLMConfig) {
	// Optional fields with defaults
	if l.TimeoutSeconds == 0 {
		l.TimeoutSeconds = 60 // Default: 60 seconds
	}
	if l.MaxRetries == 0 {
		l.MaxRetries = 3 // Default: 3 retries
	}
	if l.Temperature == 0 {
		l.Temperature = 0.7 // Default: 0.7
	}
	if l.OutputTone == "" {
		l.OutputTone = "concise, analytical, non-promotional" // Default tone
	}
	if l.Provider == "" {
		l.Provider = detectProvider(l.BaseURL) // Auto-detect from base_url
	}
}

// detectProvider auto-detects the LLM provider from the base URL
func detectProvider(baseURL string) string {
	if baseURL == "" {
		return "openai"
	}
	// Check for Gemini/Google AI
	if strings.Contains(baseURL, "generativelanguage.googleapis.com") {
		return "gemini"
	}
	// Default to OpenAI-compatible
	return "openai"
}
