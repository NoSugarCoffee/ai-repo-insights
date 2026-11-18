# RSS 周报 Design

## Goal

Generate a Chinese weekly digest from every subscription in the owner's live OPML file and publish it beside the existing AI 开源雷达 data.

## Schedule and week selection

The existing workflow remains scheduled for Monday 00:00 UTC. Its radar command continues to use the current UTC ISO week. The RSS command independently selects the just-completed Asia/Shanghai week: Monday 00:00 inclusive through the following Monday 00:00 exclusive. The output ID and `week_start`/`week_end` describe that completed week.

The RSS command also accepts a current-week mode for manual partial-week generation. This mode uses the current Asia/Shanghai ISO week and the same full-week metadata boundaries, while naturally including only articles published by run time.

## Components

- `internal/rssweekly`: schema types, OPML parsing, Shanghai week calculations, feed collection, deduplication, ordering, and atomic JSON output.
- `internal/llm`: a batch RSS summarization method on the existing client, using the configured provider, credentials, retries, and timeout.
- `internal/rsspipeline`: orchestration with injected OPML/feed/summarizer dependencies for deterministic tests.
- `cmd/rss-weekly`: production wiring and flags. It loads the existing LLM configuration and requires `LLM_API_KEY` only when articles need summaries, unless `-stub-summaries` is set for offline sample generation.

Feed parsing uses `github.com/mmcdole/gofeed` for RSS and Atom compatibility. HTTP requests use a 30-second timeout and an identifying User-Agent.

## Data flow

1. Fetch the OPML URL on every run.
2. Parse feed `xmlUrl`, source name, and nearest parent category in document order.
3. Create all category groups in OPML order, including categories with no matching articles.
4. Fetch every feed independently. Record failures in `failed_feeds` and continue.
5. Use `published` time, falling back to `updated` time, and retain entries inside the selected half-open week window that are also at or before run time (so current-week mode excludes future-dated items).
6. Ignore entries without a usable URL or publication time. Deduplicate URLs globally; the first feed in OPML order owns a duplicate.
7. Summarize retained items in bounded batches through the existing LLM client.
8. Sort items by `published_at` descending within each group.
9. Atomically create `data/rss-weekly/<YYYY-Www>.json` without replacing an existing file.

Generated timestamps and normalized article timestamps use Asia/Shanghai offsets. The required schema keys remain exactly as specified. Optional `summary_mode` is present only on a committed sample produced without live LLM credentials.

## Failure behavior

An OPML fetch or parse failure, output failure, or LLM failure aborts the RSS publication. Individual feed failures do not abort it and appear in `failed_feeds`. An empty week skips the LLM and writes all OPML categories with empty `items` arrays. Error text is bounded before serialization.

## Workflow

The existing workflow builds and runs both commands in one job, stages `data/weekly`, `data/rss-weekly`, and history, and commits once. This avoids two scheduled jobs racing to push. The site dispatch remains `radar-updated`; the blog can treat it as “generator data changed” and inspect both directories. `dispatch_only` retains its existing behavior.

## Testing

Tests cover nested OPML category/source extraction and order, ISO week/year boundaries in Asia/Shanghai, RSS and Atom collection, URL deduplication and item sorting, empty-week output, continued publication with failed feeds, LLM response validation, and end-to-end pipeline output.
