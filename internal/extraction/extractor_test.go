package extraction

import (
	"context"
	"errors"
	"strings"
	"testing"
	"text/template"
)

// mockCompleter returns preconfigured responses in order.
type mockCompleter struct {
	responses []string
	calls     []string
	index     int
}

func (m *mockCompleter) Complete(_ context.Context, prompt string) (string, error) {
	m.calls = append(m.calls, prompt)
	if m.index >= len(m.responses) {
		return "", nil
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func TestParsePatterns_CleanJSON(t *testing.T) {
	response := `[{"source_pattern":"@Stateless","source_fqn":"javax.ejb.Stateless","rationale":"EJBs removed","complexity":"medium","category":"mandatory","provider_type":"java","location_type":"ANNOTATION"}]`

	patterns, err := parsePatterns(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
	if patterns[0].SourceFQN != "javax.ejb.Stateless" {
		t.Errorf("source_fqn: got %q", patterns[0].SourceFQN)
	}
}

func TestParsePatterns_MarkdownFenced(t *testing.T) {
	response := "```json\n" +
		`[{"source_pattern":"@Stateless","rationale":"removed","complexity":"low","category":"mandatory"}]` +
		"\n```"

	patterns, err := parsePatterns(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
}

func TestParsePatterns_SurroundingText(t *testing.T) {
	response := `Here are the patterns I found:
[{"source_pattern":"InitialContext","rationale":"JNDI not supported","complexity":"medium","category":"mandatory"}]
That's all.`

	patterns, err := parsePatterns(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
}

func TestParsePatterns_NoJSON(t *testing.T) {
	_, err := parsePatterns("I don't see any patterns here.")
	if err == nil {
		t.Error("expected error for response without JSON")
	}
}

func TestParsePatterns_EmptyArray(t *testing.T) {
	_, err := parsePatterns("[]")
	if err == nil {
		t.Error("expected error for empty array")
	}
}

func TestParsePatterns_MalformedJSON(t *testing.T) {
	_, err := parsePatterns(`[{"source_pattern": "foo", "broken`)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestParsePatterns_NullEntries(t *testing.T) {
	// Go unmarshals nulls as zero-value structs, so [null, null] produces
	// 2 empty MigrationPattern objects — parsePatterns does not reject these.
	// This documents the current behavior; callers should validate patterns.
	patterns, err := parsePatterns(`[null, null]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 2 {
		t.Errorf("expected 2 zero-value patterns, got %d", len(patterns))
	}
}

func TestParsePatterns_NestedBrackets(t *testing.T) {
	// JSON with nested arrays in a field should still parse correctly
	response := `[{"source_pattern":"foo","source_fqn":"foo.Bar","rationale":"test","complexity":"low","category":"mandatory","alternative_fqns":["a","b"]}]`
	patterns, err := parsePatterns(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
}

func TestParsePatterns_OnlyClosingBracket(t *testing.T) {
	_, err := parsePatterns("]")
	if err == nil {
		t.Error("expected error for only closing bracket")
	}
}

func TestExtractJSON_NoArray(t *testing.T) {
	result := extractJSON(`{"key": "value"}`)
	if result != "" {
		t.Errorf("expected empty for object-only JSON, got %q", result)
	}
}

func TestExtractJSON_NestedArrays(t *testing.T) {
	input := `Some text [{"a": [1,2,3]}, {"b": [4]}] more text`
	result := extractJSON(input)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
		t.Errorf("result should be bracketed: %q", result)
	}
}

func TestExtractJSON_MarkdownFenceWithLanguage(t *testing.T) {
	input := "```json\n[{\"a\": 1}]\n```"
	result := extractJSON(input)
	if result == "" {
		t.Fatal("expected non-empty result from fenced JSON")
	}
}

func TestDeduplicate(t *testing.T) {
	patterns := []MigrationPattern{
		{SourceFQN: "javax.ejb.Stateless", SourcePattern: "@Stateless"},
		{SourceFQN: "javax.ejb.Stateless", SourcePattern: "@Stateless"}, // duplicate
		{SourceFQN: "javax.ejb.Stateful", SourcePattern: "@Stateful"},
	}

	unique := Deduplicate(patterns)
	if len(unique) != 2 {
		t.Errorf("expected 2 unique patterns, got %d", len(unique))
	}
}

func TestExtractor_Extract(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse("Extract patterns from: {{.Content}}"))

	mock := &mockCompleter{
		responses: []string{
			`[{"source_pattern":"@Stateless","source_fqn":"javax.ejb.Stateless","rationale":"EJBs removed","complexity":"medium","category":"mandatory","provider_type":"java","location_type":"ANNOTATION"}]`,
		},
	}

	extractor := New(mock, tmpl)
	patterns, err := extractor.Extract(context.Background(), []string{"test content"}, "java-ee", "quarkus", "java")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
	if len(mock.calls) != 1 {
		t.Errorf("expected 1 LLM call, got %d", len(mock.calls))
	}
}

func TestExtractor_MultiChunk_Dedup(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse("Extract: {{.Content}}"))

	mock := &mockCompleter{
		responses: []string{
			`[{"source_pattern":"@Stateless","source_fqn":"javax.ejb.Stateless","rationale":"removed","complexity":"medium","category":"mandatory"}]`,
			`[{"source_pattern":"@Stateless","source_fqn":"javax.ejb.Stateless","rationale":"removed","complexity":"medium","category":"mandatory"},
			  {"source_pattern":"@Stateful","source_fqn":"javax.ejb.Stateful","rationale":"also removed","complexity":"medium","category":"mandatory"}]`,
		},
	}

	extractor := New(mock, tmpl)
	patterns, err := extractor.Extract(context.Background(), []string{"chunk1", "chunk2"}, "java-ee", "quarkus", "java")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 1 from chunk1 + 2 from chunk2, but @Stateless is duplicated → 2 unique
	if len(patterns) != 2 {
		t.Errorf("expected 2 deduplicated patterns, got %d", len(patterns))
	}
}

// ---------- DetectMetadata ----------

// errCompleter always returns an error.
type errCompleter struct{ msg string }

func (e *errCompleter) Complete(_ context.Context, _ string) (string, error) {
	return "", errors.New(e.msg)
}

func TestDetectMetadata_Success(t *testing.T) {
	tmpl := template.Must(template.New("detect").Parse("Detect from: {{.Content}}"))
	mock := &mockCompleter{
		responses: []string{`{"source": "spring-boot-3", "target": "spring-boot-4", "language": "java"}`},
	}

	meta, err := DetectMetadata(context.Background(), mock, tmpl, "migrate spring boot 3 to 4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Source != "spring-boot-3" {
		t.Errorf("source = %q, want spring-boot-3", meta.Source)
	}
	if meta.Target != "spring-boot-4" {
		t.Errorf("target = %q, want spring-boot-4", meta.Target)
	}
	if meta.Language != "java" {
		t.Errorf("language = %q, want java", meta.Language)
	}
}

func TestDetectMetadata_JSONEmbeddedInText(t *testing.T) {
	tmpl := template.Must(template.New("detect").Parse("Detect: {{.Content}}"))
	mock := &mockCompleter{
		responses: []string{`Here is my answer: {"source": "java-ee", "target": "quarkus", "language": "java"} done`},
	}

	meta, err := DetectMetadata(context.Background(), mock, tmpl, "some content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Source != "java-ee" {
		t.Errorf("source = %q, want java-ee", meta.Source)
	}
}

func TestDetectMetadata_LLMError(t *testing.T) {
	tmpl := template.Must(template.New("detect").Parse("Detect: {{.Content}}"))
	_, err := DetectMetadata(context.Background(), &errCompleter{msg: "network timeout"}, tmpl, "content")
	if err == nil {
		t.Error("expected error when LLM fails")
	}
	if !strings.Contains(err.Error(), "network timeout") {
		t.Errorf("error should mention 'network timeout', got: %v", err)
	}
}

func TestDetectMetadata_InvalidJSON(t *testing.T) {
	tmpl := template.Must(template.New("detect").Parse("Detect: {{.Content}}"))
	mock := &mockCompleter{responses: []string{"not json at all"}}

	_, err := DetectMetadata(context.Background(), mock, tmpl, "content")
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestDetectMetadata_MissingSourceOrTarget(t *testing.T) {
	tmpl := template.Must(template.New("detect").Parse("Detect: {{.Content}}"))
	mock := &mockCompleter{responses: []string{`{"source": "", "target": "", "language": "java"}`}}

	_, err := DetectMetadata(context.Background(), mock, tmpl, "content")
	if err == nil {
		t.Error("expected error when source/target are empty")
	}
}

func TestDetectMetadata_TruncatesLongContent(t *testing.T) {
	tmpl := template.Must(template.New("detect").Parse("{{.Content}}"))
	mock := &mockCompleter{
		responses: []string{`{"source": "a", "target": "b", "language": "go"}`},
	}

	// Content longer than 4000 chars should be truncated before sending
	longContent := strings.Repeat("x", 5000)
	_, err := DetectMetadata(context.Background(), mock, tmpl, longContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The prompt sent to the LLM should be the template rendered with the truncated content (4000 chars)
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 LLM call, got %d", len(mock.calls))
	}
	// Template is just "{{.Content}}", so the call should be exactly 4000 chars
	if len(mock.calls[0]) != 4000 {
		t.Errorf("LLM prompt length = %d, want 4000 (content truncated)", len(mock.calls[0]))
	}
}
