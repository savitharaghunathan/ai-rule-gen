package ingestion

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIngest_RawText(t *testing.T) {
	result, err := Ingest(context.Background(), "This is raw migration guide text about javax.ejb.", 0)
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

	result, err := Ingest(context.Background(), path, 0)
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

	_, err := Ingest(context.Background(), path, 0)
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
		got := DetectType(tt.input)
		if got != tt.want {
			t.Errorf("DetectType(%q): got %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestValidateURLHost_BlocksLoopback(t *testing.T) {
	err := validateURLHost("http://127.0.0.1/secret")
	if err == nil {
		t.Error("expected error for loopback address")
	}
}

func TestValidateURLHost_BlocksLocalhost(t *testing.T) {
	err := validateURLHost("http://localhost/secret")
	if err == nil {
		t.Error("expected error for localhost")
	}
}

func TestValidateURLHost_AllowsPublic(t *testing.T) {
	// google.com resolves to public IPs
	err := validateURLHost("https://google.com")
	if err != nil {
		t.Errorf("unexpected error for public host: %v", err)
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
