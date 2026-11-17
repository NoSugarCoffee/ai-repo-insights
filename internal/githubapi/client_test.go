package githubapi_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"ai-repo-insights/internal/githubapi"
)

func serve(t *testing.T, handler http.HandlerFunc) *githubapi.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return githubapi.NewClient(srv.URL, "tkn", srv.Client())
}

func TestRepoParsesCreatedAtAndStars(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/a/x" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Write([]byte(`{"created_at":"2026-10-30T00:00:00Z","stargazers_count":19546,"topics":["agent"],"description":"d"}`))
	})
	got, err := c.Repo(context.Background(), "a/x")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	if !got.CreatedAt.Equal(want) || got.Stars != 19546 || len(got.Topics) != 1 || got.Topics[0] != "agent" || got.Description != "d" {
		t.Fatalf("got %#v", got)
	}
}

func TestReadmeDecodesAndTruncates(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("汉", 5000)))
	// GitHub wraps base64 content at 60 columns.
	var wrapped strings.Builder
	for i := 0; i < len(encoded); i += 60 {
		end := min(i+60, len(encoded))
		wrapped.WriteString(encoded[i:end] + "\n")
	}
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/a/x/readme" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Write([]byte(`{"encoding":"base64","content":"` + strings.ReplaceAll(wrapped.String(), "\n", `\n`) + `"}`))
	})
	got, err := c.Readme(context.Background(), "a/x")
	if err != nil {
		t.Fatal(err)
	}
	if n := utf8.RuneCountInString(got); n != 4000 {
		t.Fatalf("got %d runes", n)
	}
}

func TestReadmeMissingIsEmpty(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	got, err := c.Readme(context.Background(), "a/x")
	if err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestRepoNon200IsAPIError(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "rate limited", http.StatusForbidden) })
	_, err := c.Repo(context.Background(), "a/x")
	var apiErr *githubapi.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden || apiErr.FullName != "a/x" {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Fatalf("message lacks guidance: %v", err)
	}
}

func TestSendsBearerToken(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tkn" {
			t.Errorf("Authorization %q", got)
		}
		w.Write([]byte(`{"created_at":"2026-10-30T00:00:00Z"}`))
	})
	if _, err := c.Repo(context.Background(), "a/x"); err != nil {
		t.Fatal(err)
	}
}

func TestReadmeTooLargeIsEmpty(t *testing.T) {
	// GitHub omits the content of files over 1 MB and reports encoding "none".
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"encoding":"none","content":""}`))
	})
	got, err := c.Readme(context.Background(), "a/x")
	if err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}
