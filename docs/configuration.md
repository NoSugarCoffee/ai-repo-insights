# Configuration Guide

## Overview

The GitHub Repository Insights system is fully configurable through JSON files in the `config/` directory. This document describes all configuration options and their default values.

## Configuration Files

### languages.json

**Required**: Yes  
**Format**: JSON array of strings

List of programming languages to track from GitHub trending pages.

**Example**:
```json
["python", "typescript", "rust", "go", "javascript"]
```

### keywords.json

**Required**: Yes  
**Format**: JSON object

Defines keyword filtering rules and category mappings.

**Structure**:
- `include` (required): Array of keywords that repositories must match
- `exclude` (optional): Array of keywords that disqualify repositories
- `categories` (required): Object mapping category names to keyword arrays

**Example**:
```json
{
  "include": ["ai", "llm", "agent"],
  "exclude": ["tutorial", "awesome-list"],
  "categories": {
    "agent": ["agent", "autonomous"],
    "llm": ["llm", "language-model", "gpt"]
  }
}
```

### settings.json

**Required**: Yes  
**Format**: JSON object

Operational parameters for the analysis pipeline.

**Required Fields**:
- `window_days` (integer): Time window for analysis (e.g., 30)
- `short_window_days` (integer): Time window for Heat_7 calculation (e.g., 30)
- `top_n` (integer): Number of top repositories to include in reports (e.g., 50)
- `report_language` (string): Language for report generation (e.g., "zh-CN", "en")
- `filter_domain` (string): Domain being tracked (e.g., "AI", "Web Frameworks")

**Optional Fields with Defaults**:
- `cache_ttl_hours` (integer): Cache freshness threshold in hours
  - **Default**: 24
- `new_repo_threshold_days` (integer): Age threshold for "new" repositories
  - **Default**: 30
- `dark_horse_score_threshold` (integer): Minimum score for dark horse identification
  - **Default**: 100
- `report_id_format` (string): Format string for report IDs
  - **Default**: "YYYY-MM-DD"

**Example**:
```json
{
  "window_days": 30,
  "short_window_days": 30,
  "top_n": 50,
  "new_repo_threshold_days": 30,
  "dark_horse_score_threshold": 100,
  "cache_ttl_hours": 24,
  "report_language": "zh-CN",
  "report_id_format": "YYYY-MM-DD",
  "filter_domain": "AI"
}
```

### llm.json

**Required**: Yes  
**Format**: JSON object

LLM API integration settings.

**Required Fields**:
- `base_url` (string): LLM API base URL
- `model` (string): Model identifier
- `role_description` (string): Role description for the LLM prompt

**Optional Fields with Defaults**:
- `provider` (string): LLM provider ("openai" or "gemini")
  - **Default**: Auto-detected from `base_url`
  - Auto-detects "gemini" if base_url contains "generativelanguage.googleapis.com"
  - Otherwise defaults to "openai"
- `timeout_seconds` (integer): API request timeout
  - **Default**: 60
- `max_retries` (integer): Maximum retry attempts for failed requests
  - **Default**: 3
- `temperature` (float): LLM temperature parameter (0.0-2.0)
  - **Default**: 0.7
- `output_tone` (string): Desired tone for LLM output
  - **Default**: "concise, analytical, non-promotional"

**OpenAI Example**:
```json
{
  "base_url": "https://api.openai.com/v1",
  "model": "gpt-4",
  "timeout_seconds": 60,
  "max_retries": 3,
  "role_description": "GitHub open source project analyst",
  "output_tone": "concise, analytical, non-promotional",
  "temperature": 0.7
}
```

**Gemini Example**:
```json
{
  "base_url": "https://generativelanguage.googleapis.com/v1beta",
  "model": "gemini-1.5-pro",
  "provider": "gemini",
  "timeout_seconds": 60,
  "max_retries": 3,
  "role_description": "GitHub open source project analyst",
  "output_tone": "concise, analytical, non-promotional",
  "temperature": 0.7
}
```

**Note**: For Gemini, set the `LLM_API_KEY` environment variable to your Google AI API key.

## Validation Rules

The configuration loader validates all settings and returns descriptive errors for:

1. **Empty required fields**: Languages list, include keywords, categories
2. **Invalid ranges**: 
   - `window_days` and `short_window_days` must be > 0
   - `short_window_days` must be ≤ `window_days`
   - `top_n` must be > 0
   - `temperature` must be between 0 and 2
3. **Negative values**: Optional numeric fields cannot be negative

## Error Handling

- **Missing config files**: Returns a `ConfigError` and aborts pipeline execution
- **Invalid JSON**: Returns a `ConfigError` with parse details
- **Validation failures**: Returns a list of all validation errors

## Environment Variables

The following environment variables are required at runtime:

- `GITHUB_TOKEN`: GitHub API authentication token
- `LLM_API_KEY`: LLM service API key

These are NOT part of the configuration files for security reasons.
