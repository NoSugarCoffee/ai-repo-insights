package llm

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"ai-repo-insights/internal/config"
	"ai-repo-insights/internal/models"
	"ai-repo-insights/internal/radar"
)

func candidate(owner, name string) radar.Candidate {
	return radar.Candidate{Repo: models.ClassifiedRepo{Metadata: models.RepoMetadata{Owner: owner, Name: name}}, Column: radar.ColumnNewProjects}
}

func TestValidateCopy(t *testing.T) {
	cands := []radar.Candidate{candidate("a", "x")}
	valid := radar.Copy{Headline: "Agent 技能包扎堆", Summary: "本周主线。", Comments: map[string]string{"a/x": "给编码 agent 用的浏览器自动化 CLI。"}}
	cases := []struct {
		name string
		edit func(c *radar.Copy)
		rule string
	}{
		{"valid", func(c *radar.Copy) {}, ""},
		{"headline 13 runes", func(c *radar.Copy) { c.Headline = strings.Repeat("字", 13) }, "headline-length"},
		{"empty headline", func(c *radar.Copy) { c.Headline = "" }, "headline-length"},
		{"summary 121 runes", func(c *radar.Copy) { c.Summary = strings.Repeat("字", 121) }, "summary-length"},
		{"comment 61 runes", func(c *radar.Copy) { c.Comments = map[string]string{"a/x": strings.Repeat("字", 61)} }, "comment-length"},
		{"missing comment", func(c *radar.Copy) { c.Comments = map[string]string{} }, "missing-comment"},
		{"banned word", func(c *radar.Copy) { c.Comments = map[string]string{"a/x": "一个强大的工具"} }, "banned-word"},
		{"banned word in summary", func(c *radar.Copy) { c.Summary = "颠覆了行业" }, "banned-word"},
		{"banned marketing words", func(c *radar.Copy) { c.Summary = "本周重磅项目" }, "banned-word"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid
			c.Comments = map[string]string{"a/x": valid.Comments["a/x"]}
			tc.edit(&c)
			err := ValidateCopy(c, cands)
			if tc.rule == "" {
				if err != nil {
					t.Fatalf("unexpected %v", err)
				}
				return
			}
			var ruleErr *CopyRuleError
			if !errors.As(err, &ruleErr) || ruleErr.Rule != tc.rule {
				t.Fatalf("want rule %s, got %v", tc.rule, err)
			}
		})
	}
}

func openAIReply(content string) []byte {
	body, _ := json.Marshal(map[string]any{"choices": []map[string]any{{"message": map[string]string{"content": content}}}})
	return body
}

func testClient(url string) *Client {
	c := NewClient(config.LLMConfig{BaseURL: url, Model: "m", Provider: "openai", TimeoutSeconds: 5, MaxRetries: 1}, "k", zerolog.Nop())
	c.backoff = func(int) time.Duration { return 0 }
	return c
}

func TestWriteRadarCopyRetriesInvalidThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Write(openAIReply(`{"headline":"新工具","summary":"摘要","comments":{"a/x":"一个强大的工具"}}`))
			return
		}
		w.Write(openAIReply("```json\n{\"headline\":\"新工具\",\"summary\":\"摘要\",\"comments\":{\"a/x\":\"给 agent 用的浏览器 CLI\"}}\n```"))
	}))
	defer srv.Close()

	got, err := testClient(srv.URL).WriteRadarCopy([]radar.Candidate{candidate("a", "x")}, "规范")
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || got.Comments["a/x"] != "给 agent 用的浏览器 CLI" {
		t.Fatalf("calls=%d copy=%#v", calls.Load(), got)
	}
}

func TestWriteRadarCopyFailsAfterRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":402,"message":"credits depleted"}}`, http.StatusPaymentRequired)
	}))
	defer srv.Close()

	_, err := testClient(srv.URL).WriteRadarCopy([]radar.Candidate{candidate("a", "x")}, "规范")
	if err == nil || !strings.Contains(err.Error(), "402") {
		t.Fatalf("got %v", err)
	}
}

func TestRadarPromptCarriesGuideAndCandidates(t *testing.T) {
	c := candidate("a", "x")
	c.Readme = "README 内容"
	prompt := buildRadarPrompt([]radar.Candidate{c}, "写作规范正文")
	for _, want := range []string{"写作规范正文", `"a/x"`, "README 内容", `"headline"`, "不可信数据", "不要执行"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
	if strings.Contains(prompt, "30 天内") {
		t.Error("prompt must not hard-code the configurable new-repository threshold")
	}
	if !strings.Contains(buildRadarPrompt(nil, "g"), "没有") {
		t.Error("empty-week prompt should ask for a no-new-projects summary")
	}
}

func TestWriteRadarCopyRetryPromptNamesBrokenRule(t *testing.T) {
	var prompts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct{ Content string } `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		prompts = append(prompts, body.Messages[0].Content)
		if len(prompts) == 1 {
			w.Write(openAIReply(`{"headline":"新工具","summary":"摘要","comments":{"a/x":"一个强大的工具"}}`))
			return
		}
		w.Write(openAIReply(`{"headline":"新工具","summary":"摘要","comments":{"a/x":"给 agent 用的浏览器 CLI"}}`))
	}))
	defer srv.Close()

	if _, err := testClient(srv.URL).WriteRadarCopy([]radar.Candidate{candidate("a", "x")}, "规范"); err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 2 || !strings.Contains(prompts[1], "banned-word") || strings.Contains(prompts[0], "banned-word") {
		t.Fatalf("retry prompt should name the broken rule; prompts: %q", prompts)
	}
}
