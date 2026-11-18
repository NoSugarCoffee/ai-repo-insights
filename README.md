<div align="center">

<img src="logo.png" alt="ai-repo-insights" width="512"/>

[![Go Version](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-2.0.0-green.svg)](cmd/ai-repo-insights/main.go)

**🔭 Automatically track, rank, and report on GitHub's trending repositories — filtered to topics that matter to you 📊**

</div>

---

## ⚡ Quick Start

```bash
# 1. Clone and build
git clone https://github.com/NoSugarCoffee/ai-repo-insights.git
cd ai-repo-insights
go build -o ai-repo-insights ./cmd/ai-repo-insights

# 2. Set environment variables (both required)
export GITHUB_TOKEN=ghp_...   # repository details and README excerpts
export LLM_API_KEY=your-google-ai-key  # Gemini key for the default config

# 3. Run
./ai-repo-insights
```

Each radar run writes one issue of the **AI 开源雷达** to `data/weekly/<YYYY-Www>.json` (ISO week, e.g. `2026-W47`) and records every repository it saw in `data/history.json`. The site [NoSugarCoffee.github.io](https://nosugarcoffee.github.io/radar) renders the issues at `/radar`.

A sibling command writes **RSS 周报** to `data/rss-weekly/<YYYY-Www>.json` from the live OPML at [NoSugarCoffee/rss-feeds](https://github.com/NoSugarCoffee/rss-feeds). The site renders it under `/radar/rss`.

---

## ✨ How an issue is built

1. **Fetch** GitHub trending (daily, weekly and monthly pages) for the configured languages, and keep the repositories that match the AI keywords.
2. **Skip** every repository already in `data/history.json`: each repository is shown once, in the week it first surfaces.
3. **Look up** each remaining repository through the GitHub REST API: creation date, stars, topics and the first 4000 characters of its README.
4. **Split** them into two columns:
   - 🆕 `newProjects`: created within `new_repo_threshold_days` (30) of the run;
   - ↗ `newEntries`: older repositories on the list for the first time (at most 15 shown).
5. **Write** the headline, summary and one comment per repository with the LLM, following `config/writing-guide.md`. The copy is checked for length limits and banned words, and retried if it fails.
6. **Save** the issue, then update history.

**If the LLM call fails** (including exhausted credits), **the run fails and publishes nothing**. There is no template fallback. Fix the key or credits and re-run; history is untouched by a failed run.

### Issue format

```json
{
  "schemaVersion": 1,
  "id": "2026-W47", "year": 2026, "week": 47,
  "generatedAt": "2026-11-16T04:46:00Z",
  "headline": "Agent 技能包扎堆",
  "summary": "…",
  "newProjects": [{
    "source": "github", "id": "owner/repo", "url": "https://github.com/owner/repo",
    "name": "repo", "description": "…", "comment": "…", "language": "typescript",
    "createdAt": "2026-10-30T00:00:00Z", "stars": 19546,
    "starsThisWeek": 4234, "starsThisMonth": null
  }],
  "newEntries": []
}
```

`starsThisWeek` and `starsThisMonth` are `null` when the scraper reports zero or cannot parse a positive delta. This normally means the repository was absent from that trending page; an actual zero and an unknown value are not distinguished.

---

## 🚀 Usage

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `config` | Path to configuration directory |
| `-out` | `data/weekly` | Where the issue JSON is written |
| `-history` | `data/history.json` | Repository history file |
| `-log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `-version` | — | Print version and exit |

### Scheduled run

`.github/workflows/weekly-report.yml` runs every Monday 00:00 UTC (and on manual dispatch). In one job it generates the radar issue and the RSS 周报, commits both data directories plus history when they change, then sends a `radar-updated` repository dispatch to the site so it redeploys. Keep listening for `radar-updated`; there is no separate RSS event.

The RSS scheduled run uses the just-completed Asia/Shanghai week (Monday 00:00 through Sunday 24:00). That week id can differ from the radar issue id written in the same workflow run; they are independent. Manual sample generation can use `-week current` for the in-progress Shanghai week. `DEPLOY_TOKEN` should be a fine-grained personal access token restricted to `NoSugarCoffee/NoSugarCoffee.github.io`, with repository **Contents: Read and write** permission. If publication succeeds but dispatch fails, manually run the workflow with `dispatch_only` enabled; this retries the dispatch without regenerating the non-overwritable weekly files.

### RSS 周报

```bash
go build -o rss-weekly ./cmd/rss-weekly
export LLM_API_KEY=...   # required when the selected week has articles
./rss-weekly                 # previous completed Shanghai week (default)
./rss-weekly -week current   # current partial Shanghai week
```

Each file follows schema `rss-weekly/1` with snake_case fields (`summary_zh`, `published_at`, …), date-only `week_start`/`week_end`, ISO timestamps for `generated_at`/`published_at`, OPML category groups in source order, and `failed_feeds` for subscriptions that could not be fetched. An empty week still writes a valid file with empty groups.

### Backfilling history

`go run ./cmd/backfill-history -reports reports -history data/history.json` records the first-seen week of every repository in the old Markdown reports under `reports/`. It was run once when the radar replaced those reports.

---

## ⚙️ Configuration

All configuration lives in JSON files under `config/` (or a custom directory via `-config`).

| File | Description |
|------|-------------|
| `languages.json` | Programming languages to track |
| `keywords.json` | Include/exclude keywords and category mappings |
| `settings.json` | Operational parameters (`top_n`, `window_days`, `report_language`, etc.) |
| `llm.json` | LLM API settings (`base_url`, `model`, `provider`, `temperature`, `session_header`, etc.). The default uses OpenCode Go (`glm-5.3`), which requires the `x-opencode-session` header |
| `writing-guide.md` | Writing rules sent verbatim to the LLM; required |

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

Each example directory contains every required configuration file. See [examples/README.md](examples/README.md) for how to create custom domain configs.

---

## 🗂️ Project Structure

```
ai-repo-insights/
├── cmd/
│   ├── ai-repo-insights/     # Weekly radar run
│   ├── rss-weekly/           # Weekly RSS digest
│   └── backfill-history/     # One-off history backfill from old reports
├── internal/
│   ├── config/               # Configuration and writing guide loading
│   ├── fetcher/              # GitHub trending scraper
│   ├── classifier/           # Keyword-based AI filter
│   ├── githubapi/            # GitHub REST client (created_at, README)
│   ├── history/              # First-seen tracking
│   ├── radar/                # Selection, issue model, JSON output
│   ├── rssweekly/            # OPML/feed collection and RSS digest schema
│   ├── rsspipeline/          # RSS digest run order
│   ├── llm/                  # LLM client (OpenAI-compatible / Gemini) and copy rules
│   └── pipeline/             # Radar run order
├── config/                   # Default configuration and writing guide
├── data/
│   ├── weekly/               # One radar JSON file per issue
│   ├── rss-weekly/           # One RSS digest JSON file per week
│   └── history.json          # Every repository ever shown
└── reports/                  # Markdown reports up to 2026-W47 (no longer generated)
```

## 🔨 Building

```bash
go build -o ai-repo-insights ./cmd/ai-repo-insights
```

Requires Go 1.25+.

## 📦 Dependencies

| Package | Purpose |
|---------|---------|
| [github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery) | HTML parsing for web scraping |
| [github.com/mmcdole/gofeed](https://github.com/mmcdole/gofeed) | RSS and Atom feed parsing |
| [github.com/rs/zerolog](https://github.com/rs/zerolog) | Structured logging |

## 📄 License

MIT
