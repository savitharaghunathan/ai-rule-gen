package groundtruth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFromGuide(t *testing.T) {
	guide := `# Migration Guide

Migrate from ` + "`org.apache.http.client.methods.HttpGet`" + ` to the new API.

In code blocks:
` + "```java" + `
import org.apache.http.impl.client.HttpClients;
import org.apache.http.client.config.RequestConfig;
` + "```" + `

Also references org.apache.http.cookie.CookieSpecs in the text.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	os.WriteFile(path, []byte(guide), 0o644)

	entries, err := ExtractFromGuide(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fqns := make(map[string]bool)
	for _, e := range entries {
		fqns[e.OldAPI] = true
	}

	want := []string{
		"org.apache.http.client.methods.HttpGet",
		"org.apache.http.impl.client.HttpClients",
		"org.apache.http.client.config.RequestConfig",
		"org.apache.http.cookie.CookieSpecs",
	}
	for _, w := range want {
		if !fqns[w] {
			t.Errorf("missing expected FQN: %s", w)
		}
	}

	for _, e := range entries {
		if e.ActionType != "package_change" {
			t.Errorf("entry %s: action_type=%q, want package_change", e.OldAPI, e.ActionType)
		}
		if e.ReviewedBy != "guide-extract" {
			t.Errorf("entry %s: reviewed_by=%q, want guide-extract", e.OldAPI, e.ReviewedBy)
		}
	}
}

func TestExtractFromGuideDedup(t *testing.T) {
	guide := `
Use org.apache.http.client.methods.HttpGet for requests.
Also see org.apache.http.client.methods.HttpGet in the examples.
` + "```java" + `
import org.apache.http.client.methods.HttpGet;
` + "```" + `
`
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	os.WriteFile(path, []byte(guide), 0o644)

	entries, err := ExtractFromGuide(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := 0
	for _, e := range entries {
		if e.OldAPI == "org.apache.http.client.methods.HttpGet" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("HttpGet appeared %d times, want 1", count)
	}
}

func TestExtractFromGuideNoFQNs(t *testing.T) {
	guide := `# Simple Guide

This guide has no Java FQNs, just plain text about migration steps.
Use the new HttpClient class instead of the old one.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	os.WriteFile(path, []byte(guide), 0o644)

	entries, err := ExtractFromGuide(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestExtractFromGuideFileNotFound(t *testing.T) {
	_, err := ExtractFromGuide("/nonexistent/guide.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
