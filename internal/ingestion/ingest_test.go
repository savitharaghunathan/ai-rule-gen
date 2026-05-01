package ingestion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIngest_RawText(t *testing.T) {
	result, err := Ingest("This is raw migration guide text about javax.ejb.", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Source != InputText {
		t.Errorf("source: got %v, want InputText", result.Source)
	}
	if result.Content != "This is raw migration guide text about javax.ejb." {
		t.Errorf("content mismatch: %q", result.Content)
	}
	if len(result.Chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(result.Chunks))
	}
}

func TestIngest_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(path, []byte("# Migration Guide\n\nReplace javax.ejb with CDI."), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Ingest(path, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Source != InputFile {
		t.Errorf("source: got %v, want InputFile", result.Source)
	}
	if result.Content == "" {
		t.Error("empty content")
	}
}

func TestIngest_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(path, []byte("   \n  \n  "), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Ingest(path, 0)
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestDetectType(t *testing.T) {
	tests := []struct {
		input string
		want  InputType
	}{
		{"https://example.com/guide", InputURL},
		{"http://example.com/guide", InputURL},
		{"just some text", InputText},
	}

	for _, tt := range tests {
		got := detectType(tt.input)
		if got != tt.want {
			t.Errorf("detectType(%q): got %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestHTMLToMarkdown(t *testing.T) {
	html := `<h1>Title</h1><p>Replace <code>javax.ejb</code> with CDI.</p>`
	md, err := HTMLToMarkdown(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md == "" {
		t.Error("empty markdown output")
	}
}

func TestChunk_SmallContent(t *testing.T) {
	content := "small content"
	chunks := Chunk(content, 1000)
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != content {
		t.Errorf("chunk content mismatch")
	}
}

func TestExtractArticle_WithArticleTag(t *testing.T) {
	raw := `<html><body>
		<nav><a href="/">Home</a><a href="/blog">Blog</a></nav>
		<article><h1>Migration Guide</h1><p>Replace javax with jakarta.</p></article>
		<footer><p>Copyright 2024</p></footer>
	</body></html>`

	result := ExtractArticle(raw)
	if !containsStr(result, "Migration Guide") {
		t.Error("expected article content")
	}
	if containsStr(result, "Home") {
		t.Error("nav content should be stripped")
	}
	if containsStr(result, "Copyright") {
		t.Error("footer content should be stripped")
	}
}

func TestExtractArticle_WithMainTag(t *testing.T) {
	raw := `<html><body>
		<nav><ul><li>Nav item</li></ul></nav>
		<main><h2>Step 1</h2><p>Do the thing.</p></main>
		<aside><p>Related posts</p></aside>
	</body></html>`

	result := ExtractArticle(raw)
	if !containsStr(result, "Step 1") {
		t.Error("expected main content")
	}
	if containsStr(result, "Nav item") {
		t.Error("nav content should be stripped")
	}
	if containsStr(result, "Related posts") {
		t.Error("aside content should be stripped")
	}
}

func TestExtractArticle_NoArticleOrMain(t *testing.T) {
	raw := `<html><body>
		<nav><a href="/">Home</a></nav>
		<div><h1>Guide</h1><p>Content here.</p></div>
		<footer><p>Footer</p></footer>
		<script>alert("hi")</script>
	</body></html>`

	result := ExtractArticle(raw)
	if !containsStr(result, "Guide") {
		t.Error("expected body content")
	}
	if containsStr(result, "Home") {
		t.Error("nav should be stripped")
	}
	if containsStr(result, "Footer") {
		t.Error("footer should be stripped")
	}
	if containsStr(result, "alert") {
		t.Error("script should be stripped")
	}
}

func TestExtractArticle_InvalidHTML(t *testing.T) {
	raw := "not html at all, just text"
	result := ExtractArticle(raw)
	if result == "" {
		t.Error("should return something for invalid HTML")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}

func TestChunk_LargeContent(t *testing.T) {
	// Build content with multiple sections
	var content string
	for i := 0; i < 10; i++ {
		content += "## Section " + string(rune('A'+i)) + "\n\n"
		content += "This is content for section. " + string(make([]byte, 100)) + "\n\n"
	}

	chunks := Chunk(content, 300)
	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks, got %d", len(chunks))
	}
	for i, chunk := range chunks {
		if len(chunk) > 300+100 { // allow some margin for section boundaries
			t.Errorf("chunk %d too large: %d chars", i, len(chunk))
		}
	}
}
