# RSS 周报 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish a schema-compatible Chinese digest for the previous completed Asia/Shanghai week from the owner's live OPML subscriptions.

**Architecture:** A focused `rssweekly` package owns input parsing, time windows, collection, schema, and output. A small orchestration package injects network and LLM boundaries, while a sibling CLI wires the existing configuration and credentials. The existing workflow runs radar and RSS in one publication job and emits the existing dispatch.

**Tech Stack:** Go 1.25, standard `encoding/xml`/`net/http`, `github.com/mmcdole/gofeed`, existing OpenAI/Gemini-compatible LLM client, GitHub Actions.

## Global Constraints

- Required schema identifier: `rss-weekly/1`.
- Source OPML: `https://raw.githubusercontent.com/NoSugarCoffee/rss-feeds/main/feeds.opml`, fetched on every run.
- Week timezone: `Asia/Shanghai`; scheduled mode selects the previous completed Monday-through-Sunday week.
- Groups follow OPML order; items sort by publication descending and deduplicate globally by URL.
- Individual feed failures are serialized and do not fail the run.
- Empty weeks publish valid groups with empty item arrays.
- Scheduled publication dispatch event remains `radar-updated`.

---

### Task 1: OPML and week primitives

**Files:**
- Create: `internal/rssweekly/rssweekly.go`
- Test: `internal/rssweekly/rssweekly_test.go`

**Interfaces:**
- Produces: `ParseOPML(io.Reader) ([]Feed, []string, error)`, `WeekFor(time.Time, WeekMode) (Week, error)`, and schema structs.

- [ ] **Step 1: Write failing tests**

Add table tests asserting nested outlines become ordered `Feed{Category, Source, XMLURL}` values, category title fallback works, `2027-01-01` belongs to `2026-W53`, and previous mode from Monday selects the prior Shanghai week.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/rssweekly -run 'Test(ParseOPML|WeekFor)'`
Expected: FAIL because the package API does not exist.

- [ ] **Step 3: Implement minimal primitives**

Use recursive XML outline traversal, inheriting the nearest category outline. Calculate Monday with Shanghai-local calendar arithmetic and derive the ID from `start.ISOWeek()`.

- [ ] **Step 4: Verify green**

Run: `go test ./internal/rssweekly`
Expected: PASS.

### Task 2: Feed collection and JSON output

**Files:**
- Modify: `internal/rssweekly/rssweekly.go`
- Modify: `internal/rssweekly/rssweekly_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Consumes: ordered feeds, categories, and `Week`.
- Produces: `Collect(context.Context, ..., FeedParser)`, `BuildDigest(...)`, and `WriteDigest(string, Digest)`.

- [ ] **Step 1: Write failing tests**

Use local HTTP RSS/Atom fixtures to assert in-window filtering, updated-time fallback, global URL deduplication, descending sorting, empty categories, and a failed feed entry while successful feeds remain.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/rssweekly`
Expected: FAIL because collection APIs do not exist.

- [ ] **Step 3: Add the parser dependency**

Run: `go get github.com/mmcdole/gofeed@latest`
Expected: `go.mod` and `go.sum` include the latest resolved release.

- [ ] **Step 4: Implement collection and output**

Adapt `gofeed.Parser.ParseURLWithContext`, selecting `PublishedParsed` then `UpdatedParsed`; normalize times to Shanghai, deduplicate by trimmed link, and use temporary-file-plus-link publication to refuse replacement.

- [ ] **Step 5: Verify green**

Run: `go test ./internal/rssweekly`
Expected: PASS.

### Task 3: LLM summaries

**Files:**
- Create: `internal/llm/rss.go`
- Create: `internal/llm/rss_test.go`

**Interfaces:**
- Produces: `(*llm.Client).WriteRSSSummaries([]rssweekly.Article) (map[string]string, error)`.

- [ ] **Step 1: Write failing tests**

Stub the LLM HTTP endpoint and assert URL-keyed Chinese sentence JSON is parsed, missing/blank summaries are rejected, untrusted article content is delimited as data, empty input makes no API call, and transient failures use existing retries.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/llm -run RSS`
Expected: FAIL because `WriteRSSSummaries` does not exist.

- [ ] **Step 3: Implement batched summaries**

Send bounded article batches containing title and sanitized description/content excerpts. Require one non-empty, single-line summary per URL and merge batch results.

- [ ] **Step 4: Verify green**

Run: `go test ./internal/llm -run RSS`
Expected: PASS.

### Task 4: Pipeline and CLI

**Files:**
- Create: `internal/rsspipeline/orchestrator.go`
- Create: `internal/rsspipeline/orchestrator_test.go`
- Create: `cmd/rss-weekly/main.go`

**Interfaces:**
- Produces: `rsspipeline.Run(context.Context, Deps, zerolog.Logger) (string, error)` and CLI flags `-config`, `-out`, `-opml`, `-week`, `-stub-summaries`, `-log-level`.

- [ ] **Step 1: Write failing pipeline tests**

Assert an empty week writes a file without invoking summaries, a failed feed is retained, and populated items receive summaries before writing.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/rsspipeline`
Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement orchestration and CLI wiring**

Fetch OPML using a timeout/User-Agent client, calculate `previous` by default or `current` when selected, collect feeds, summarize only populated articles, and write the digest.

- [ ] **Step 4: Verify green**

Run: `go test ./internal/rsspipeline && go build ./cmd/rss-weekly`
Expected: both commands exit 0.

### Task 5: Workflow, documentation, and sample

**Files:**
- Modify: `.github/workflows/weekly-report.yml`
- Modify: `README.md`
- Create: `data/rss-weekly/2026-W47.json`

**Interfaces:**
- Scheduled run invokes `rss-weekly -week previous`; publication stages both data directories and sends `radar-updated`.

- [ ] **Step 1: Add workflow assertions**

Extend repository checks to verify the workflow builds/runs the RSS command, stages `data/rss-weekly`, and retains `radar-updated`.

- [ ] **Step 2: Modify workflow and README**

Document the path contract, Monday previous-week behavior, manual current mode, failure handling, and shared dispatch.

- [ ] **Step 3: Generate current partial-week sample**

Run `go run ./cmd/rss-weekly -week current` when `LLM_API_KEY` is available. Otherwise run `go run ./cmd/rss-weekly -week current -stub-summaries`, which uses the same pipeline with a deterministic stub summarizer and sets optional `"summary_mode": "stub"` on the sample only.

- [ ] **Step 4: Verify all outputs**

Run: `gofmt -w cmd/rss-weekly internal/rssweekly internal/rsspipeline internal/llm/rss.go internal/llm/rss_test.go && go test ./... && go vet ./... && go build ./... && git diff --check`
Expected: all commands exit 0.

- [ ] **Step 5: Commit**

```bash
git add .github README.md cmd internal go.mod go.sum data/rss-weekly docs/superpowers
git commit -m "feat: add weekly RSS digest"
```
