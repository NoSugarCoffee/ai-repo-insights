# AI Repository Insights

An automated analysis platform that tracks, ranks, and reports on trending repositories from GitHub based on configurable filtering criteria.

## Features

- **Automated Trending Tracking**: Scrapes trending repositories directly from GitHub's trending page
- **Smart Classification**: Categorizes repositories based on configurable keyword matching
- **Scoring System**: Ranks repositories using a weighted formula combining daily, weekly, and monthly star data
- **Historical Tracking**: Monitors consecutive appearances in top rankings
- **LLM-Enhanced Reports**: Generates analytical commentary using LLM integration (supports OpenAI and Gemini)
- **Flexible Configuration**: Fully customizable through JSON configuration files

## Metrics

The system tracks key metrics from GitHub trending data:

- **Heat_7**: Stars gained in the last 7 days (from GitHub trending)
- **Heat_30**: Stars gained in the last 30 days (from GitHub trending)
- **Score**: Weighted scoring formula: `0.6 × stars_today + 0.3 × (stars_week / 7) + 0.1 × (stars_month / 30)`
  - Emphasizes recent activity (60%) while considering sustained trends (40%)

Repositories are ranked primarily by Heat_30 (descending), with Score as a tiebreaker.

## Project Structure

```
ai-repo-insights/
├── cmd/
│   └── ai-repo-insights/     # Main application entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── errors/              # Error types and handling
│   ├── logging/             # Logging utilities
│   ├── models/              # Core data models
│   ├── fetcher/             # Trending data scraper (direct GitHub scraping)
│   ├── classifier/          # Repository classifier
│   ├── calculator/          # Score calculator
│   ├── history/             # History manager
│   ├── summary/             # Summary builder
│   ├── llm/                 # LLM client
│   ├── report/              # Report generator
│   └── pipeline/            # Pipeline orchestrator
├── config/                  # Configuration files
│   ├── languages.json       # Languages to track
│   ├── keywords.json        # Keyword filtering rules
│   ├── settings.json        # Operational settings
│   └── llm.json            # LLM integration settings
├── data/                    # Data storage
│   ├── trending_raw/        # Raw trending data (cached)
│   ├── stars_raw/           # Raw star history data (cached)
│   ├── history.json         # Historical tracking data
│   └── summaries/           # Summary backups
└── reports/                 # Generated reports
```

## Configuration

The system is fully configurable through JSON files in the `config/` directory:

- **languages.json**: List of programming languages to track
- **keywords.json**: Include/exclude keywords and category mappings
- **settings.json**: Operational parameters (windows, thresholds, etc.)
- **llm.json**: LLM API configuration

See [docs/configuration.md](docs/configuration.md) for detailed configuration options.

## Environment Variables

- `GITHUB_TOKEN`: (Optional) GitHub API token for authentication
- `LLM_API_KEY`: (Optional) API key for LLM service - if not set, uses template-based reports
  - For OpenAI: Your OpenAI API key
  - For Gemini: Your Google AI API key

## Building

```bash
go build -o ai-repo-insights ./cmd/ai-repo-insights
```

## Running

```bash
./ai-repo-insights
```

The application will:
1. Scrape trending repositories from GitHub (daily, weekly, monthly)
2. Classify and score repositories
3. Generate a markdown report in `reports/`
4. Save summary data in `data/summaries/`

## Dependencies

- Go 1.24.0+
- [github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery) - HTML parsing for web scraping
- [github.com/rs/zerolog](https://github.com/rs/zerolog) - Structured logging

## License

MIT
