package extraction

import (
	"context"
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
