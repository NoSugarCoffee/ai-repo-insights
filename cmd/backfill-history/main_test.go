package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateReportsDir(t *testing.T) {
	if err := validateReportsDir(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected a missing reports directory error")
	}

	file := filepath.Join(t.TempDir(), "reports")
	if err := os.WriteFile(file, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateReportsDir(file); err == nil {
		t.Fatal("expected a non-directory reports path error")
	}

	if err := validateReportsDir(t.TempDir()); err != nil {
		t.Fatalf("valid directory rejected: %v", err)
	}
}
