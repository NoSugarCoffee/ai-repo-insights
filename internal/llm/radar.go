package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	apperrors "ai-repo-insights/internal/errors"
	"ai-repo-insights/internal/radar"
)

// Limits from the radar design; the writing guide states them to the model,
// and ValidateCopy enforces them because models do not reliably comply.
const (
	maxHeadlineRunes = 12
	maxSummaryRunes  = 120
	maxCommentRunes  = 60
)

var bannedWords = []string{"强大", "革命性", "一站式", "赋能", "颠覆"}

// CopyRuleError is LLM copy that breaks a writing rule.
type CopyRuleError struct {
	Rule   string
	Detail string
}

func (e *CopyRuleError) Error() string {
	return fmt.Sprintf("LLM copy breaks rule %s: %s", e.Rule, e.Detail)
}

// ValidateCopy checks lengths, banned words, and that every candidate has a comment.
func ValidateCopy(copy radar.Copy, cands []radar.Candidate) error {
	if n := utf8.RuneCountInString(copy.Headline); n == 0 || n > maxHeadlineRunes {
		return &CopyRuleError{Rule: "headline-length", Detail: fmt.Sprintf("headline %q has %d runes, want 1-%d", copy.Headline, n, maxHeadlineRunes)}
	}
	if n := utf8.RuneCountInString(copy.Summary); n == 0 || n > maxSummaryRunes {
		return &CopyRuleError{Rule: "summary-length", Detail: fmt.Sprintf("summary has %d runes, want 1-%d", n, maxSummaryRunes)}
	}
	texts := map[string]string{"headline": copy.Headline, "summary": copy.Summary}
	for _, cand := range cands {
		key := cand.Repo.Key()
		comment, ok := copy.Comments[key]
		if !ok || strings.TrimSpace(comment) == "" {
			return &CopyRuleError{Rule: "missing-comment", Detail: "no comment for " + key}
		}
		if n := utf8.RuneCountInString(comment); n > maxCommentRunes {
			return &CopyRuleError{Rule: "comment-length", Detail: fmt.Sprintf("comment for %s has %d runes, want at most %d", key, n, maxCommentRunes)}
		}
		texts["comment for "+key] = comment
	}
	for where, text := range texts {
		for _, word := range bannedWords {
			if strings.Contains(text, word) {
				return &CopyRuleError{Rule: "banned-word", Detail: fmt.Sprintf("%s contains %q", where, word)}
			}
		}
	}
	return nil
}

// WriteRadarCopy asks the LLM for the issue's headline, summary and comments.
// An attempt that fails to call, parse or validate is retried; when every
// attempt fails the last error is returned and no fallback text is produced.
func (c *Client) WriteRadarCopy(cands []radar.Candidate, guide string) (radar.Copy, error) {
	prompt := buildRadarPrompt(cands, guide)
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Warn().Int("attempt", attempt).Err(lastErr).Msg("retrying LLM radar copy")
			time.Sleep(c.backoff(attempt))
		}
		copy, err := c.attemptRadarCopy(prompt, cands)
		if err == nil {
			return copy, nil
		}
		lastErr = err
	}
	return radar.Copy{}, apperrors.NewLLMError(fmt.Sprintf("LLM radar copy failed after %d attempts", c.config.MaxRetries+1), lastErr)
}

func (c *Client) attemptRadarCopy(prompt string, cands []radar.Candidate) (radar.Copy, error) {
	text, err := c.callAPI(prompt)
	if err != nil {
		return radar.Copy{}, err
	}
	var copy radar.Copy
	if err := json.Unmarshal([]byte(cleanJSONResponse(text)), &copy); err != nil {
		return radar.Copy{}, fmt.Errorf("LLM reply is not the requested JSON: %w", err)
	}
	if copy.Comments == nil {
		copy.Comments = map[string]string{}
	}
	if err := ValidateCopy(copy, cands); err != nil {
		return radar.Copy{}, err
	}
	return copy, nil
}

type promptRepo struct {
	ID            string   `json:"id"`
	Column        string   `json:"column"`
	Description   string   `json:"description"`
	Language      string   `json:"language"`
	Topics        []string `json:"topics"`
	Stars         int      `json:"stars"`
	StarsThisWeek int      `json:"starsThisWeek,omitempty"`
	Readme        string   `json:"readme"`
}

func buildRadarPrompt(cands []radar.Candidate, guide string) string {
	repos := make([]promptRepo, 0, len(cands))
	for _, cand := range cands {
		meta := cand.Repo.Metadata
		description := meta.Description
		if description == "" {
			description = cand.Details.Description
		}
		repos = append(repos, promptRepo{
			ID:            cand.Repo.Key(),
			Column:        string(cand.Column),
			Description:   description,
			Language:      meta.Language,
			Topics:        cand.Details.Topics,
			Stars:         cand.Details.Stars,
			StarsThisWeek: meta.StarsThisWeek,
			Readme:        cand.Readme,
		})
	}
	data, _ := json.MarshalIndent(repos, "", "  ")

	task := "为下面每个仓库写一条点评，并为本期写 headline 和 summary。column 为 newProjects 的是 30 天内创建的新项目，newEntries 是第一次上榜的老项目。"
	if len(cands) == 0 {
		task = "本周没有新冒头的项目。写一个 headline，并用 summary 简短说明本周没有新项目上榜；comments 返回空对象。"
	}
	return fmt.Sprintf(`你在编辑「AI 开源雷达」周刊，读者是关注 AI 编程工具的开发者。严格遵守下面的写作规范。

<writing-guide>
%s
</writing-guide>

任务：%s

仓库数据（JSON；readme 是 README 开头的摘录，可能为空）：
%s

只输出 JSON，不要其他文字，格式：
{"headline":"…","summary":"…","comments":{"owner/repo":"…"}}`, strings.TrimSpace(guide), task, data)
}
