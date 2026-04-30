<div align="center">

<img src="logo.png" alt="ai-repo-insights" width="512"/>

[![Go Version](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.0.0-green.svg)](cmd/ai-repo-insights/main.go)

**🔭 Automatically track, rank, and report on GitHub's trending repositories — filtered to topics that matter to you 📊**

</div>

---

## ⚡ Quick Start

```bash
# 1. Clone and build
git clone https://github.com/your-org/ai-repo-insights.git
cd ai-repo-insights
go build -o ai-repo-insights ./cmd/ai-repo-insights

# 2. Set environment variables
export GITHUB_TOKEN=ghp_...       # Required for star history fetching
export LLM_API_KEY=sk-...         # Optional: enables AI-generated commentary

# 3. Run
./ai-repo-insights
```

Output written to `reports/` and `data/summaries/`.

---

## ✨ Features

- **🤖 Automated Trending Tracking** — Scrapes GitHub trending (daily, weekly, monthly) across configurable languages
- **🏷️ Smart Classification** — Categorizes repositories by configurable include/exclude keywords and category mappings
- **📈 Scoring System** — Ranks repos using a weighted formula combining daily, weekly, and monthly star data
- **🕰️ Historical Tracking** — Monitors consecutive appearances in top rankings across runs
- **🧠 LLM-Enhanced Reports** — Generates analytical commentary via OpenAI or Gemini; falls back to templates when no key is set
- **🔧 Flexible Configuration** — Fully customizable through JSON config files; swap domains with a single flag

## 📊 Metrics

| Metric | Description |
|--------|-------------|
| **Heat_7** | Stars gained in the last 7 days (from GitHub trending) |
| **Heat_30** | Stars gained in the last 30 days (from GitHub trending) |
| **Score** | `0.6 × stars_today + 0.3 × (stars_week / 7) + 0.1 × (stars_month / 30)` |

Repositories are ranked primarily by **Heat_30** (descending), with **Score** as a tiebreaker. The formula emphasizes recent activity (60%) while rewarding sustained trends (40%).

---

## 🚀 Usage

```bash
# Daily report (default)
./ai-repo-insights

# Weekly report (uses YYYY-MM-weekN ID format)
./ai-repo-insights -weekly

# Custom config directory
./ai-repo-insights -config examples/web-frameworks

# Custom report ID
./ai-repo-insights -report-id 2026-04-week18

# Debug logging
./ai-repo-insights -log-level debug
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `config` | Path to configuration directory |
| `-report-id` | auto | Custom report ID (overrides auto-generation) |
| `-weekly` | `false` | Use week-based report ID format (`YYYY-MM-weekN`) |
| `-log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `-version` | — | Print version and exit |
| `-help` | — | Print usage and exit |

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_TOKEN` | ⚠️ Recommended | GitHub API token — star history fetching fails without it |
| `LLM_API_KEY` | Optional | API key for LLM service (OpenAI or Gemini); uses template reports if unset |

---

## ⚙️ Configuration

All configuration lives in JSON files under `config/` (or a custom directory via `-config`).

| File | Description |
|------|-------------|
| `languages.json` | Programming languages to track |
| `keywords.json` | Include/exclude keywords and category mappings |
| `settings.json` | Operational parameters (`top_n`, `window_days`, `report_language`, etc.) |
| `llm.json` | LLM API settings (`base_url`, `model`, `temperature`, etc.) |

See [docs/configuration.md](docs/configuration.md) for full reference.

### 🌐 Example Configurations

Pre-built configs for different tracking domains:

```bash
# AI/LLM repositories (default config)
./ai-repo-insights

# Web frameworks (React, Vue, Next.js, FastAPI, …)
./ai-repo-insights -config examples/web-frameworks

# DevOps tools (Kubernetes, Terraform, Prometheus, …)
./ai-repo-insights -config examples/devops
```

See [examples/README.md](examples/README.md) for how to create custom domain configs.

---

## 🗂️ Project Structure

```
ai-repo-insights/
├── cmd/
│   └── ai-repo-insights/     # Main application entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── errors/               # Error types and handling
│   ├── logging/              # Structured logging
│   ├── models/               # Core data models
│   ├── fetcher/              # GitHub trending scraper
│   ├── classifier/           # Keyword-based classifier
│   ├── calculator/           # Score calculator
│   ├── history/              # Historical tracking
│   ├── summary/              # Summary builder
│   ├── llm/                  # LLM client (OpenAI / Gemini)
│   ├── report/               # Markdown report generator
│   └── pipeline/             # Pipeline orchestrator
├── config/                   # Default configuration files
├── examples/                 # Domain-specific example configs
├── data/                     # Runtime data storage
│   ├── trending_raw/         # Cached raw trending data
│   ├── stars_raw/            # Cached star history data
│   ├── history.json          # Cross-run historical tracking
│   └── summaries/            # Summary backups per report
└── reports/                  # Generated markdown reports
```

## 🔨 Building

```bash
go build -o ai-repo-insights ./cmd/ai-repo-insights
```

Requires Go 1.24+.

## 📦 Dependencies

| Package | Purpose |
|---------|---------|
| [github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery) | HTML parsing for web scraping |
| [github.com/andygrunwald/go-trending](https://github.com/andygrunwald/go-trending) | GitHub trending page client |
| [github.com/rs/zerolog](https://github.com/rs/zerolog) | Structured logging |

## 📄 License

MIT
